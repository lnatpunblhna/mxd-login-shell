// Package maplecrypto ports CMS079 MapleAESOFB + MapleCustomEncryption from
// MapleStory (tools/MapleAESOFB.java, MapleCustomEncryption.java, BitTools.java).
// Do not invent constants — keep funnyBytes / AES key / IV math identical to Java.
package maplecrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

// Classic Maple AES key (32 bytes with zero padding between meaningful bytes).
// Java: new SecretKeySpec({19,0,0,0, 8,0,0,0, 6,0,0,0, -76,...}, "AES")
var mapleAESKey = []byte{
	19, 0, 0, 0,
	8, 0, 0, 0,
	6, 0, 0, 0,
	0xB4, 0, 0, 0, // -76
	27, 0, 0, 0,
	15, 0, 0, 0,
	51, 0, 0, 0,
	82, 0, 0, 0,
}

// AESOFB is MapleAESOFB: AES-ECB OFB stream + funnyBytes IV rollover.
type AESOFB struct {
	iv           []byte
	block        cipher.Block
	mapleVersion uint16 // already byte-swapped like Java ctor
}

// NewAESOFB mirrors MapleAESOFB(byte[] iv, short mapleVersion).
// mapleVersion is the game version BEFORE the ctor's byte swap
// (e.g. 79 for recv, 65535-79 for send).
func NewAESOFB(iv []byte, mapleVersion uint16) (*AESOFB, error) {
	if len(iv) != 4 {
		return nil, fmt.Errorf("maplecrypto: IV must be 4 bytes, got %d", len(iv))
	}
	block, err := aes.NewCipher(mapleAESKey)
	if err != nil {
		return nil, err
	}
	ivCopy := make([]byte, 4)
	copy(ivCopy, iv)
	swapped := uint16((mapleVersion>>8)&0xFF) | uint16((mapleVersion<<8)&0xFF00)
	return &AESOFB{iv: ivCopy, block: block, mapleVersion: swapped}, nil
}

// IV returns a copy of the current IV.
func (a *AESOFB) IV() []byte {
	out := make([]byte, 4)
	copy(out, a.iv)
	return out
}

// Crypt XORs data in place with the OFB stream and advances the IV (same as Java crypt).
func (a *AESOFB) Crypt(data []byte) {
	remaining := len(data)
	llength := 1456
	start := 0
	for remaining > 0 {
		myIv := multiplyBytes(a.iv, 4, 4)
		if remaining < llength {
			llength = remaining
		}
		for x := start; x < start+llength; x++ {
			if (x-start)%len(myIv) == 0 {
				newIv := make([]byte, len(myIv))
				a.block.Encrypt(newIv, myIv)
				copy(myIv, newIv)
			}
			data[x] ^= myIv[(x-start)%len(myIv)]
		}
		start += llength
		remaining -= llength
		llength = 1460
	}
	a.iv = GetNewIv(a.iv)
}

// GetPacketHeader builds the 4-byte Maple packet header for a plaintext length.
func (a *AESOFB) GetPacketHeader(length int) []byte {
	iiv := int(a.iv[3]&0xFF) | int(int(a.iv[2])<<8&0xFF00)
	iiv ^= int(a.mapleVersion)
	mlength := ((length << 8 & 0xFF00) | (length >> 8)) ^ iiv
	return []byte{
		byte(iiv >> 8 & 0xFF),
		byte(iiv & 0xFF),
		byte(mlength >> 8 & 0xFF),
		byte(mlength & 0xFF),
	}
}

// CheckPacket validates the first two header bytes against the current IV / version.
func (a *AESOFB) CheckPacket(header []byte) bool {
	if len(header) < 2 {
		return false
	}
	return ((header[0]^a.iv[2])&0xFF) == byte(a.mapleVersion>>8&0xFF) &&
		((header[1]^a.iv[3])&0xFF) == byte(a.mapleVersion&0xFF)
}

// CheckPacketHeader interprets a big-endian int32 header like Java checkPacket(int).
func (a *AESOFB) CheckPacketHeader(packetHeader uint32) bool {
	return a.CheckPacket([]byte{
		byte(packetHeader >> 24),
		byte(packetHeader >> 16),
	})
}

// GetPacketLength extracts payload length from a 4-byte header (big-endian int layout).
func GetPacketLength(packetHeader uint32) int {
	packetLength := int(packetHeader>>16) ^ int(packetHeader&0xFFFF)
	packetLength = ((packetLength << 8 & 0xFF00) | (packetLength >> 8 & 0xFF))
	return packetLength
}

// GetPacketLengthBytes reads length from raw 4 header bytes (network / Java int order).
func GetPacketLengthBytes(hdr []byte) int {
	if len(hdr) < 4 {
		return 0
	}
	packetHeader := uint32(hdr[0])<<24 | uint32(hdr[1])<<16 | uint32(hdr[2])<<8 | uint32(hdr[3])
	return GetPacketLength(packetHeader)
}

// GetNewIv ports MapleAESOFB.getNewIv.
func GetNewIv(oldIv []byte) []byte {
	in := []byte{0xF2, 0x53, 0x50, 0xC6} // {-14, 83, 80, -58}
	for x := 0; x < 4; x++ {
		funnyShit(oldIv[x], in)
	}
	return in
}

func funnyShit(inputByte byte, in []byte) {
	elina := in[1]
	anna := inputByte
	moritz := funnyBytes[elina]
	moritz = byte(int8(moritz) - int8(inputByte))
	in[0] = byte(int8(in[0]) + int8(moritz))
	moritz = in[2]
	moritz ^= funnyBytes[anna]
	elina = byte(int8(elina) - int8(moritz))
	in[1] = elina
	elina = in[3]
	moritz = in[3]
	elina = byte(int8(elina) - int8(in[0]))
	moritz = funnyBytes[moritz]
	moritz = byte(int8(moritz) + int8(inputByte))
	moritz ^= in[2]
	in[2] = moritz
	elina = byte(int8(elina) + int8(funnyBytes[anna]))
	in[3] = elina

	// Match Java int (32-bit) merry assembly with signed byte shifts.
	var merry int32
	merry = int32(in[0]) & 0xFF
	merry |= (int32(int8(in[1])) << 8) & 0xFF00
	merry |= (int32(int8(in[2])) << 16) & 0xFF0000
	merry |= int32((uint32(int32(int8(in[3]))) << 24) & 0xFF000000)
	ret := int32(uint32(merry) >> 29) // >>>
	merry <<= 3
	ret |= merry
	in[0] = byte(ret)
	in[1] = byte(ret >> 8)
	in[2] = byte(ret >> 16)
	in[3] = byte(ret >> 24)
}

func multiplyBytes(in []byte, count, mul int) []byte {
	ret := make([]byte, count*mul)
	for x := 0; x < count*mul; x++ {
		ret[x] = in[x%count]
	}
	return ret
}
