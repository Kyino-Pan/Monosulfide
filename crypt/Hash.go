package crypt

type Hash struct{ Hash string }

func (h Hash) Bytes() []byte {
	if h.Hash == "" {
		return nil
	}
	return []byte(h.Hash)
}

func (h Hash) String() string {
	return h.Hash
}

func NewHash(byteHash []byte) *Hash {
	return &Hash{
		Hash: string(byteHash),
	}
}

func EmptyHash() *Hash {
	return &Hash{Hash: ""}
}
