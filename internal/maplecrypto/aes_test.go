package maplecrypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestGetNewIvDeterministic(t *testing.T) {
	old := []byte{82, 48, 120, 1}
	n1 := GetNewIv(old)
	n2 := GetNewIv(old)
	if !bytes.Equal(n1, n2) {
		t.Fatalf("GetNewIv not deterministic")
	}
	if bytes.Equal(n1, old) {
		t.Fatalf("IV should change")
	}
	if len(n1) != 4 {
		t.Fatalf("len")
	}
}

func TestCryptRoundTripSameKeystream(t *testing.T) {
	iv := []byte{70, 114, 122, 9}
	a, err := NewAESOFB(append([]byte(nil), iv...), 79)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewAESOFB(append([]byte(nil), iv...), 79)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("hello maple cms079 packet body!!")
	c1 := append([]byte(nil), plain...)
	c2 := append([]byte(nil), plain...)
	a.Crypt(c1)
	b.Crypt(c2)
	if !bytes.Equal(c1, c2) {
		t.Fatalf("crypt diverge\n%s\n%s", hex.EncodeToString(c1), hex.EncodeToString(c2))
	}
	// Same IV start + same Crypt = same OFB; applying twice with fresh AES restores
	a2, _ := NewAESOFB(append([]byte(nil), iv...), 79)
	restored := append([]byte(nil), c1...)
	a2.Crypt(restored)
	if !bytes.Equal(restored, plain) {
		t.Fatalf("OFB not involution on fresh twin: got %q", restored)
	}
}

func TestShandaRoundTrip(t *testing.T) {
	cases := [][]byte{
		{},
		{0x00, 0x01},
		{0x0B, 0x00, 0x00, 0x00, 127, 0, 0, 1, 0x85, 0x21, 42, 0, 0, 0, 1, 0, 0, 0, 0},
		bytes.Repeat([]byte{0xAB}, 64),
		bytes.Repeat([]byte{0x01}, 200),
	}
	for i, c := range cases {
		orig := append([]byte(nil), c...)
		buf := append([]byte(nil), c...)
		EncryptShanda(buf)
		if len(c) > 0 && bytes.Equal(buf, orig) {
			t.Fatalf("case %d: encrypt left plaintext", i)
		}
		DecryptShanda(buf)
		if !bytes.Equal(buf, orig) {
			t.Fatalf("case %d: roundtrip fail\nwant %s\ngot  %s", i, hex.EncodeToString(orig), hex.EncodeToString(buf))
		}
	}
}

func TestEncodeDecodePacket(t *testing.T) {
	sendIV := []byte{82, 48, 120, 3}
	recvIV := []byte{70, 114, 122, 4}
	// Server send crypt uses 65535-79; client would use sendIV as its recv — we simulate server side only.
	send, err := NewAESOFB(append([]byte(nil), sendIV...), 65535-79)
	if err != nil {
		t.Fatal(err)
	}
	// Peer that decrypts server packets uses the same IV sequence as send (client recv = server send IV)
	peerRecv, err := NewAESOFB(append([]byte(nil), sendIV...), 65535-79)
	if err != nil {
		t.Fatal(err)
	}
	_ = recvIV
	plain := []byte{0x0B, 0x00, 0x00, 0x00, 127, 0, 0, 1, 0x85, 0x21, 0x2A, 0, 0, 0, 1, 0, 0, 0, 0}
	wire := EncodeSend(send, plain)
	if len(wire) != 4+len(plain) {
		t.Fatalf("wire len %d", len(wire))
	}
	if !peerRecv.CheckPacket(wire[:4]) {
		t.Fatalf("header check failed hdr=%x iv=%x", wire[:4], peerRecv.IV())
	}
	body := DecodeRecv(peerRecv, wire[4:])
	if !bytes.Equal(body, plain) {
		t.Fatalf("decode\nwant %s\ngot  %s", hex.EncodeToString(plain), hex.EncodeToString(body))
	}
}

func TestPacketLengthRoundTrip(t *testing.T) {
	send, _ := NewAESOFB([]byte{1, 2, 3, 4}, 65535-79)
	for _, n := range []int{0, 1, 19, 100, 1456, 2000} {
		hdr := send.GetPacketHeader(n)
		// Advance IV was NOT done by GetPacketHeader — good.
		// Rebuild twin at same IV
		twin, _ := NewAESOFB([]byte{1, 2, 3, 4}, 65535-79)
		_ = twin
		ph := uint32(hdr[0])<<24 | uint32(hdr[1])<<16 | uint32(hdr[2])<<8 | uint32(hdr[3])
		got := GetPacketLength(ph)
		if got != n {
			t.Fatalf("len %d -> hdr %x -> %d", n, hdr, got)
		}
	}
}

func TestGetNewIvKnownVector(t *testing.T) {
	got := GetNewIv([]byte{82, 48, 120, 1})
	want := []byte{220, 171, 167, 177}
	if !bytes.Equal(got, want) {
		t.Fatalf("GetNewIv = %v want %v", got, want)
	}
	got0 := GetNewIv([]byte{0, 0, 0, 0})
	want0 := []byte{17, 187, 100, 199}
	if !bytes.Equal(got0, want0) {
		t.Fatalf("GetNewIv0 = %v want %v", got0, want0)
	}
}
