package maplecrypto

// EncryptShanda ports MapleCustomEncryption.encryptData (in-place).
func EncryptShanda(data []byte) {
	for j := 0; j < 6; j++ {
		var remember byte
		dataLength := byte(len(data) & 0xFF)
		if j%2 == 0 {
			for i := 0; i < len(data); i++ {
				cur := data[i]
				cur = rollLeft(cur, 3)
				cur = byte(int8(cur) + int8(dataLength))
				remember ^= cur
				cur = remember
				cur = rollRight(cur, int(dataLength&0xFF))
				cur = byte(^cur & 0xFF)
				cur = byte(int8(cur) + 72)
				dataLength--
				data[i] = cur
			}
		} else {
			for i := len(data) - 1; i >= 0; i-- {
				cur := data[i]
				cur = rollLeft(cur, 4)
				cur = byte(int8(cur) + int8(dataLength))
				remember ^= cur
				cur = remember
				cur ^= 0x13
				cur = rollRight(cur, 3)
				dataLength--
				data[i] = cur
			}
		}
	}
}

// DecryptShanda ports MapleCustomEncryption.decryptData (in-place).
func DecryptShanda(data []byte) {
	for j := 1; j <= 6; j++ {
		var remember byte
		dataLength := byte(len(data) & 0xFF)
		var nextRemember byte
		if j%2 == 0 {
			for i := 0; i < len(data); i++ {
				cur := data[i]
				cur = byte(int8(cur) - 72)
				cur = byte(^cur & 0xFF)
				nextRemember = rollLeft(cur, int(dataLength&0xFF))
				cur = nextRemember
				cur ^= remember
				remember = nextRemember
				cur = byte(int8(cur) - int8(dataLength))
				cur = rollRight(cur, 3)
				data[i] = cur
				dataLength--
			}
		} else {
			for i := len(data) - 1; i >= 0; i-- {
				cur := data[i]
				cur = rollLeft(cur, 3)
				nextRemember = cur ^ 0x13
				cur = nextRemember
				cur ^= remember
				remember = nextRemember
				cur = byte(int8(cur) - int8(dataLength))
				cur = rollRight(cur, 4)
				data[i] = cur
				dataLength--
			}
		}
	}
}

func rollLeft(in byte, count int) byte {
	tmp := int(in) & 0xFF
	tmp <<= count % 8
	return byte((tmp & 0xFF) | (tmp >> 8))
}

func rollRight(in byte, count int) byte {
	tmp := int(in) & 0xFF
	tmp = (tmp << 8) >> (count % 8)
	return byte((tmp & 0xFF) | (tmp >> 8))
}
