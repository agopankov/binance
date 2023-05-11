package main

import (
	"context"
	"encoding/json"
	"github.com/agopankov/binance/client/pkg/cancelfuncs"
	"github.com/agopankov/binance/client/pkg/monitor"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/client/pkg/tracker"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
	"strconv"
	"time"
)

type SecretKeys struct {
	TelegramBotToken       string `json:"TELEGRAM_BOT_TOKEN"`
	TelegramBotTokenSecond string `json:"TELEGRAM_BOT_TOKEN_SECOND"`
}

func main() {
	var secrets SecretKeys
	var secondChatID int64

	changePercent24 := &telegram.ChangePercent24{}
	changePercent24.SetPercent(20)

	chatState := &telegram.ChatState{}

	pumpSettings := &telegram.PumpSettings{}
	pumpSettings.SetPumpPercent(0.05)
	pumpSettings.SetWaitTime(15 * time.Minute)

	firstBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	secondBotToken := os.Getenv("TELEGRAM_BOT_TOKEN_SECOND")

	if firstBotToken == "" || secondBotToken == "" {
		secretsFile, err := os.ReadFile("/mnt/secrets-store/prod_binance_secret")
		if err != nil {
			log.Fatalf("Failed to read secrets file: %v", err)
		}
		err = json.Unmarshal(secretsFile, &secrets)
		if err != nil {
			log.Fatalf("Failed to unmarshal secrets JSON: %v", err)
		}
		firstBotToken = secrets.TelegramBotToken
		secondBotToken = secrets.TelegramBotTokenSecond
	}

	conn, err := grpc.Dial("binance-server:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}()

	binanceClient := proto.NewBinanceServiceClient(conn)

	telegramClient, err := telegram.NewClient(firstBotToken, binanceClient)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	secondTelegramClient, err := telegram.NewClient(secondBotToken, binanceClient)
	if err != nil {
		log.Fatalf("Error creating second Telegram bot: %v", err)
	}

	cancelFuncs := cancelfuncs.NewCancelFuncs()

	secondTelegramClient.HandleCommand("/start", func(m *tele.Message) {
		log.Printf("Received /start command from second chat ID %d", m.Sender.ID)

		chatState.SetSecondChatID(m.Sender.ID)

		secondChatID = m.Sender.ID
		recipient := &tele.User{ID: secondChatID}
		if _, err := secondTelegramClient.SendMessage(recipient, "The service for monitoring coins that are being pumped has been launched"); err != nil {
			log.Printf("Error sending message to second chat: %v", err)
		} else {
			log.Printf("Sent message to second chat ID %d: %s", secondChatID, "Hi")
		}
	})

	telegramClient.HandleCommand("/start", func(m *tele.Message) {
		log.Printf("Received /start command from chat ID %d", m.Sender.ID)

		chatState.SetFirstChatID(m.Sender.ID)

		chatID := m.Sender.ID
		trackerInstance := tracker.NewTracker()

		ctx, cancel := context.WithCancel(context.Background())
		cancelFuncs.Add(chatID, cancel)

		go monitor.PriceChanges(ctx, telegramClient, secondTelegramClient, binanceClient, chatState, trackerInstance, changePercent24, pumpSettings)

		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "Tracking service launched"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "Hi")
		}
	})

	telegramClient.HandleCommand("/stop", func(m *tele.Message) {
		log.Printf("Received /stop command from chat ID %d", m.Sender.ID)

		chatID := m.Sender.ID

		cancelFuncs.Remove(chatID)
	})

	telegramClient.HandleCommand("/change24percent", func(m *tele.Message) {
		chatState.SetState(telegram.StateAwaitingPercent)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "Please enter the new percent value"); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	})

	telegramClient.HandleOnMessage(func(m *tele.Message) {
		if chatState.GetState() == telegram.StateAwaitingPercent {
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
	})

	secondTelegramClient.HandleCommand("/setwaittime", func(m *tele.Message) {
		waitTime, err := strconv.Atoi(m.Text)
		if err != nil {
			return
		}
		pumpSettings.SetWaitTime(time.Duration(waitTime) * time.Minute)
	})

	secondTelegramClient.HandleCommand("/setpumppercent", func(m *tele.Message) {
		chatState.SetState(telegram.StateAwaitingPercent)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := secondTelegramClient.SendMessage(recipient, "Please enter the new percent value"); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	})

	secondTelegramClient.HandleOnMessage(func(m *tele.Message) {
		if chatState.GetState() == telegram.StateAwaitingPercent {
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
		}
	})

	go secondTelegramClient.Start()
	telegramClient.Start()
}
