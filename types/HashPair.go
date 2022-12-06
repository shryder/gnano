package types

type HashPair struct {
	Hash Hash
	Root Hash
}

func (pair *HashPair) FromSlice(b []byte) {
	copy(pair.Hash[:], b[0:32])
	copy(pair.Root[:], b[32:64])
}

func (pair *HashPair) ToSlice() []byte {
	return append(pair.Hash[:], pair.Root[:]...)
}
