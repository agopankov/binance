package main

import (
	"encoding/json"
	"github.com/agopankov/imPulse/server/pkg/grpcbinance"
	"github.com/agopankov/imPulse/server/pkg/grpcbinance/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

type SecretKeys struct {
	BinanceAPIKey    string `json:"BINANCE_API_KEY"`
	BinanceSecretKey string `json:"BINANCE_SECRET_KEY"`
}

func main() {
	var secrets SecretKeys

	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		secretsFile, err := os.ReadFile("/mnt/secrets-store/prod_binance_secret")
		if err != nil {
			log.Fatalf("Failed to read secrets file: %v", err)
		}
		err = json.Unmarshal(secretsFile, &secrets)
		if err != nil {
			log.Fatalf("Failed to unmarshal secrets JSON: %v", err)
		}
		apiKey = secrets.BinanceAPIKey
		secretKey = secrets.BinanceSecretKey
	}

	server := grpcbinance.NewBinanceServiceServer(apiKey, secretKey)

	grpcServer := grpc.NewServer()
	proto.RegisterBinanceServiceServer(grpcServer, server)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	log.Println("gRPC server started successfully")
	select {}
}
