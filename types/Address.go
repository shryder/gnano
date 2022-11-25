package types

import (
	"encoding/base32"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/shryder/ed25519-blake2b"
	"golang.org/x/crypto/blake2b"
)

var NanoEncoding = base32.NewEncoding("13456789abcdefghijkmnopqrstuwxyz")

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

	return "nano_" + NanoEncoding.EncodeToString(pubkey)[4:] + NanoEncoding.EncodeToString(checksum)
}

func (address *Address) ToNodeAddress() string {
	checksum, err := checksum((*address)[:])
	if err != nil {
		return ""
	}

	pubkey := append([]byte{0, 0, 0}, (*address)[:]...)

	return "node_" + NanoEncoding.EncodeToString(pubkey)[4:] + NanoEncoding.EncodeToString(checksum)
}

func (address *Address) ToPublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(address[:])
}

func (address *Address) MarshalJSON() ([]byte, error) {
	return []byte(`"` + address.ToHexString() + `"`), nil
}

func (address *Address) UnmarshalJSON(data []byte) error {
	address_slice, err := hex.DecodeString(strings.Trim(string(data), `"`))
	if err != nil {
		return err
	}

	if len(address_slice) != 32 {
		return errors.New("String representation of an address must be 32 bytes")
	}

	copy(address[:], address_slice)

	return nil
}

func DecodeNanoAddress(nano_address string) (addy *Address, err error) {
	if nano_address[:5] != "nano_" {
		return nil, errors.New("Invalid address format")
	}

	// A valid nano address is 64 bytes long
	// First 5 are simply a hard-coded string nano_ for ease of use
	// The following 52 characters form the address, and the final
	// 8 are a checksum.
	// They are base 32 encoded with a custom encoding.

	// Remove nano_ prefix
	nano_address = nano_address[5:]
	if len(nano_address) != 60 {
		return nil, errors.New("Invalid address size")
	}

	// The nano address string is 260bits which doesn't fall on a
	// byte boundary. pad with zeros to 280bits.
	// (zeros are encoded as 1 in nano's 32bit alphabet)
	key_b32nano := "1111" + nano_address[0:52]
	input_checksum := nano_address[52:]

	key_bytes, err := NanoEncoding.DecodeString(key_b32nano)
	if err != nil {
		return nil, err
	}
	// strip off upper 24 bits (3 bytes). 20 padding was added by us,
	// 4 is unused as account is 256 bits.
	key_bytes = key_bytes[3:]

	// nano checksum is calculated by hashing the key and reversing the bytes
	address_checksum, err := checksum(key_bytes)
	if err != nil {
		return nil, errors.New("Couldn't create checksum")
	}

	valid := NanoEncoding.EncodeToString(address_checksum) == input_checksum
	if !valid {
		return nil, errors.New("Invalid address checksum")
	}

	addy = new(Address)
	copy(addy[:], key_bytes)

	return addy, nil
}

func StringPublicKeyToAddress(public_key_str string) (*Address, error) {
	public_key_slice, err := hex.DecodeString(public_key_str)
	if err != nil {
		return nil, err
	}

	address := new(Address)
	copy(address[:], public_key_slice)

	return address, nil
}
