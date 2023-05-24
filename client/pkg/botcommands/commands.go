package botcommands

import (
	"context"
	"fmt"
	"github.com/agopankov/imPulse/client/pkg/cancelfuncs"
	"github.com/agopankov/imPulse/client/pkg/emailverify"
	"github.com/agopankov/imPulse/client/pkg/monitor"
	"github.com/agopankov/imPulse/client/pkg/telegram"
	"github.com/agopankov/imPulse/client/pkg/tracker"
	"github.com/agopankov/imPulse/client/pkg/user"
	"github.com/agopankov/imPulse/server/pkg/grpcbinance/proto"
	"github.com/aws/aws-sdk-go/aws/session"
	tele "gopkg.in/telebot.v3"
	"log"
	"net/mail"
	"strconv"
	"time"
)

func StartCommandHandlerFirstClient(m *tele.Message, telegramClient *telegram.Client, usr *user.User) {
	log.Printf("Received /start command from chat ID %d", m.Sender.ID)
	usr.SetState(user.StateAwaitingEmail)
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	if _, err := telegramClient.SendMessage(recipient, "Please enter your email address for verification"); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func StartCommandHandlerSecondClient(m *tele.Message, secondTelegramClient *telegram.Client, usr *user.User) {
	log.Printf("Received /start command from second chat ID %d", m.Sender.ID)
	usr.SetSecondChatID(m.Sender.ID)
	secondChatID := m.Sender.ID
	recipient := &tele.User{ID: secondChatID}
	if _, err := secondTelegramClient.SendMessage(recipient, "The service for monitoring coins that are being pumped has been launched"); err != nil {
		log.Printf("Error sending message to second chat: %v", err)
	} else {
		log.Printf("Sent message to second chat ID %d: %s", secondChatID, "Hi")
	}
}

func StopCommandHandler(m *tele.Message, cancelFuncs *cancelfuncs.CancelFuncs) {
	log.Printf("Received /stop command from chat ID %d", m.Sender.ID)
	chatID := m.Sender.ID
	cancelFuncs.Remove(chatID)
}

func Change24PercentCommandHandler(m *tele.Message, telegramClient *telegram.Client, usr *user.User) {
	usr.SetState(user.StateAwaitingPercent)
	currentPercent24 := usr.ChangePercent24.GetPercent()
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	msg := fmt.Sprintf("Please enter the new percent value (current value is %.2f)", currentPercent24)
	if _, err := telegramClient.SendMessage(recipient, msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func SetWaitTimeCommandHandler(m *tele.Message, secondTelegramClient *telegram.Client, usr *user.User) {
	usr.SetState(user.StateAwaitingWaitTime)
	currentWaitTime := usr.PumpSettings.GetWaitTime()
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	msg := fmt.Sprintf("Please enter the new wait time in minutes (current wait time is %s)", currentWaitTime)
	if _, err := secondTelegramClient.SendMessage(recipient, msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func SetPumpPercentCommandHandler(m *tele.Message, secondTelegramClient *telegram.Client, usr *user.User) {
	usr.SetState(user.StateAwaitingPercent)
	currentPumpPercent := usr.PumpSettings.GetPumpPercent()
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	msg := fmt.Sprintf("Please enter the new percent value (current percent is %.2f)", currentPumpPercent)
	if _, err := secondTelegramClient.SendMessage(recipient, msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func MessageHandlerFirstClient(m *tele.Message, telegramClient *telegram.Client, secondTelegramClient *telegram.Client, cancelFuncs *cancelfuncs.CancelFuncs, usr *user.User, binanceClient proto.BinanceServiceClient) {
	switch usr.GetState() {
	case user.StateAwaitingEmail:
		email := m.Text
		_, err := mail.ParseAddress(email)
		if err != nil {
			log.Printf("Invalid email value: %v", err)
			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}
			if _, err := telegramClient.SendMessage(recipient, "Invalid email value, please enter a valid email"); err != nil {
				log.Printf("Error sending message: %v", err)
			}
			return
		}

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		if !emailverify.ShouldSendVerificationEmail(sess, email) {
			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}

			trackerInstance := tracker.NewTracker()

			ctx, cancel := context.WithCancel(context.Background())
			cancelFuncs.Add(chatID, cancel)

			go monitor.PriceChanges(ctx, telegramClient, secondTelegramClient, binanceClient, usr, trackerInstance)

			if _, err := telegramClient.SendMessage(recipient, "Tracking service launched"); err != nil {
				log.Printf("Error sending message: %v", err)
			} else {
				log.Printf("Sent message to chat ID %d: %s", chatID, "Hi")
			}
			return
		} else {
			chatID := m.Sender.ID

			usr.SetEmail(email)
			emailverify.SendVerificationEmail(sess, email, usr.FirstChatID, usr.SecondChatID)

			recipient := &tele.User{ID: chatID}
			if _, err := telegramClient.SendMessage(recipient, "A verification code has been sent to your email. Please enter it."); err != nil {
				log.Printf("Error sending message: %v", err)
			}

			usr.SetState(user.StateAwaitingVerification)
		}

	case user.StateAwaitingVerification:
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		if emailverify.VerifyCode(sess, usr.GetEmail(), m.Text) {
			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}
			usr.SetState(user.StateNone)

			trackerInstance := tracker.NewTracker()

			ctx, cancel := context.WithCancel(context.Background())
			cancelFuncs.Add(chatID, cancel)

			go monitor.PriceChanges(ctx, telegramClient, secondTelegramClient, binanceClient, usr, trackerInstance)

			if _, err := telegramClient.SendMessage(recipient, "Tracking service launched"); err != nil {
				log.Printf("Error sending message: %v", err)
			} else {
				log.Printf("Sent message to chat ID %d: %s", chatID, "Hi")
			}

		} else {
			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}
			if _, err := telegramClient.SendMessage(recipient, "Verification failed. Please enter the correct verification code."); err != nil {
				log.Printf("Error sending message: %v", err)
			}
		}

	case user.StateAwaitingPercent:
		newPercent, err := strconv.ParseFloat(m.Text, 64)
		if err != nil {
			log.Printf("Invalid percent value: %v", err)

			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}
			if _, err := telegramClient.SendMessage(recipient, "Invalid percent value, please enter a valid number"); err != nil {
				log.Printf("Error sending message: %v", err)
			}
			return
		}
		usr.ChangePercent24.SetPercent(newPercent)
		log.Printf("Percent changed to %f", newPercent)

		usr.SetState(user.StateNone)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "The percentage of pumping for tracked coins has been changed"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "The percentage of pumping for tracked coins has been changed")
		}
	}
}

