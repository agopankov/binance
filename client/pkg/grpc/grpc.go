package grpc

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewGRPCConnection(serverAddress string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
