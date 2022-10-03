package packets

import (
	"bufio"
	"errors"
	"io"

	"github.com/Shryder/gnano/types"
)

type PacketReader struct {
	Buffer *bufio.Reader
}

func (reader *PacketReader) ReadAddress() (*types.Address, error) {
	address_bytes := make([]byte, 32)
	_, err := io.ReadFull(reader, address_bytes)
	if err != nil {
		return nil, err
	}

	var address types.Address
	copy(address[:], address_bytes)

	return &address, nil
}

func (reader *PacketReader) ReadSignature() (*types.Signature, error) {
	signature_bytes := make([]byte, 64)
	_, err := io.ReadFull(reader, signature_bytes)
	if err != nil {
		return nil, err
	}

	var signature types.Signature
	copy(signature[:], signature_bytes)

	return &signature, nil
}

func (reader *PacketReader) ReadHash() (*types.Hash, error) {
	hash_bytes := make([]byte, 32)
	_, err := io.ReadFull(reader, hash_bytes)
	if err != nil {
		return nil, err
	}

	var hash types.Hash
	copy(hash[:], hash_bytes)

	return &hash, nil
}

func (reader *PacketReader) ReadBlock(blockType BlockType) (*types.Block, error) {
	block_data := make([]byte, blockType.Size())
	_, err := io.ReadFull(reader, block_data)
	if err != nil {
		return nil, err
	}

	switch blockType {
	case BLOCK_TYPE_OPEN:
		return ParseOpenBlock(block_data), nil
	case BLOCK_TYPE_STATE:
		return ParseStateBlock(block_data), nil
	case BLOCK_TYPE_CHANGE:
		return ParseChangeBlock(block_data), nil
	case BLOCK_TYPE_SEND:
		return ParseSendBlock(block_data), nil
	case BLOCK_TYPE_RECEIVE:
		return ParseReceiveBlock(block_data), nil
	}

	return nil, errors.New("Can't parse this block type")
}

func (reader PacketReader) Read(p []byte) (int, error) {
	return reader.Buffer.Read(p)
}
