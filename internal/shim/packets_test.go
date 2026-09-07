package shim

import (
	"encoding/binary"
	"testing"

	"github.com/lnatpunblhna/mxd-login-shell/internal/maplecrypto"
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

func TestLoginStatusHasOpcode(t *testing.T) {
	p := BuildLoginStatusSuccess(7, 0, false, "testacc")
	if binary.LittleEndian.Uint16(p[0:2]) != OpcodeLoginStatus {
		t.Fatalf("op")
	}
	if p[2] != 0 {
		t.Fatalf("status")
	}
}

func TestCharListOneRoundTripEncrypt(t *testing.T) {
	ch := FakeChar{ID: 42, Name: "Hero", Level: 10, Job: 100, Gender: 0}
	plain := BuildCharListOne(ch, 3)
	if binary.LittleEndian.Uint16(plain[0:2]) != OpcodeCharList {
		t.Fatalf("opcode")
	}
	send, err := maplecrypto.NewAESOFB([]byte{1, 2, 3, 4}, 65535-79)
	if err != nil {
		t.Fatal(err)
	}
	peer, err := maplecrypto.NewAESOFB([]byte{1, 2, 3, 4}, 65535-79)
	if err != nil {
		t.Fatal(err)
	}
	wire := maplecrypto.EncodeSend(send, plain)
	if !peer.CheckPacket(wire[:4]) {
		t.Fatalf("header")
	}
	body := maplecrypto.DecodeRecv(peer, wire[4:])
	if binary.LittleEndian.Uint16(body[0:2]) != OpcodeCharList {
		t.Fatalf("decoded op")
	}
}

func TestServerIPEncryptHeader(t *testing.T) {
	plain, err := BuildServerIPPlain("127.0.0.1", 8585, 99)
	if err != nil {
		t.Fatal(err)
	}
	send, _ := maplecrypto.NewAESOFB([]byte{82, 48, 120, 5}, 65535-79)
	peer, _ := maplecrypto.NewAESOFB([]byte{82, 48, 120, 5}, 65535-79)
	wire := maplecrypto.EncodeSend(send, plain)
	if len(wire) != 4+len(plain) {
		t.Fatalf("len")
	}
	if !peer.CheckPacket(wire[:4]) {
		t.Fatalf("check")
	}
	got := maplecrypto.DecodeRecv(peer, wire[4:])
	if binary.LittleEndian.Uint16(got[0:2]) != OpcodeServerIP {
		t.Fatalf("op")
	}
}
