package telegram

import (
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"time"

	tele "gopkg.in/telebot.v3"
)

type Client struct {
	botToken      string
	bot           *tele.Bot
	binanceClient proto.BinanceServiceClient // Обновленный тип поля binanceClient
}

// Добавьте аргумент binanceClient в функцию NewClient
func NewClient(botToken string, binanceClient proto.BinanceServiceClient) (*Client, error) {
	bot, err := tele.NewBot(tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		botToken:      botToken,
		bot:           bot,
		binanceClient: binanceClient, // Инициализируйте поле binanceClient
	}, nil
}

func (c *Client) Start() {
	c.bot.Start()
}

func (c *Client) HandleText(handler func(m *tele.Message)) {
	c.bot.Handle(tele.OnText, func(c tele.Context) error {
		handler(c.Message())
		return nil
	})
}

func (c *Client) SendMessage(recipient *tele.User, text string) (*tele.Message, error) {
	return c.bot.Send(recipient, text)
}
