package types

import (
	"encoding/base32"
	"encoding/hex"

	"github.com/shryder/ed25519-blake2b"
	"golang.org/x/crypto/blake2b"
)

var b32encoding = base32.NewEncoding("13456789abcdefghijkmnopqrstuwxyz")

type Address [32]byte

func (address *Address) ToHexString() string {
	return hex.EncodeToString(address[:])
}

func checksum(pubkey []byte) (checksum []byte, err error) {
	hash, err := blake2b.New(5, nil)
	if err != nil {
		return
	}
	hash.Write(pubkey)
	for _, b := range hash.Sum(nil) {
		checksum = append([]byte{b}, checksum...)
	}
	return
}

func (address *Address) ToNanoAddress() string {
	checksum, err := checksum((*address)[:])
	if err != nil {
		return ""
	}

	pubkey := append([]byte{0, 0, 0}, (*address)[:]...)

	return "nano_" + b32encoding.EncodeToString(pubkey)[4:] + b32encoding.EncodeToString(checksum)
}

func (address *Address) ToNodeAddress() string {
	checksum, err := checksum((*address)[:])
	if err != nil {
		return ""
	}

	pubkey := append([]byte{0, 0, 0}, (*address)[:]...)

	return "node_" + b32encoding.EncodeToString(pubkey)[4:] + b32encoding.EncodeToString(checksum)
}

func (address *Address) ToPublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(address[:])
}
