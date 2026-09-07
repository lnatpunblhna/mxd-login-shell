package shim

import (
	"encoding/binary"
	"strconv"
	"net"
	"time"
)

// CMS079 / RoyMS sendops (sendops.properties).
const (
	OpcodeLoginStatus  uint16 = 0x00
	OpcodeServerStatus uint16 = 0x06
	OpcodeServerList   uint16 = 0x09
	OpcodeCharList     uint16 = 0x0A
	OpcodeServerIP     uint16 = 0x0B
	OpcodePing         uint16 = 0x14
)

// CMS079 recvops (recvops.properties) — client → server.
const (
	RecvLoginPassword       uint16 = 0x01
	RecvServerListRequest   uint16 = 0x02
	RecvLicenseRequest      uint16 = 0x03
	RecvSetGender           uint16 = 0x04
	RecvServerStatusRequest uint16 = 0x05
	RecvCharListRequest     uint16 = 0x09
	RecvCharSelect          uint16 = 0x0A
	RecvPong                uint16 = 0x13
)

// MapleVersion matches ServerConstants.MAPLE_VERSION on feat/login-bridge.
const MapleVersion uint16 = 79

// BuildHello builds the unencrypted handshake packet (LoginPacket.getHello).
// Layout: u16 length=13 | u16 version | 2 zero bytes | recvIV[4] | sendIV[4] | locale=4
// Call with the same sendIV/recvIV you will use for MapleAESOFB (server→client / client→server).
func BuildHello(sendIV, recvIV [4]byte) []byte {
	payload := make([]byte, 0, 15)
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
// before sending on a live session.
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

func writeMapleString(out []byte, s string) []byte {
	b := []byte(s)
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(len(b)))
	out = append(out, buf...)
	out = append(out, b...)
	return out
}

func writeAsciiFixed(out []byte, s string, n int) []byte {
	buf := make([]byte, n)
	copy(buf, []byte(s))
	return append(out, buf...)
}

func mapleFileTime(ms int64) uint64 {
	// PacketHelper.getTime: (ms/1000)*10000000 + 116444592000000000
	return uint64(ms/1000)*10000000 + 116444592000000000
}

// BuildLoginStatusSuccess ports LoginPacket.getAuthSuccessRequest (minimal fields).
func BuildLoginStatusSuccess(accountID int, gender byte, gm bool, accountName string) []byte {
	out := make([]byte, 0, 64)
	op := make([]byte, 2)
	binary.LittleEndian.PutUint16(op, OpcodeLoginStatus)
	out = append(out, op...)
	out = append(out, 0) // success
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, uint32(accountID))
	out = append(out, b4...)
	out = append(out, gender)
	gmShort := make([]byte, 2)
	if gm {
		binary.LittleEndian.PutUint16(gmShort, 1)
	}
	out = append(out, gmShort...)
	out = writeMapleString(out, accountName)
	// HexTool.getByteArrayFromHexString("00 00 00 03 01 00 00 00 E2 ED A3 7A FA C9 01")
	out = append(out, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x00, 0x00, 0xE2, 0xED, 0xA3, 0x7A, 0xFA, 0xC9, 0x01)
	out = append(out, 0, 0, 0, 0) // int 0
	out = append(out, 0, 0, 0, 0, 0, 0, 0, 0) // long 0
	out = writeMapleString(out, strconv.Itoa(accountID))
	out = writeMapleString(out, accountName)
	out = append(out, 1)
	return out
}


// BuildServerListOneWorld is a minimal LoginPacket.getServerList for world 0 + one channel.
func BuildServerListOneWorld(worldID byte, worldName string, channelCount byte) []byte {
	if channelCount == 0 {
		channelCount = 1
	}
	out := make([]byte, 0, 128)
	op := make([]byte, 2)
	binary.LittleEndian.PutUint16(op, OpcodeServerList)
	out = append(out, op...)
	out = append(out, worldID)
	out = writeMapleString(out, worldName)
	out = append(out, 0) // flag
	out = writeMapleString(out, "") // event message
	out = append(out, 100, 0, 100, 0) // two shorts 100
	out = append(out, channelCount)
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, 500)
	out = append(out, b4...)
	for j := byte(1); j <= channelCount; j++ {
		out = writeMapleString(out, worldName+"-"+strconv.Itoa(int(j)))
		binary.LittleEndian.PutUint32(b4, 100)
		out = append(out, b4...)
		out = append(out, worldID)
		ch := make([]byte, 2)
		binary.LittleEndian.PutUint16(ch, uint16(j-1))
		out = append(out, ch...)
	}
	out = append(out, 0, 0) // balloons short 0
	return out
}

