package shim

import (
	"encoding/binary"
	"testing"
)

func TestBuildHelloLength(t *testing.T) {
	var s, r [4]byte
	s = [4]byte{1, 2, 3, 4}
	r = [4]byte{5, 6, 7, 8}
	h := BuildHello(s, r)
	if len(h) != 15 {
		t.Fatalf("hello len=%d want 15", len(h))
	}
	if binary.LittleEndian.Uint16(h[0:2]) != 13 {
		t.Fatalf("length prefix %d", binary.LittleEndian.Uint16(h[0:2]))
	}
	if binary.LittleEndian.Uint16(h[2:4]) != MapleVersion {
		t.Fatalf("version")
	}
	if h[14] != 4 {
		t.Fatalf("locale")
	}
}

func TestBuildServerIPPlain(t *testing.T) {
	p, err := BuildServerIPPlain("127.0.0.1", 8585, 42)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(p[0:2]) != OpcodeServerIP {
		t.Fatalf("opcode")
	}
	if p[4] != 127 || p[5] != 0 || p[6] != 0 || p[7] != 1 {
		t.Fatalf("ip bytes %v", p[4:8])
	}
	if binary.LittleEndian.Uint16(p[8:10]) != 8585 {
		t.Fatalf("port")
	}
	if binary.LittleEndian.Uint32(p[10:14]) != 42 {
		t.Fatalf("charId")
	}
}
