package maplecrypto

// EncodeSend mirrors MaplePacketEncoder:
// header = sendAES.GetPacketHeader(len); Shanda.encrypt; AES.crypt; prepend header.
func EncodeSend(send *AESOFB, plaintext []byte) []byte {
	unenc := make([]byte, len(plaintext))
	copy(unenc, plaintext)
	hdr := send.GetPacketHeader(len(unenc))
	EncryptShanda(unenc)
	send.Crypt(unenc)
	out := make([]byte, 0, 4+len(unenc))
	out = append(out, hdr...)
	out = append(out, unenc...)
	return out
}

// DecodeRecv mirrors MaplePacketDecoder body path (after header length known):
// AES.crypt then Shanda.decrypt. Mutates a copy.
func DecodeRecv(recv *AESOFB, ciphertext []byte) []byte {
	dec := make([]byte, len(ciphertext))
	copy(dec, ciphertext)
	recv.Crypt(dec)
	DecryptShanda(dec)
	return dec
}
