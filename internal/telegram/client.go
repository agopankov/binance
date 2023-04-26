package telegram

import (
	"time"

	tele "gopkg.in/telebot.v3"
)

type Client struct {
	botToken string
	bot      *tele.Bot
}

func NewClient(botToken string) (*Client, error) {
	bot, err := tele.NewBot(tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		botToken: botToken,
		bot:      bot,
	}, nil
}

func (c *Client) Start() {
	c.bot.Start()
}

func (c *Client) HandleText(handler func(m *tele.Message)) {
	c.bot.Handle(tele.OnText, func(c tele.Context) error {
		return c.Send("Hello!")
	})
}

func (c *Client) SendMessage(recipient *tele.User, text string) {
	c.bot.Send(recipient, text)
}
