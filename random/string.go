package random

import (
	"log"
	"math/rand"
	"time"
)

func init() {
	log.Printf("initializing random seed")
	rand.Seed(time.Now().UTC().UnixNano())
}

func String(n int, characters []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}
