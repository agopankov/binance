package botcommands

import (
	"context"
	"fmt"
	"github.com/agopankov/binance/client/pkg/cancelfuncs"
	"github.com/agopankov/binance/client/pkg/emailverify"
	"github.com/agopankov/binance/client/pkg/monitor"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/client/pkg/tracker"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"github.com/aws/aws-sdk-go/aws/session"
	tele "gopkg.in/telebot.v3"
	"log"
	"net/mail"
	"strconv"
	"time"
)

func StartCommandHandlerFirstClient(m *tele.Message, telegramClient *telegram.Client, chatState *telegram.ChatState) {
	log.Printf("Received /start command from chat ID %d", m.Sender.ID)
	chatState.SetState(telegram.StateAwaitingEmail)
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	if _, err := telegramClient.SendMessage(recipient, "Please enter your email address for verification"); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func StartCommandHandlerSecondClient(m *tele.Message, secondTelegramClient *telegram.Client, chatState *telegram.ChatState) {
	log.Printf("Received /start command from second chat ID %d", m.Sender.ID)
	chatState.SetSecondChatID(m.Sender.ID)
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

func Change24PercentCommandHandler(m *tele.Message, telegramClient *telegram.Client, chatState *telegram.ChatState, changePercent24 *telegram.ChangePercent24) {
	chatState.SetState(telegram.StateAwaitingPercent)
	currentPercent24 := changePercent24.GetPercent()
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	msg := fmt.Sprintf("Please enter the new percent value (current value is %.2f)", currentPercent24)
	if _, err := telegramClient.SendMessage(recipient, msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func SetWaitTimeCommandHandler(m *tele.Message, secondTelegramClient *telegram.Client, chatState *telegram.ChatState, pumpSettings *telegram.PumpSettings) {
	chatState.SetState(telegram.StateAwaitingWaitTime)
	currentWaitTime := pumpSettings.GetWaitTime()
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	msg := fmt.Sprintf("Please enter the new wait time in minutes (current wait time is %s)", currentWaitTime)
	if _, err := secondTelegramClient.SendMessage(recipient, msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func SetPumpPercentCommandHandler(m *tele.Message, secondTelegramClient *telegram.Client, chatState *telegram.ChatState, pumpSettings *telegram.PumpSettings) {
	chatState.SetState(telegram.StateAwaitingPercent)
	currentPumpPercent := pumpSettings.GetPumpPercent()
	chatID := m.Sender.ID
	recipient := &tele.User{ID: chatID}
	msg := fmt.Sprintf("Please enter the new percent value (current percent is %.2f)", currentPumpPercent)
	if _, err := secondTelegramClient.SendMessage(recipient, msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func MessageHandlerFirstClient(m *tele.Message, telegramClient *telegram.Client, secondTelegramClient *telegram.Client, cancelFuncs *cancelfuncs.CancelFuncs, chatState *telegram.ChatState, binanceClient proto.BinanceServiceClient, changePercent24 *telegram.ChangePercent24, pumpSettings *telegram.PumpSettings) {
	switch chatState.GetState() {
	case telegram.StateAwaitingEmail:
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
		chatState.SetEmail(email)
		emailverify.SendVerificationEmail(sess, email)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "A verification code has been sent to your email. Please enter it."); err != nil {
			log.Printf("Error sending message: %v", err)
		}

		chatState.SetState(telegram.StateAwaitingVerification)

	case telegram.StateAwaitingVerification:
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		if emailverify.VerifyCode(sess, chatState.GetEmail(), m.Text) {
			chatID := m.Sender.ID
			recipient := &tele.User{ID: chatID}
			chatState.SetState(telegram.StateNone)

			trackerInstance := tracker.NewTracker()

			ctx, cancel := context.WithCancel(context.Background())
			cancelFuncs.Add(chatID, cancel)

			go monitor.PriceChanges(ctx, telegramClient, secondTelegramClient, binanceClient, chatState, trackerInstance, changePercent24, pumpSettings)

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

	case telegram.StateAwaitingPercent:
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
		changePercent24.SetPercent(newPercent)
		log.Printf("Percent changed to %f", newPercent)

		chatState.SetState(telegram.StateNone)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "The percentage of pumping for tracked coins has been changed"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "The percentage of pumping for tracked coins has been changed")
		}
	}
}

func MessageHandlerSecondClient(m *tele.Message, secondTelegramClient *telegram.Client, chatState *telegram.ChatState, pumpSettings *telegram.PumpSettings) {
	switch chatState.GetState() {
	case telegram.StateAwaitingPercent:
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
		pumpSettings.SetPumpPercent(pumpPercent)
		log.Printf("Percent of pump changed to %f", pumpPercent)

		chatState.SetState(telegram.StateNone)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := secondTelegramClient.SendMessage(recipient, "The percentage expected for the pump has been changed"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "The percentage expected for the pump has been changed")
		}

	case telegram.StateAwaitingWaitTime:
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
		pumpSettings.SetWaitTime(time.Duration(waitTime) * time.Minute)
		log.Printf("Wait time changed to %d minutes", waitTime)

		chatState.SetState(telegram.StateNone)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := secondTelegramClient.SendMessage(recipient, "The wait time for coin pumping has been changed"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "The wait time for coin pumping has been changed")
		}
	}
}
