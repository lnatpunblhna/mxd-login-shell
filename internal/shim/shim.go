// Package shim is the Scheme A local fake-login helper.
//
// Phase 2: Hello → MapleAES+Shanda session → minimal CMS079 login/world/char
// responses → encrypted SERVER_IP (0x0B) so the stock client reconnects to the
// real channel where LoginServer.putLoginAuth already waits.
package shim

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/lnatpunblhna/mxd-login-shell/internal/maplecrypto"
)

// Config describes the real channel handoff already registered via putLoginAuth.
type Config struct {
	ChannelHost string
	ChannelPort int
	CharID      int
	AuthIP      string
	Log         io.Writer

	// Optional identity for LOGIN_STATUS / CHARLIST stubs (filled from Go shell).
	AccountID   int
	AccountName string
	Gender      byte
	GM          bool
	CharName    string
	CharLevel   byte
	CharJob     uint16
	WorldName   string
}

// Server is a short-lived local listener.
type Server struct {
	cfg      Config
	ln       net.Listener
	mu       sync.Mutex
	closed   bool
	plainSIP []byte
}

// Start listens on 127.0.0.1:0 and accepts client connections.
func Start(cfg Config) (*Server, error) {
	if cfg.Log == nil {
		cfg.Log = os.Stderr
	}
	if cfg.WorldName == "" {
		cfg.WorldName = "World0"
	}
	if cfg.AccountName == "" {
		cfg.AccountName = "player"
	}
	if cfg.CharName == "" {
		cfg.CharName = "Hero"
	}
	if cfg.CharLevel == 0 {
		cfg.CharLevel = 1
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	sip, err := BuildServerIPPlain(cfg.ChannelHost, cfg.ChannelPort, cfg.CharID)
	if err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("build SERVER_IP: %w", err)
	}
	s := &Server{cfg: cfg, ln: ln, plainSIP: sip}
	go s.serve()
	fmt.Fprintf(cfg.Log, "shim listening on %s → redirect to %s:%d charId=%d (authIp=%s)\n",
		ln.Addr().String(), cfg.ChannelHost, cfg.ChannelPort, cfg.CharID, cfg.AuthIP)
	fmt.Fprintf(cfg.Log, "shim Phase2: Hello + MapleAES/Shanda + fake login → encrypted SERVER_IP (%d plaintext bytes)\n", len(sip))
	return s, nil
}

func (s *Server) Addr() string { return s.ln.Addr().String() }

func (s *Server) Port() int {
	return s.ln.Addr().(*net.TCPAddr).Port
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.ln.Close()
}

