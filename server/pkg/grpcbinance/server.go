package grpcbinance

import (
	"context"
	"github.com/adshao/go-binance/v2"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"strconv"
	"strings"
)

type BinanceServiceServer struct {
	proto.UnimplementedBinanceServiceServer
	client *binance.Client
}

func NewBinanceServiceServer(apiKey, secretKey string) *BinanceServiceServer {
	return &BinanceServiceServer{
		client: binance.NewClient(apiKey, secretKey),
	}
}

func (s *BinanceServiceServer) GetUSDTPrices(ctx context.Context, _ *proto.Empty) (*proto.USDTPricesResponse, error) {
	prices, err := s.client.NewListPricesService().Do(ctx)
	if err != nil {
		return nil, err
	}

	usdtPrices := make([]*proto.USDTPrice, 0)
	for _, price := range prices {
		if strings.HasSuffix(price.Symbol, "USDT") {
			priceFloat, _ := strconv.ParseFloat(price.Price, 64)
			usdtPrice := &proto.USDTPrice{
				Symbol: price.Symbol,
				Price:  priceFloat,
			}
			usdtPrices = append(usdtPrices, usdtPrice)
		}
	}

	response := &proto.USDTPricesResponse{
		Prices: usdtPrices,
	}
	return response, nil
}

func (s *BinanceServiceServer) Get24HChangePercent(ctx context.Context, _ *proto.Empty) (*proto.ChangePercentResponse, error) {
	ticker24h, err := s.client.NewListPriceChangeStatsService().Do(ctx)
	if err != nil {
		return nil, err
	}

	changePercents := make([]*proto.ChangePercent, 0)
	for _, ticker := range ticker24h {
		change, err := strconv.ParseFloat(ticker.PriceChangePercent, 64)
		if err != nil {
			continue
		}
		changePercent := &proto.ChangePercent{
			Symbol:        ticker.Symbol,
			ChangePercent: change,
		}
		changePercents = append(changePercents, changePercent)
	}

	response := &proto.ChangePercentResponse{
		ChangePercents: changePercents,
	}
	return response, nil
}
