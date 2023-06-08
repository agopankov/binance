package database

import (
	"time"
)

type Verification struct {
	Email        string
	Code         string
	FirstBotID   int64
	SecondBotID  int64
	LastVerified time.Time
}

type Database interface {
	SendVerificationEmail(emailAddress string, firstBotID int64, secondBotID int64, postmarkToken string)
	VerifyCode(emailAddress string, code string) bool
	ShouldSendVerificationEmail(emailAddress string) bool
	GetAllUsers() ([]Verification, error)
}
