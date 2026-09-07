package shim

import (
	"encoding/binary"
	"net"
)

// CMS079 / RoyMS sendops: SERVER_IP = 0x0B (from sendops.properties).
const OpcodeServerIP uint16 = 0x0B

// MapleVersion matches ServerConstants.MAPLE_VERSION on feat/login-bridge.
const MapleVersion uint16 = 79

// BuildHello builds the unencrypted handshake packet (LoginPacket.getHello).
// Layout: u16 length=13 | u16 version | 2 zero bytes | recvIV[4] | sendIV[4] | locale=4
// Call with the same sendIV/recvIV you will use for MapleAESOFB (server→client / client→server).
func BuildHello(sendIV, recvIV [4]byte) []byte {
	payload := make([]byte, 0, 15)
	// length of body after this short
	payload = append(payload, 13, 0) // little-endian 13
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, MapleVersion)
	payload = append(payload, buf...)
	payload = append(payload, 0, 0) // patch stub used by this CMS079 fork
	payload = append(payload, recvIV[:]...)
	payload = append(payload, sendIV[:]...)
	payload = append(payload, 4) // locale
	return payload
}

// BuildServerIPPlain builds the plaintext SERVER_IP body+opcode matching
// MaplePacketCreator.getServerIP. Must be MapleAES-encrypted + 4-byte header
// before sending on a live session (Phase 2 — see TODOs in shim.go).
func BuildServerIPPlain(host string, port int, charID int) ([]byte, error) {
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, &net.AddrError{Err: "no IPs", Addr: host}
		}
		ip = ips[0]
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, &net.AddrError{Err: "not IPv4", Addr: host}
	}
	out := make([]byte, 0, 19)
	op := make([]byte, 2)
	binary.LittleEndian.PutUint16(op, OpcodeServerIP)
	out = append(out, op...)
	out = append(out, 0, 0) // short 0
	out = append(out, ip4...)
	p := make([]byte, 2)
	binary.LittleEndian.PutUint16(p, uint16(port))
	out = append(out, p...)
	c := make([]byte, 4)
	binary.LittleEndian.PutUint32(c, uint32(charID))
	out = append(out, c...)
	out = append(out, 1, 0, 0, 0, 0)
	return out, nil
}
