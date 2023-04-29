package monitor

import (
	"context"
	"fmt"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/client/pkg/tracker"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	tele "gopkg.in/telebot.v3"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Monitor struct {
	TelegramClient  *telegram.Client
	BinanceClient   proto.BinanceServiceClient
	ChatID          int64
	SecondChatID    int64
	TrackerInstance *tracker.Tracker
}

func NewMonitor(telegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatID int64, secondChatID int64, trackerInstance *tracker.Tracker) *Monitor {
	return &Monitor{
		TelegramClient:  telegramClient,
		BinanceClient:   binanceClient,
		ChatID:          chatID,
		SecondChatID:    secondChatID,
		TrackerInstance: trackerInstance,
	}
}

func getPriceForSymbol(symbol string, prices []*proto.USDTPrice) string {
	for _, price := range prices {
		if price.Symbol == symbol {
			return fmt.Sprintf("%.8f", price.Price)
		}
	}
	return ""
}

func calculateChange(oldPrice, newPrice string) float64 {
	oldPriceFloat, err := strconv.ParseFloat(oldPrice, 64)
	if err != nil {
		log.Printf("Error parsing price: %v\n", err)
		return 0
	}

	newPriceFloat, err := strconv.ParseFloat(newPrice, 64)
	if err != nil {
		log.Printf("Error parsing price: %v\n", err)
		return 0
	}

	if oldPriceFloat == 0 {
		return 0
	}

	return ((newPriceFloat - oldPriceFloat) / oldPriceFloat) * 100
}

func MonitorPriceChanges(telegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatID int64, secondChatID int64, trackerInstance *tracker.Tracker) {
	ticker := time.NewTicker(5 * time.Second)
	notifyTicker := time.NewTicker(1 * time.Minute)
	logTicker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-logTicker.C:
			processLogTicker(trackerInstance)
		case <-ticker.C:
			processTicker(telegramClient, binanceClient, chatID, secondChatID, trackerInstance)
		case <-notifyTicker.C:
			processNotifyTicker(telegramClient, binanceClient, chatID, trackerInstance)
		}
	}
}

func processTicker(telegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatID int64, secondChatID int64, trackerInstance *tracker.Tracker) {
	ctx := context.Background()
	usdtPrices, err := binanceClient.GetUSDTPrices(ctx, &proto.Empty{})
	if err != nil {
		log.Printf("Error getting USDT prices: %v", err)
		return
	}

	changePercent, err := binanceClient.Get24HChangePercent(ctx, &proto.Empty{})
	if err != nil {
		log.Printf("Error getting 24h change percent: %v", err)
		return
	}

	var newTrackedSymbols []tracker.SymbolChange
	for _, price := range usdtPrices.Prices {
		change := 0.0
		for _, changePercent := range changePercent.ChangePercents {
			if price.Symbol == changePercent.Symbol {
				change = changePercent.ChangePercent
				break
			}
		}

		if change >= 20 && !trackerInstance.IsTracked(price.Symbol) {
			newSymbol := tracker.SymbolChange{
				Symbol:         price.Symbol,
				PriceChange:    fmt.Sprintf("%.8f", price.Price),
				PriceChangePct: change,
				AddedAt:        time.Now(),
			}
			trackerInstance.UpdateTrackedSymbol(newSymbol)
			newTrackedSymbols = append(newTrackedSymbols, newSymbol)
		}
	}

	sort.Slice(newTrackedSymbols, func(i, j int) bool {
		return newTrackedSymbols[i].PriceChangePct > newTrackedSymbols[j].PriceChangePct
	})

	var messageBuilder strings.Builder
	for _, symbolChange := range newTrackedSymbols {
		emoji := "✅"
		price := strings.TrimRight(strings.TrimRight(symbolChange.PriceChange, "0"), ".")
		message := fmt.Sprintf("%s / USDT P: %s Ch24h: *%.2f%%* %s\n", symbolChange.Symbol[:len(symbolChange.Symbol)-4], price, symbolChange.PriceChangePct, emoji)
		messageBuilder.WriteString(message)
	}

	if messageBuilder.Len() > 0 {
		message := messageBuilder.String()
		recipient := &tele.User{ID: chatID}
		_, err := telegramClient.SendMessage(recipient, message)
		if err != nil {
			log.Printf("Error sending message: %v\n", err)
		}
	}

	for symbol, symbolChange := range trackerInstance.GetTrackedSymbols() {
		change24h := 0.0
		for _, changePercent := range changePercent.ChangePercents {
			if symbol == changePercent.Symbol {
				change24h = changePercent.ChangePercent
				break
			}
		}

		if change24h <= 20 {
			trackerInstance.RemoveTrackedSymbol(symbol)
			continue
		}

		if time.Since(symbolChange.AddedAt) <= 15*time.Minute && change24h >= symbolChange.PriceChangePct+10 {
			message := fmt.Sprintf("Symbol: %s\nPrice: %s\nChange: %.2f%%\n", symbol, getPriceForSymbol(symbol, usdtPrices.Prices), change24h)
			recipient := &tele.User{ID: secondChatID}
			_, err := telegramClient.SendMessage(recipient, message)
			if err != nil {
				log.Printf("Error sending message: %v\n", err)
			}
		}
	}
}

func processNotifyTicker(telegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatID int64, trackerInstance *tracker.Tracker) {
	ctx := context.Background()
	usdtPrices, err := binanceClient.GetUSDTPrices(ctx, &proto.Empty{})
	if err != nil {
		log.Printf("Error getting USDT prices: %v", err)
		return
	}

	trackedSymbols := trackerInstance.GetTrackedSymbols()

	var sortedSymbols []tracker.SymbolChange
	for _, symbolChange := range trackedSymbols {
		sortedSymbols = append(sortedSymbols, symbolChange)
	}
	sort.Slice(sortedSymbols, func(i, j int) bool {
		return sortedSymbols[i].PriceChangePct > sortedSymbols[j].PriceChangePct
	})

	var messageBuilder strings.Builder
	for _, symbolChange := range sortedSymbols {
		currentPrice := getPriceForSymbol(symbolChange.Symbol, usdtPrices.Prices)
		newChange := calculateChange(symbolChange.PriceChange, currentPrice)

		price := strings.TrimRight(strings.TrimRight(currentPrice, "0"), ".")
		emoji := ""
		if newChange > symbolChange.PriceChangePct {
			emoji = "📈"
		} else if newChange < symbolChange.PriceChangePct {
			emoji = "📉"
		} else {
			emoji = "➖"
		}

		message := fmt.Sprintf("%s / USDT P: %s Ch24h: *%.2f%%* %s\n", symbolChange.Symbol[:len(symbolChange.Symbol)-4], price, newChange, emoji)
		messageBuilder.WriteString(message)
	}

	if messageBuilder.Len() > 0 {
		message := messageBuilder.String()
		recipient := &tele.User{ID: chatID}
		_, err := telegramClient.SendMessage(recipient, message)
		if err != nil {
			log.Printf("Error sending message: %v\n", err)
		}
	}
}

func processLogTicker(trackerInstance *tracker.Tracker) {
	trackedSymbols := trackerInstance.GetTrackedSymbols()
	if len(trackedSymbols) == 0 {
		log.Printf("TrackedSymbols is empty")
	}
	for symbol, symbolChange := range trackedSymbols {
		log.Printf("Symbol: %s, PriceChange: %s, PriceChangePct: %.2f%%, AddedAt: %v\n",
			symbol, symbolChange.PriceChange, symbolChange.PriceChangePct, symbolChange.AddedAt)
	}
}
