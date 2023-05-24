package main

import (
	"github.com/agopankov/imPulse/client/pkg/botcommands"
	"github.com/agopankov/imPulse/client/pkg/cancelfuncs"
	"github.com/agopankov/imPulse/client/pkg/grpc"
	"github.com/agopankov/imPulse/client/pkg/secrets"
	"github.com/agopankov/imPulse/client/pkg/telegram"
	"github.com/agopankov/imPulse/client/pkg/user"
	"github.com/agopankov/imPulse/server/pkg/grpcbinance/proto"
	tele "gopkg.in/telebot.v3"
	"log"
	"time"
)

func main() {
	userManager := user.NewUserManager()

	usr := user.NewUser()
	usr.ChangePercent24.SetPercent(20)

	usr.PumpSettings.SetPumpPercent(5)
	usr.PumpSettings.SetWaitTime(15 * time.Minute)

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
		usr, ok := userManager.GetUser(m.Sender.ID)
		if !ok {
			usr = user.NewUser()
			usr.ChangePercent24.SetPercent(20)
			usr.PumpSettings.SetPumpPercent(5)
			usr.PumpSettings.SetWaitTime(15 * time.Minute)
			userManager.AddUser(m.Sender.ID, usr)
		}

		usr.FirstChatID = m.Sender.ID
		botcommands.StartCommandHandlerFirstClient(m, telegramClient, usr)
	})
	telegramClient.HandleCommand("/stop", func(m *tele.Message) {
		botcommands.StopCommandHandler(m, cancelFuncs)
	})
	telegramClient.HandleCommand("/change24percent", func(m *tele.Message) {
		usr, ok := userManager.GetUser(m.Sender.ID)
		if !ok {
			log.Printf("Unknown user with ID %d", m.Sender.ID)
			return
		}

		botcommands.Change24PercentCommandHandler(m, telegramClient, usr)
	})

	secondTelegramClient.HandleCommand("/start", func(m *tele.Message) {
		usr, ok := userManager.GetUser(m.Sender.ID)
		if !ok {
			usr = user.NewUser()
			usr.ChangePercent24.SetPercent(20)
			usr.PumpSettings.SetPumpPercent(5)
			usr.PumpSettings.SetWaitTime(15 * time.Minute)
			userManager.AddUser(m.Sender.ID, usr)
		}

		usr.SecondChatID = m.Sender.ID
		botcommands.StartCommandHandlerSecondClient(m, secondTelegramClient, usr)
	})
	secondTelegramClient.HandleCommand("/setwaittime", func(m *tele.Message) {
		usr, ok := userManager.GetUser(m.Sender.ID)
		if !ok {
			log.Printf("Unknown user with ID %d", m.Sender.ID)
			return
		}

		botcommands.SetWaitTimeCommandHandler(m, secondTelegramClient, usr)
	})
	secondTelegramClient.HandleCommand("/setpumppercent", func(m *tele.Message) {
		usr, ok := userManager.GetUser(m.Sender.ID)
		if !ok {
			log.Printf("Unknown user with ID %d", m.Sender.ID)
			return
		}

		botcommands.SetPumpPercentCommandHandler(m, secondTelegramClient, usr)
	})

	telegramClient.HandleOnMessage(func(m *tele.Message) {
		usr, ok := userManager.GetUser(m.Sender.ID)
		if !ok {
			log.Printf("Unknown user with ID %d", m.Sender.ID)
			return
		}

		botcommands.MessageHandlerFirstClient(m, telegramClient, secondTelegramClient, cancelFuncs, usr, binanceClient)
	})

	secondTelegramClient.HandleOnMessage(func(m *tele.Message) {
		usr, ok := userManager.GetUser(m.Sender.ID)
		if !ok {
			log.Printf("Unknown user with ID %d", m.Sender.ID)
			return
		}

		botcommands.MessageHandlerSecondClient(m, secondTelegramClient, usr)
	})

	go secondTelegramClient.Start()
	telegramClient.Start()
}
