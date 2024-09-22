package main

import (
	"context"
	"log"
	"net"
	"sync"

	chat "github.com/christophekede/maxvalue/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type ChatServer struct {
	chat.UnimplementedChatServiceServer
	mu           sync.Mutex // Sync variables acces
	counterId    int32      // Count Id message
	maxValue     int32
	maxMessageId int32
	clients      map[string]chat.ChatService_StreamMessagesServer
}

// Return current @MessageID to client
func (s *ChatServer) GetMessageID(ctx context.Context, req *chat.Empty) (*chat.MessageID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counterId += 1
	return &chat.MessageID{
		Id: s.counterId,
	}, nil
}

func (s *ChatServer) StreamMessages(stream chat.ChatService_StreamMessagesServer) error {
	uid := s.addClient(stream)

	//Clear clients Map when stream close
	defer func() {
		s.mu.Lock()
		delete(s.clients, uid)
		s.mu.Unlock()
		log.Printf("client uid %s disconnected", uid)
	}()

	for {
		clientMessage, err := stream.Recv()

		if err != nil {
			log.Printf("Error in client message: %v", err)
			return err
		}

		log.Printf("Message received from client: MessageID=%d, num=%d", clientMessage.Id, clientMessage.Num)

		isClientValueIsBigger := s.evaluateClientNum(clientMessage)
		if isClientValueIsBigger {
			s.updateCurrentMax(clientMessage.Id, clientMessage.Num)
		}

		serverMessage := &chat.ServerMessage{
			MessageID: s.maxMessageId,
			Max:       s.maxValue,
		}

		if isClientValueIsBigger {
			SendMessageToAllClients(s.clients, serverMessage) //Notify all clients, new max value
		} else {
			SendMessageToClient(stream, serverMessage)
		}
	}
}

func SendMessageToAllClients(clients map[string]chat.ChatService_StreamMessagesServer, serverMessage *chat.ServerMessage) error {
	for _, client := range clients {
		if err := client.Send(serverMessage); err != nil {
			log.Printf("broadcast err: %v", err)
			return err
		}
	}
	return nil
}

func SendMessageToClient(client chat.ChatService_StreamMessagesServer, serverMessage *chat.ServerMessage) error {
	if err := client.Send(serverMessage); err != nil {
		return err
	}
	return nil
}

func (s *ChatServer) addClient(stream chat.ChatService_StreamMessagesServer) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	uid := uuid.Must(uuid.NewRandom()).String()
	log.Printf("New client connected %s", uid)

	s.clients[uid] = stream

	return uid
}

func (s *ChatServer) evaluateClientNum(clientMessage *chat.ClientMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientValue := clientMessage.Num
	return clientValue > s.maxValue
}

func (s *ChatServer) updateCurrentMax(messageID int32, num int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxMessageId = messageID
	s.maxValue = num
	return nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")

	if err != nil {
		log.Fatalf("tcp connection failed: %v", err)
	}
	log.Printf("listening at %v", lis.Addr())

	grpcServer := grpc.NewServer()

	chat.RegisterChatServiceServer(grpcServer, &ChatServer{
		clients:   make(map[string]chat.ChatService_StreamMessagesServer),
		counterId: -1,
	})
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("grpc server failed: %v", err)
	}
}
