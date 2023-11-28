package random

import (
	cryptrand "crypto/rand"
	"log"
	"math/rand"
)

func String(n int, characters []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

func Bytes(n int) []byte {
	b := make([]byte, n)
	if _, err := cryptrand.Read(b); err != nil {
		log.Fatalf("failed to generate random bytes: %v", err)
	}
	return b
}
