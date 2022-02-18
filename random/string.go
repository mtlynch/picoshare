package random

import (
	"log"
	"math/rand"
	"time"
)

var characters = []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func init() {
	log.Printf("initialize random seed")
	rand.Seed(time.Now().UTC().UnixNano())
}

func String(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}
