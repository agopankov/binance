package main

import (
	"github.com/agopankov/binance/client/pkg/botcommands"
	"github.com/agopankov/binance/client/pkg/cancelfuncs"
	"github.com/agopankov/binance/client/pkg/grpc"
	"github.com/agopankov/binance/client/pkg/secrets"
	"github.com/agopankov/binance/client/pkg/telegram"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	tele "gopkg.in/telebot.v3"
	"log"
	"time"
)

func main() {

	changePercent24 := &telegram.ChangePercent24{}
	changePercent24.SetPercent(20)

	pumpSettings := &telegram.PumpSettings{}
	pumpSettings.SetPumpPercent(5)
	pumpSettings.SetWaitTime(15 * time.Minute)

	secretsForTelegramBots, err := secrets.LoadSecrets()
	if err != nil {
		log.Fatalf("Failed to load secrets: %v", err)
	}

	firstBotToken := secretsForTelegramBots.TelegramBotToken
	secondBotToken := secretsForTelegramBots.TelegramBotTokenSecond

	conn, err := grpc.NewGRPCConnection("binance-server:50051")
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}()

	binanceClient := proto.NewBinanceServiceClient(conn)

	chatState := &telegram.ChatState{}

	telegramClient, err := telegram.NewClient(firstBotToken)
	if err != nil {
		log.Fatalf("Error creating Telegram bot: %v", err)
	}

	secondTelegramClient, err := telegram.NewClient(secondBotToken)
	if err != nil {
		log.Fatalf("Error creating second Telegram bot: %v", err)
	}

	cancelFuncs := cancelfuncs.NewCancelFuncs()

	telegramClient.HandleCommand("/start", func(m *tele.Message) {
		chatState.SetFirstChatID(m.Sender.ID)
		botcommands.StartCommandHandlerFirstClient(m, telegramClient, chatState)
	})
	telegramClient.HandleCommand("/stop", func(m *tele.Message) {
		botcommands.StopCommandHandler(m, cancelFuncs)
	})
	telegramClient.HandleCommand("/change24percent", func(m *tele.Message) {
		botcommands.Change24PercentCommandHandler(m, telegramClient, chatState, changePercent24)
	})

	secondTelegramClient.HandleCommand("/start", func(m *tele.Message) {
		chatState.SetSecondChatID(m.Sender.ID)
		botcommands.StartCommandHandlerSecondClient(m, secondTelegramClient, chatState)
	})
	secondTelegramClient.HandleCommand("/setwaittime", func(m *tele.Message) {
		botcommands.SetWaitTimeCommandHandler(m, secondTelegramClient, chatState, pumpSettings)
	})
	secondTelegramClient.HandleCommand("/setpumppercent", func(m *tele.Message) {
		botcommands.SetPumpPercentCommandHandler(m, secondTelegramClient, chatState, pumpSettings)
	})

	telegramClient.HandleOnMessage(func(m *tele.Message) {
		botcommands.MessageHandlerFirstClient(m, telegramClient, secondTelegramClient, cancelFuncs, chatState, binanceClient, changePercent24, pumpSettings)
	})

	secondTelegramClient.HandleOnMessage(func(m *tele.Message) {
		botcommands.MessageHandlerSecondClient(m, secondTelegramClient, chatState, pumpSettings)
	})

	go secondTelegramClient.Start()
	telegramClient.Start()
}
