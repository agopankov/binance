package database

import (
	"github.com/aws/aws-sdk-go/aws/session"
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
	SendVerificationEmail(sess *session.Session, emailAddress string, firstBotID int64, secondBotID int64, postmarkToken string)
	VerifyCode(sess *session.Session, emailAddress string, code string) bool
	ShouldSendVerificationEmail(sess *session.Session, emailAddress string) bool
}
