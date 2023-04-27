package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
	"sort"
	"strings"
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

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Println("Error closing connection:", closeErr)
		}
	}()

	binanceClient := proto.NewBinanceServiceClient(conn)

	telegramClient, err := telegram.NewClient(botToken, binanceClient)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	handler := func(m *tele.Message) {
		ctx := context.Background()
		usdtPrices, err := binanceClient.GetUSDTPrices(ctx, &proto.Empty{})
		if err != nil {
			if _, err := telegramClient.SendMessage(m.Sender, fmt.Sprintf("Error getting USDT prices: %v", err)); err != nil {
				log.Printf("Error sending message: %v", err)
			}
			return
		}

		changePercent, err := binanceClient.Get24HChangePercent(ctx, &proto.Empty{})
		if err != nil {
			if _, err := telegramClient.SendMessage(m.Sender, fmt.Sprintf("Error getting 24h change percent: %v", err)); err != nil {
				log.Printf("Error sending message: %v", err)
			}
			return
		}

		var filteredSymbols []SymbolChange
		for _, price := range usdtPrices.Prices {
			change := 0.0
			for _, changePercent := range changePercent.ChangePercents {
				if price.Symbol == changePercent.Symbol {
					change = changePercent.ChangePercent
					break
				}
			}

			if change < 20 {
				continue
			}
			filteredSymbols = append(filteredSymbols, SymbolChange{
				Symbol:         price.Symbol,
				PriceChange:    price.Price,
				PriceChangePct: change,
			})
		}

		sort.Slice(filteredSymbols, func(i, j int) bool {
			return filteredSymbols[i].PriceChangePct > filteredSymbols[j].PriceChangePct
		})

		var sb strings.Builder
		sb.WriteString("Prices in USDT with more than 20% change in the last 24h:\n\n")
		for _, symbolChange := range filteredSymbols {
			formattedSymbol := strings.Replace(symbolChange.Symbol, "USDT", "/USDT", 1)
			sb.WriteString(fmt.Sprintf("%s: %.2f (24h change: %.2f%%)\n", formattedSymbol, symbolChange.PriceChange, symbolChange.PriceChangePct))
		}

		if _, err := telegramClient.SendMessage(m.Sender, sb.String()); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}

	telegramClient.HandleText(handler)

	telegramClient.Start()
}
