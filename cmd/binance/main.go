package main

import (
	"fmt"
	"github.com/agopankov/binance/internal/binanceAPI"
	"github.com/agopankov/binance/internal/telegram"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
)

func main() {
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	binanceClient := binanceAPI.NewClient(apiKey, secretKey)

	account, err := binanceClient.GetAccountInfo()
	if err != nil {
		log.Fatalf("Error getting account info: %v", err)
	}

	fmt.Println("Balances:")
	for _, balance := range account.Balances {
		fmt.Println(balance.Asset, balance.Free)
	}

	telegramClient, err := telegram.NewClient(botToken)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	telegramClient.HandleText(func(m *tele.Message) {
		telegramClient.SendMessage(m.Sender, "Hello, World!")
	})

	telegramClient.Start()
}
