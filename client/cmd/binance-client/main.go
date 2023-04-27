package main

import (
	"context"
	"fmt"
	"github.com/agopankov/binance/client/internal/telegram"
	"github.com/agopankov/binance/server/internal/grpcbinance"
	"google.golang.org/grpc"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
)

type SymbolChange struct {
	Symbol         string
	PriceChange    float64
	PriceChangePct float64
}

func main() {
	_ = os.Getenv("BINANCE_API_KEY")
	_ = os.Getenv("BINANCE_SECRET_KEY")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	binanceClient := grpcbinance.NewBinanceServiceClient(conn)

	telegramClient, err := telegram.NewClient(botToken, binanceClient)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	telegramClient.HandleText(handleText)

	telegramClient.Start()
}

func handleText(m *tele.Message) {
	ctx := context.Background()
	usdtPrices, err := c.binanceClient.GetUSDTPrices(ctx, &grpcbinance.Empty{})
	if err != nil {
		c.SendMessage(m.Sender, fmt.Sprintf("Error getting USDT prices: %v", err))
		return
	}

	changePercent, err := c.binanceClient.Get24HChangePercent(ctx, &grpcbinance.Empty{})
	if err != nil {
		c.SendMessage(m.Sender, fmt.Sprintf("Error getting 24h change percent: %v", err))
		return
	}

}
