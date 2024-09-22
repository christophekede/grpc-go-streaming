package main

import (
	"context"
	"log"
	"math/rand/v2"
	"time"

	chat "github.com/christophekede/maxvalue/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Get message id from server
func getSyncMessageId(client chat.ChatServiceClient) int32 {
	messageID, err := client.GetMessageID(context.Background(), &chat.Empty{})
	if err != nil {
		log.Fatalf("could not get message ID: %v", err)
	}
	return messageID.Id
}

func generateRandomInt32InRange(min int32, max int32) int32 {
	return rand.Int32N(max-min+1) + min
}

func main() {
	conn, err := grpc.NewClient(":50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("problem with the server: %v", err)
	}

	defer conn.Close()
	client := chat.NewChatServiceClient(conn)

	stream, err := client.StreamMessages(context.Background())
	if err != nil {
		log.Fatalf("could not start stream: %v", err)
	}
	log.Print("Client connected to server")

	go func() {

		for {

			msg := &chat.ClientMessage{
				Id:  getSyncMessageId(client),
				Num: generateRandomInt32InRange(-10000, 10000),
			}

			if err := stream.Send(msg); err != nil {
				log.Printf("failed to send message: %v", err)
				return
			}
			log.Printf("Client sent: ID=%d, Value=%d", msg.Id, msg.Num)

			time.Sleep(time.Duration(generateRandomInt32InRange(1, 5)) * time.Second)

		}

	}()

	// Receive messages from the server
	for {
		serverMessage, err := stream.Recv()
		if err != nil {
			log.Printf("failed to receive message: %v", err)
			break
		}
		log.Printf("Server: MessageID=%d, Max=%d", serverMessage.MessageID, serverMessage.Max)
	}

}
