// Package shim is the Scheme A local fake-login helper.
//
// Phase 1 (this PR): listen on 127.0.0.1, send CMS079 Hello, log client bytes.
// Phase 2 (TODO): MapleAESOFB + minimal login/world/char packets + encrypted SERVER_IP
// so the stock client thinks char-select finished and reconnects to the real channel
// where LoginServer.putLoginAuth already waits.
package shim

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// Config describes the real channel handoff already registered via putLoginAuth.
type Config struct {
	ChannelHost string
	ChannelPort int
	CharID      int
	AuthIP      string
	Log         io.Writer
}

// Server is a short-lived local listener.
type Server struct {
	cfg      Config
	ln       net.Listener
	mu       sync.Mutex
	closed   bool
	plainSIP []byte // prebuilt plaintext SERVER_IP for Phase 2
}

// Start listens on 127.0.0.1:0 and accepts one (or a few) client connections.
func Start(cfg Config) (*Server, error) {
	if cfg.Log == nil {
		cfg.Log = os.Stderr
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
	fmt.Fprintf(cfg.Log, "shim listening on %s → will eventually redirect to %s:%d charId=%d (authIp=%s)\n",
		ln.Addr().String(), cfg.ChannelHost, cfg.ChannelPort, cfg.CharID, cfg.AuthIP)
	fmt.Fprintf(cfg.Log, "shim Phase1: Hello + log only. TODOs: MapleAESOFB IV/crypt, fake LOGIN_STATUS/SERVERLIST/CHARLIST, encrypt SERVER_IP 0x0B (%d plaintext bytes ready)\n", len(sip))
	return s, nil
}

// Addr returns the listen address (host:port).
func (s *Server) Addr() string {
	return s.ln.Addr().String()
}

// Port returns the TCP port.
func (s *Server) Port() int {
	a := s.ln.Addr().(*net.TCPAddr)
	return a.Port
}

// Close stops the listener.
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
	_ = conn.SetDeadline(time.Now().Add(2 * time.Minute))

	var sendIV, recvIV [4]byte
	// Match MapleServerHandler sessionOpened seed style (partially random last byte).
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

	// TODO(Phase2): initialize MapleAESOFB
	//   sendCrypt := AES(sendIV, 65535-79)  // server→client
	//   recvCrypt := AES(recvIV, 79)        // client→server
	// then read 4-byte headers, decrypt payloads, respond to login opcodes,
	// finally: header+encrypt(BuildServerIPPlain(...)) and close.

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			fmt.Fprintf(s.cfg.Log, "shim: recv %d bytes from client (encrypted; need MapleAESOFB to parse): %x\n", n, buf[:min(n, 64)])
			_ = s.plainSIP // keep referenced for Phase 2 wiring
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
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
