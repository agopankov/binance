package binanceAPI

import (
	"context"
	"github.com/adshao/go-binance/v2"
	"strconv"
	"strings"
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

func (c *Client) GetUSDTPrices() (map[string]float64, error) {
	prices, err := c.client.NewListPricesService().Do(context.Background())
	if err != nil {
		return nil, err
	}

	usdtPrices := make(map[string]float64)
	for _, price := range prices {
		if strings.HasSuffix(price.Symbol, "USDT") {
			priceFloat, _ := strconv.ParseFloat(price.Price, 64)
			usdtPrices[price.Symbol] = priceFloat
		}
	}

	return usdtPrices, nil
}

func (c *Client) Get24hChangePercent() (map[string]float64, error) {
	ticker24h, err := c.client.NewListPriceChangeStatsService().Do(context.Background())
	if err != nil {
		return nil, err
	}

	changePercent := make(map[string]float64)
	for _, ticker := range ticker24h {
		change, err := strconv.ParseFloat(ticker.PriceChangePercent, 64)
		if err != nil {
			continue
		}
		changePercent[ticker.Symbol] = change
	}
	return changePercent, nil
}
