package main

import (
	"context"
	"log"
	"time"

	"github.com/temmyjay001/fabricx-core/protos"
	"google.golang.org/grpc"
)

func main() {
	time.Sleep(3 * time.Second)
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := protos.NewFabricXClient(conn)

	r, err := c.InitNetwork(context.Background(), &protos.InitNetworkRequest{})
	if err != nil {
		log.Fatalf("could not init network: %v", err)
	}
	log.Printf("Response: %s", r.Message)
}
