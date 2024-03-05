package pkg

import (
	"math/rand"
	"time"
)

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxy")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func Random(seq int) string {
	rand.Seed(time.Now().UnixNano())
	return randSeq(seq)
}
