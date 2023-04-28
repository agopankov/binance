package main

import (
	"context"
	"fmt"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"google.golang.org/grpc"
	tele "gopkg.in/telebot.v3"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type SymbolChange struct {
	Symbol         string
	PriceChange    float64
	PriceChangePct float64
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	conn, err := grpc.Dial("binance-server:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	binanceClient := proto.NewBinanceServiceClient(conn)

	telegramClient, err := telegram.NewClient(botToken, binanceClient)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	telegramClient.HandleCommand("start", func(m *tele.Message) {
		log.Printf("Received /start command from chat ID %d", m.Sender.ID)

		chatID := int64(m.Sender.ID)
		go monitorPriceChanges(telegramClient, binanceClient, chatID)

		recipient := &tele.User{ID: chatID}
		if _, err := telegramClient.SendMessage(recipient, "Hi"); err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			log.Printf("Sent message to chat ID %d: %s", chatID, "Hi")
		}
	})

	telegramClient.Start()
}

func monitorPriceChanges(telegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatID int64) {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		ctx := context.Background()
		usdtPrices, err := binanceClient.GetUSDTPrices(ctx, &proto.Empty{})
		if err != nil {
			log.Printf("Error getting USDT prices: %v", err)
			continue
		}

		changePercent, err := binanceClient.Get24HChangePercent(ctx, &proto.Empty{})
		if err != nil {
			log.Printf("Error getting 24h change percent: %v", err)
			continue
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
		message := sb.String()
		if message != "" {
			recipient := &tele.User{ID: chatID}
			if _, err := telegramClient.SendMessage(recipient, message); err != nil {
				log.Printf("Error sending message: %v", err)
			} else {
				log.Printf("Sent message to chat ID %d: %s", chatID, message)
			}
		} else {
			log.Printf("No symbols with 20%% change to report.")
		}
	}
}
