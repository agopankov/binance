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
	TelegramClient       *telegram.Client
	SecondTelegramClient *telegram.Client
	BinanceClient        proto.BinanceServiceClient
	ChatID               int64
	SecondChatID         int64
	TrackerInstance      *tracker.Tracker
}

func getPriceForSymbol(symbol string, prices []*proto.USDTPrice) string {
	for _, price := range prices {
		if price.Symbol == symbol {
			return fmt.Sprintf("%.8f", price.Price)
		}
	}
	return ""
}

func PriceChanges(ctx context.Context, client *telegram.Client, secondTelegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatState *telegram.ChatState, trackerInstance *tracker.Tracker, changePercent24 *telegram.ChangePercent24, pumpSettings *telegram.PumpSettings) {
	ticker := time.NewTicker(5 * time.Second)
	notifyTicker := time.NewTicker(1 * time.Minute)
	logTicker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-logTicker.C:
			processLogTicker(trackerInstance)
		case <-ticker.C:
			processTicker(client, secondTelegramClient, binanceClient, chatState, trackerInstance, changePercent24, pumpSettings)
		case <-notifyTicker.C:
			processNotifyTicker(client, binanceClient, chatState, trackerInstance)
		}
	}
}

func processTicker(telegramClient *telegram.Client, secondTelegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatState *telegram.ChatState, trackerInstance *tracker.Tracker, changePercent24 *telegram.ChangePercent24, pumpSettings *telegram.PumpSettings) {
	chatID := chatState.GetFirstChatID()
	secondChatID := chatState.GetSecondChatID()

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

		if change >= changePercent24.GetPercent() && !trackerInstance.IsTracked(price.Symbol) {
			newSymbol := tracker.SymbolChange{
				Symbol:           price.Symbol,
				PriceChange:      fmt.Sprintf("%.8f", price.Price),
				FirstPriceChange: fmt.Sprintf("%.8f", price.Price),
				PriceChangePct:   change,
				AddedAt:          time.Now(),
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
		message := fmt.Sprintf("%s %s / USDT P: %s Ch24h: %.2f%% \n", emoji, symbolChange.Symbol[:len(symbolChange.Symbol)-4], price, symbolChange.PriceChangePct)
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

	for symbol, _ := range trackerInstance.GetTrackedSymbols() {
		change24h := 0.0
		for _, changePercent := range changePercent.ChangePercents {
			if symbol == changePercent.Symbol {
				change24h = changePercent.ChangePercent
				break
			}
		}

		if change24h <= changePercent24.GetPercent() {
			trackerInstance.RemoveTrackedSymbol(symbol)
			continue
		}
	}

	trackedSymbols := trackerInstance.GetTrackedSymbols()

	var sortedSymbols []tracker.SymbolChange
	for _, symbolChange := range trackedSymbols {
		sortedSymbols = append(sortedSymbols, symbolChange)
	}

	for _, symbolChange := range sortedSymbols {
		currentPrice := getPriceForSymbol(symbolChange.Symbol, usdtPrices.Prices)

		currentPriceFloat, _ := strconv.ParseFloat(currentPrice, 64)
		previousPriceFloat, _ := strconv.ParseFloat(symbolChange.FirstPriceChange, 64)

		if !symbolChange.NotificationOfPump && time.Since(symbolChange.AddedAt) <= pumpSettings.GetWaitTime() && (currentPriceFloat/previousPriceFloat)-1 >= pumpSettings.GetPumpPercent() {
			log.Printf("Pump %s, current pump persent %.5f%%, firstPrice: %.7f, currentPrice: %.7f, notification: %t",
				symbolChange.Symbol[:len(symbolChange.Symbol)-4],
				((currentPriceFloat/previousPriceFloat)-1)*100,
				previousPriceFloat,
				currentPriceFloat,
				symbolChange.NotificationOfPump)
			message := fmt.Sprintf("🚀 %s / USDT P: %.7f Ch24h: %.2f%% (PrP: %.7f) \n",
				symbolChange.Symbol[:len(symbolChange.Symbol)-4],
				currentPriceFloat,
				symbolChange.PriceChangePct,
				previousPriceFloat,
			)
			messageBuilder.WriteString(message)

			recipient := &tele.User{ID: secondChatID}
			_, err := secondTelegramClient.SendMessage(recipient, message)
			if err != nil {
				log.Printf("Error sending message to the second chat: %v\n", err)
			} else {
				symbolChange.NotificationOfPump = true
				trackerInstance.UpdateTrackedSymbol(symbolChange)
			}
		} else {
			log.Printf("Don't pump %s, current pump persent %.5f%%, notification: %t",
				symbolChange.Symbol[:len(symbolChange.Symbol)-4],
				((currentPriceFloat/previousPriceFloat)-1)*100,
				symbolChange.NotificationOfPump)
		}
	}

}

func processNotifyTicker(telegramClient *telegram.Client, binanceClient proto.BinanceServiceClient, chatState *telegram.ChatState, trackerInstance *tracker.Tracker) {
	chatID := chatState.GetFirstChatID()

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

		var change24h float64
		for _, changePercentData := range changePercent.ChangePercents {
			if symbolChange.Symbol == changePercentData.Symbol {
				change24h = changePercentData.ChangePercent
				break
			}
		}

		price := strings.TrimRight(strings.TrimRight(currentPrice, "0"), ".")
		emoji := ""
		currentPriceFloat, _ := strconv.ParseFloat(currentPrice, 64)
		previousPriceFloat, _ := strconv.ParseFloat(symbolChange.PriceChange, 64)
		if currentPriceFloat > previousPriceFloat {
			emoji = "📈"
		} else if currentPriceFloat < previousPriceFloat {
			emoji = "📉"
		} else {
			emoji = "🔹"
		}

		message := fmt.Sprintf("%s %s / USDT P: %s Ch24h: %.2f%% \n", emoji, symbolChange.Symbol[:len(symbolChange.Symbol)-4], price, change24h)
		messageBuilder.WriteString(message)

		symbolChange.PriceChange = currentPrice
		trackerInstance.UpdateTrackedSymbol(symbolChange)
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
