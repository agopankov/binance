package emailverify

import (
	"math/rand"
	"time"
)

const (
	CharSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type Verification struct {
	Email        string
	Code         string
	FirstBotID   int64
	SecondBotID  int64
	LastVerified time.Time
}

func GenerateVerificationCode(length int) string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = CharSet[rand.Intn(len(CharSet))]
	}
	return string(b)
}
