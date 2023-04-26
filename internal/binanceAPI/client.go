package binanceAPI

import (
	"context"
	"github.com/adshao/go-binance/v2"
)

type Client struct {
	apiKey    string
	secretKey string
	client    *binance.Client
}

func NewClient(apiKey, secretKey string) *Client {
	return &Client{
		apiKey:    apiKey,
		secretKey: secretKey,
		client:    binance.NewClient(apiKey, secretKey),
	}
}

func (c *Client) GetAccountInfo() (*binance.Account, error) {
	account, err := c.client.NewGetAccountService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	return account, nil
}
