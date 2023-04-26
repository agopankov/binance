package main

import (
	"github.com/agopankov/binance/internal/telegram"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
)

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	telegramClient, err := telegram.NewClient(botToken)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	telegramClient.HandleText(func(m *tele.Message) {
		telegramClient.SendMessage(m.Sender, "Hello, World!")
	})

	telegramClient.Start()
}