func (s *Server) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			fmt.Fprintf(s.cfg.Log, "shim accept: %v\n", err)
			return
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	fmt.Fprintf(s.cfg.Log, "shim: client connected from %s\n", remote)
	_ = conn.SetDeadline(time.Now().Add(5 * time.Minute))

	var sendIV, recvIV [4]byte
	copy(sendIV[:], []byte{82, 48, 120, 0})
	copy(recvIV[:], []byte{70, 114, 122, 0})
	_, _ = rand.Read(sendIV[3:])
	_, _ = rand.Read(recvIV[3:])

	hello := BuildHello(sendIV, recvIV)
	if _, err := conn.Write(hello); err != nil {
		fmt.Fprintf(s.cfg.Log, "shim: write Hello: %v\n", err)
		return
	}
	fmt.Fprintf(s.cfg.Log, "shim: sent Hello (v%d) sendIV=%x recvIV=%x (%d bytes)\n", MapleVersion, sendIV, recvIV, len(hello))

	sendCrypt, err := maplecrypto.NewAESOFB(sendIV[:], 65535-MapleVersion)
	if err != nil {
		fmt.Fprintf(s.cfg.Log, "shim: send AES: %v\n", err)
		return
	}
	recvCrypt, err := maplecrypto.NewAESOFB(recvIV[:], MapleVersion)
	if err != nil {
		fmt.Fprintf(s.cfg.Log, "shim: recv AES: %v\n", err)
		return
	}

	sendPacket := func(plain []byte) error {
		wire := maplecrypto.EncodeSend(sendCrypt, plain)
		_, err := conn.Write(wire)
		return err
	}

	handedOff := false
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)
	for !handedOff {
		n, err := conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		for {
			if len(buf) < 4 {
				break
			}
			if !recvCrypt.CheckPacket(buf[:4]) {
				fmt.Fprintf(s.cfg.Log, "shim: bad packet header %x (iv=%x) — closing\n", buf[:4], recvCrypt.IV())
				return
			}
			bodyLen := maplecrypto.GetPacketLengthBytes(buf[:4])
			if bodyLen < 0 || bodyLen > 1<<20 {
				fmt.Fprintf(s.cfg.Log, "shim: absurd bodyLen=%d\n", bodyLen)
				return
			}
			if len(buf) < 4+bodyLen {
				break
			}
			hdr := buf[:4]
			cipherBody := append([]byte(nil), buf[4:4+bodyLen]...)
			buf = buf[4+bodyLen:]
			_ = hdr
			plain := maplecrypto.DecodeRecv(recvCrypt, cipherBody)
			if len(plain) < 2 {
				fmt.Fprintf(s.cfg.Log, "shim: short packet %x\n", plain)
				continue
			}
			op := binary.LittleEndian.Uint16(plain[0:2])
			fmt.Fprintf(s.cfg.Log, "shim: client opcode 0x%02X (%d bytes)\n", op, len(plain))

			switch op {
			case RecvPong:
				if err := sendPacket(BuildPing()); err != nil {
					fmt.Fprintf(s.cfg.Log, "shim: ping: %v\n", err)
					return
				}
			case RecvLoginPassword, RecvLicenseRequest, RecvSetGender:
				pkt := BuildLoginStatusSuccess(s.cfg.AccountID, s.cfg.Gender, s.cfg.GM, s.cfg.AccountName)
				if err := sendPacket(pkt); err != nil {
					fmt.Fprintf(s.cfg.Log, "shim: LOGIN_STATUS: %v\n", err)
					return
				}
				fmt.Fprintf(s.cfg.Log, "shim: sent LOGIN_STATUS success\n")
			case RecvServerListRequest:
				if err := sendPacket(BuildServerListOneWorld(0, s.cfg.WorldName, 1)); err != nil {
					fmt.Fprintf(s.cfg.Log, "shim: SERVERLIST: %v\n", err)
					return
				}
				if err := sendPacket(BuildEndOfServerList()); err != nil {
					fmt.Fprintf(s.cfg.Log, "shim: SERVERLIST end: %v\n", err)
					return
				}
				fmt.Fprintf(s.cfg.Log, "shim: sent SERVERLIST + end\n")
			case RecvServerStatusRequest:
				if err := sendPacket(BuildServerStatus(0)); err != nil {
					fmt.Fprintf(s.cfg.Log, "shim: SERVERSTATUS: %v\n", err)
					return
				}
			case RecvCharListRequest:
				ch := FakeChar{
					ID:     s.cfg.CharID,
					Name:   s.cfg.CharName,
					Level:  s.cfg.CharLevel,
					Job:    s.cfg.CharJob,
					Gender: s.cfg.Gender,
				}
				if err := sendPacket(BuildCharListOne(ch, 3)); err != nil {
					fmt.Fprintf(s.cfg.Log, "shim: CHARLIST: %v\n", err)
					return
				}
				fmt.Fprintf(s.cfg.Log, "shim: sent CHARLIST stub id=%d name=%s\n", ch.ID, ch.Name)
			case RecvCharSelect:
				if err := sendPacket(append([]byte(nil), s.plainSIP...)); err != nil {
					fmt.Fprintf(s.cfg.Log, "shim: SERVER_IP: %v\n", err)
					return
				}
				fmt.Fprintf(s.cfg.Log, "shim: sent encrypted SERVER_IP → %s:%d charId=%d — handoff done\n",
					s.cfg.ChannelHost, s.cfg.ChannelPort, s.cfg.CharID)
				handedOff = true
			default:
				fmt.Fprintf(s.cfg.Log, "shim: unhandled opcode 0x%02X payload=%x\n", op, plain[:min(len(plain), 32)])
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(s.cfg.Log, "shim: read: %v\n", err)
			} else {
				fmt.Fprintf(s.cfg.Log, "shim: client closed %s\n", remote)
			}
			return
		}
	}
	// Give the client a moment to read SERVER_IP before we drop the TCP session.
	time.Sleep(500 * time.Millisecond)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
