package random

import (
	"crypto/rand"
	"log"
	"math/big"
)

func String(n int, characters []rune) string {
	b := make([]rune, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(characters))))
		if err != nil {
			log.Fatalf("failed to generate random index: %v", err)
		}
		b[i] = characters[idx.Int64()]
	}
	return string(b)
}

func Bytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("failed to generate random bytes: %v", err)
	}
	return b
}