func MessageHandlerSecondClient(m *tele.Message, secondTelegramClient *telegram.Client, usr *user.User) {
	switch usr.GetState() {
	case user.StateAwaitingPercent:
		pumpPercent, err := strconv.ParseFloat(m.Text, 64)
		if err != nil {
			log.Printf("Invalid percent value: %v", err)

			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}
			if _, err := secondTelegramClient.SendMessage(recipient, "Invalid percent value, please enter a valid number"); err != nil {
				log.Printf("Error sending message: %v", err)
			}
			return
		}
		usr.PumpSettings.SetPumpPercent(pumpPercent)
		log.Printf("Percent of pump changed to %f", pumpPercent)

		usr.SetState(user.StateNone)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := secondTelegramClient.SendMessage(recipient, "The percentage expected for the pump has been changed"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "The percentage expected for the pump has been changed")
		}

	case user.StateAwaitingWaitTime:
		waitTime, err := strconv.Atoi(m.Text)
		if err != nil {
			log.Printf("Invalid wait time value: %v", err)

			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}
			if _, err := secondTelegramClient.SendMessage(recipient, "Invalid wait time value, please enter a valid number"); err != nil {
				log.Printf("Error sending message: %v", err)
			}
			return
		}
		usr.PumpSettings.SetWaitTime(time.Duration(waitTime) * time.Minute)
		log.Printf("Wait time changed to %d minutes", waitTime)

		usr.SetState(user.StateNone)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := secondTelegramClient.SendMessage(recipient, "The wait time for coin pumping has been changed"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "The wait time for coin pumping has been changed")
		}
	}
}
