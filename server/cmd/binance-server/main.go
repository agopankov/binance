package main

import (
	"github.com/agopankov/binance/server/pkg/grpcbinance"
	"github.com/agopankov/binance/server/pkg/grpcbinance/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")

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
