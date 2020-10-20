package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/DayDzen/chatGoLang/chat/chatpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

// Server is structure for creating server
type Server struct {
	connection []*Connection
}

// Connection is structure for creating new connections
type Connection struct {
	stream chatpb.Broadcast_CreateStreamServer
	userID string
	chatID string
	active bool
	err    chan error
}

// CreateStream is implementing of interface from chat.pb.go file
func (s *Server) CreateStream(pconn *chatpb.Connect, stream chatpb.Broadcast_CreateStreamServer) error {
	conn := &Connection{
		stream: stream,
		userID: pconn.GetUser().GetId(),
		chatID: pconn.GetUser().GetChat().GetId(),
		active: true,
		err:    make(chan error),
	}
	s.connection = append(s.connection, conn)

	return <-conn.err
}

//BroadcastMessage is implementation of interface from chat.pb.go file
func (s *Server) BroadcastMessage(ctx context.Context, msg *chatpb.Message) (*chatpb.Close, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, conn := range s.connection {
		wait.Add(1)

		go func(msg *chatpb.Message, conn *Connection) {
			defer wait.Done()

			if conn.active {
				if msg.GetUser().GetChat().GetId() == conn.chatID {
					err := conn.stream.Send(msg)
					log.Printf("User's ID is: %v\n", conn.userID)
					log.Printf("Chat's ID is: %v\n", conn.chatID)
					log.Printf("Msg is: %v\n\n", msg.GetContent())
					// grpclog.Infof("Sending message %v by user %v", msg.GetId(), conn.chatID)

					if err != nil {
						grpclog.Errorf("Error with stream: %v - Error: %v", conn.stream, err)
						conn.active = false
						conn.err <- err
					}
				}
			}
		}(msg, conn)
	}
	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
	return &chatpb.Close{}, nil
}

func main() {
	var connections []*Connection
	server := &Server{connections}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("Chat Service started")

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed while creating listener: %v", err)
	}

	s := grpc.NewServer()
	chatpb.RegisterBroadcastServer(s, server)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to server: %v", err)
	}
}