// BuildEndOfServerList ports LoginPacket.getEndOfServerList.
func BuildEndOfServerList() []byte {
	out := make([]byte, 3)
	binary.LittleEndian.PutUint16(out[0:2], OpcodeServerList)
	out[2] = 0xFF
	return out
}

// BuildServerStatus ports LoginPacket.getServerStatus.
func BuildServerStatus(status uint16) []byte {
	out := make([]byte, 4)
	binary.LittleEndian.PutUint16(out[0:2], OpcodeServerStatus)
	binary.LittleEndian.PutUint16(out[2:4], status)
	return out
}

// BuildPing ports sendops PING.
func BuildPing() []byte {
	out := make([]byte, 2)
	binary.LittleEndian.PutUint16(out, OpcodePing)
	return out
}

// FakeChar is enough identity for a CHARLIST stub the client can click.
type FakeChar struct {
	ID    int
	Name  string
	Level byte
	Job   uint16
	Gender byte
	Face  int
	Hair  int
	Skin  byte
	MapID int
}

// BuildCharListOne ports LoginPacket.getCharList with a single stub character
// (naked look: no equips). Stats use beginner defaults so the UI can render.
func BuildCharListOne(ch FakeChar, slots int) []byte {
	if ch.Face == 0 {
		ch.Face = 20100
	}
	if ch.Hair == 0 {
		ch.Hair = 30000
	}
	if ch.Level == 0 {
		ch.Level = 1
	}
	if ch.MapID == 0 {
		ch.MapID = 10000
	}
	if slots <= 0 {
		slots = 3
	}
	out := make([]byte, 0, 256)
	op := make([]byte, 2)
	binary.LittleEndian.PutUint16(op, OpcodeCharList)
	out = append(out, op...)
	out = append(out, 0)             // secondpw / status
	out = append(out, 0, 0, 0, 0)    // int 0
	out = append(out, 1)             // chars.size()
	out = appendCharEntry(out, ch)
	out = append(out, 3, 0) // short 3
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, uint32(slots))
	out = append(out, b4...)
	return out
}

func appendCharEntry(out []byte, ch FakeChar) []byte {
	// PacketHelper.addCharStats
	b4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(b4, uint32(ch.ID))
	out = append(out, b4...)
	out = writeAsciiFixed(out, ch.Name, 13)
	out = append(out, ch.Gender, ch.Skin)
	binary.LittleEndian.PutUint32(b4, uint32(ch.Face))
	out = append(out, b4...)
	binary.LittleEndian.PutUint32(b4, uint32(ch.Hair))
	out = append(out, b4...)
	out = append(out, make([]byte, 24)...) // pets / zeros
	out = append(out, ch.Level)
	job := make([]byte, 2)
	binary.LittleEndian.PutUint16(job, ch.Job)
	out = append(out, job...)
	// PlayerStats.connectData: str dex int luk hp maxhp mp maxmp (all short)
	stats := []uint16{4, 4, 4, 4, 50, 50, 50, 50}
	for _, s := range stats {
		binary.LittleEndian.PutUint16(job, s)
		out = append(out, job...)
	}
	out = append(out, 0, 0) // remainingAp
	out = append(out, 0, 0) // remainingSp
	out = append(out, 0, 0, 0, 0) // exp
	out = append(out, 0, 0)       // fame
	out = append(out, 0, 0, 0, 0) // gacha?
	ft := make([]byte, 8)
	binary.LittleEndian.PutUint64(ft, mapleFileTime(time.Now().UnixMilli()))
	out = append(out, ft...)
	binary.LittleEndian.PutUint32(b4, uint32(ch.MapID))
	out = append(out, b4...)
	out = append(out, 0) // spawnpoint

	// PacketHelper.addCharLook(..., mega=true, viewAll=false → channelserver false in LoginPacket.addCharEntry)
	// LoginPacket: addCharLook(mplew, chr, true, false) — mega=true, viewAll=false
	// addCharLook signature (mega, channelserver): write mega?0:1 → mega true → 0
	out = append(out, ch.Gender, ch.Skin)
	binary.LittleEndian.PutUint32(b4, uint32(ch.Face))
	out = append(out, b4...)
	out = append(out, 0) // mega ? 0 : 1
	binary.LittleEndian.PutUint32(b4, uint32(ch.Hair))
	out = append(out, b4...)
	out = append(out, 0xFF)       // end equips
	out = append(out, 0xFF)       // end masked
	out = append(out, 0, 0, 0, 0) // cash weapon
	out = append(out, 0, 0, 0, 0) // pet0
	out = append(out, 0, 0, 0, 0) // pet1
	out = append(out, 0, 0, 0, 0) // pet2

	out = append(out, 0) // ranking byte from addCharEntry
	return out
}
