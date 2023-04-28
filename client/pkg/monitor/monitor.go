package monitor

import (
	"context"
	"fmt"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/client/pkg/tracker"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	tele "gopkg.in/telebot.v3"
	"log"
	"strconv"
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
	notifyTicker := time.NewTicker(15 * time.Minute)

	for {
		select {
		case <-ticker.C:
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

			for _, symbolChange := range newTrackedSymbols {
				message := fmt.Sprintf("Symbol: %s\nPrice: %s\nChange: %.2f%%\n", symbolChange.Symbol, symbolChange.PriceChange, symbolChange.PriceChangePct)
				recipient := &tele.User{ID: chatID}
				_, err := telegramClient.SendMessage(recipient, message)
				if err != nil {
					log.Printf("Error sending message: %v\n", err)
				}
			}

			for symbol, symbolChange := range trackerInstance.GetTrackedSymbols() {
				currentPrice := getPriceForSymbol(symbol, usdtPrices.Prices)
				newChange := calculateChange(symbolChange.PriceChange, currentPrice)

				if newChange < 20 {
					trackerInstance.RemoveTrackedSymbol(symbol)
					continue
				}

				if time.Since(symbolChange.AddedAt) <= 15*time.Minute && newChange >= symbolChange.PriceChangePct+10 {
					message := fmt.Sprintf("Symbol: %s\nPrice: %s\nChange: %.2f%%\n", symbol, currentPrice, newChange)
					recipient := &tele.User{ID: secondChatID}
					_, err := telegramClient.SendMessage(recipient, message)
					if err != nil {
						log.Printf("Error sending message: %v\n", err)
					}
				}

				symbolChange.PriceChange = currentPrice
				symbolChange.PriceChangePct = newChange
				trackerInstance.UpdateTrackedSymbol(symbolChange)
			}
		case <-notifyTicker.C:
			ctx := context.Background()
			usdtPrices, err := binanceClient.GetUSDTPrices(ctx, &proto.Empty{})
			if err != nil {
				log.Printf("Error getting USDT prices: %v", err)
				continue
			}

			for symbol, symbolChange := range trackerInstance.GetTrackedSymbols() {
				currentPrice := getPriceForSymbol(symbol, usdtPrices.Prices)
				newChange := calculateChange(symbolChange.PriceChange, currentPrice)

				message := fmt.Sprintf("Symbol: %s\nPrice: %s\nChange: %.2f%%\n", symbol, currentPrice, newChange)
				recipient := &tele.User{ID: chatID}
				_, err := telegramClient.SendMessage(recipient, message)
				if err != nil {
					log.Printf("Error sending message: %v\n", err)
				}
			}
		}
	}
}
