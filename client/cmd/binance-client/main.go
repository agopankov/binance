package main

import (
	"github.com/agopankov/binance/client/pkg/aws"
	"github.com/agopankov/binance/client/pkg/monitor"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/client/pkg/tracker"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"google.golang.org/grpc"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
)

func main() {
	secretID := os.Getenv("AWS_SECRET_ID")
	secrets, err := aws.GetSecrets(context.Background(), secretID)
	if err != nil {
		log.Fatalf("Failed to get secrets from AWS Secrets Manager: %v", err)
	}

	firstBotToken := secrets.TelegramBotToken
	secondBotToken := secrets.TelegramBotTokenSecond

	conn, err := grpc.Dial("binance-server:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

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
	secondChatID := int64(secondUser.ID)

	telegramClient.HandleCommand("start", func(m *tele.Message) {
		log.Printf("Received /start command from chat ID %d", m.Sender.ID)

		chatID := int64(m.Sender.ID)
		trackerInstance := tracker.NewTracker()
		go monitor.MonitorPriceChanges(telegramClient, binanceClient, chatID, secondChatID, trackerInstance)

		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "Hi"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "Hi")
		}
	})

	secondTelegramClient.HandleCommand("start", func(m *tele.Message) {
		log.Printf("Received /start command from second chat ID %d", m.Sender.ID)

		chatID := int64(m.Sender.ID)
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
