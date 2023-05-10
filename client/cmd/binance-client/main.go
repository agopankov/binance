package main

import (
	"encoding/json"
	"github.com/agopankov/binance/client/pkg/monitor"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/client/pkg/tracker"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
)

type SecretKeys struct {
	TelegramBotToken       string `json:"TELEGRAM_BOT_TOKEN"`
	TelegramBotTokenSecond string `json:"TELEGRAM_BOT_TOKEN_SECOND"`
}

func main() {
	var secrets SecretKeys

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

	secondUser := secondTelegramClient.Bot().Me
	secondChatID := secondUser.ID

	telegramClient.HandleCommand("start", func(m *tele.Message) {
		log.Printf("Received /start command from chat ID %d", m.Sender.ID)

		chatID := m.Sender.ID
		trackerInstance := tracker.NewTracker()
		go monitor.PriceChanges(telegramClient, binanceClient, chatID, secondChatID, trackerInstance)

		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "Hi"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "Hi")
		}
	})

	secondTelegramClient.HandleCommand("start", func(m *tele.Message) {
		log.Printf("Received /start command from second chat ID %d", m.Sender.ID)

		chatID := m.Sender.ID
		recipient := &tele.User{ID: chatID}
		if _, err := secondTelegramClient.SendMessage(recipient, "Hi"); err != nil {
			log.Printf("Error sending message to second chat: %v", err)
		} else {
			log.Printf("Sent message to second chat ID %d: %s", chatID, "Hi")
		}
	})

	go secondTelegramClient.Start()
	telegramClient.Start()
}
