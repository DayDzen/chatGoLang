package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/DayDzen/chatGoLang/chat/chatpb"

	"google.golang.org/grpc"
)

// Server is structure for creating server
type Server struct {
	connection []*Connection
}

// Connection is structure for creating new connections
type Connection struct {
	stream chatpb.Broadcast_CreateStreamServer
	id     string
	active bool
	err    chan error
}

// CreateStream is implementing of interface from chat.pb.go file
func (s *Server) CreateStream(pconn *chatpb.Connect, stream chatpb.Broadcast_CreateStreamServer) error {
	conn := &Connection{
		stream: stream,
		id:     pconn.User.Id,
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
				err := conn.stream.Send(msg)
				log.Printf("Sending message to: %s", conn.stream)

				if err != nil {
					log.Fatalf("Error with stream: %s - Error: %v", conn.stream, err)
					conn.active = false
					conn.err <- err
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