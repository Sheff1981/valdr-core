package storage

type fixedStorageRandomXHasher struct {
	hash []byte
}

func (h *fixedStorageRandomXHasher) Hash(key, input []byte) ([]byte, error) {
	return append([]byte(nil), h.hash...), nil
}

func lowStorageHash() []byte {
	out := make([]byte, 32)
	out[31] = 1
	return out
}
