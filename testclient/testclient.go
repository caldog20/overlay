package testclient

import (
	"context"
	"log"

	"github.com/caldog20/go-overlay/msg"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Run(ctx context.Context) {
	log.SetPrefix("client: ")
	conn, err := grpc.DialContext(ctx, "localhost:9000", grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("error connecting to grpc server: %v", err)
	}
	defer conn.Close()

	client := msg.NewControlServiceClient(conn)

	for {
		response, rerr := client.Register(ctx, &msg.RegisterRequest{User: "simp"})
		if rerr != nil {
			log.Printf("error sending/recv message: %v", rerr)
			return
		}

		log.Println(response)
	}
}
