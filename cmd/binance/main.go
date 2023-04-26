package main

import (
	"fmt"
	"github.com/agopankov/binance/internal/binanceAPI"
	"github.com/agopankov/binance/internal/telegram"
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
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	binanceClient := binanceAPI.NewClient(apiKey, secretKey)

	telegramClient, err := telegram.NewClient(botToken)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	telegramClient.HandleText(func(m *tele.Message) {
		usdtPrices, err := binanceClient.GetUSDTPrices()
		if err != nil {
			telegramClient.SendMessage(m.Sender, fmt.Sprintf("Error getting USDT prices: %v", err))
			return
		}

		changePercent, err := binanceClient.Get24hChangePercent()
		if err != nil {
			telegramClient.SendMessage(m.Sender, fmt.Sprintf("Error getting 24h change percent: %v", err))
			return
		}

		var filteredSymbols []SymbolChange
		for symbol, price := range usdtPrices {
			change, ok := changePercent[symbol]
			if !ok || change < 20 {
				continue
			}
			filteredSymbols = append(filteredSymbols, SymbolChange{
				Symbol:         symbol,
				PriceChange:    price,
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

		telegramClient.SendMessage(m.Sender, sb.String())
	}, binanceClient)

	telegramClient.Start()
}
