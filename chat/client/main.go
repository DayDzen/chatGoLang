package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/DayDzen/chatGoLang/chat/chatpb"

	"google.golang.org/grpc"
)

var client chatpb.BroadcastClient
var wait *sync.WaitGroup

func init() {
	wait = &sync.WaitGroup{}
}

func connect(user *chatpb.User) error {
	var streamError error
	fmt.Println(user)

	stream, err := client.CreateStream(context.Background(), &chatpb.Connect{
		User:   user,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("Connect failed: %v", err)
	}

	wait.Add(1)
	go func(str chatpb.Broadcast_CreateStreamClient) {
		defer wait.Done()

		for {
			msg, err := str.Recv()

			if err != nil {
				streamError = fmt.Errorf("Error reading message: %v", err)
				break
			}

			fmt.Printf("%v : %s\n", msg.User.Name, msg.Content)
		}
	}(stream)

	return streamError
}

func main() {
	fmt.Println("New client is started")

	timestamp := time.Now()
	done := make(chan int)

	userName := flag.String("N", "Anon", "The name of the user")
	chatName := flag.String("C", "General", "The chat name")
	flag.Parse()

	userID := sha256.Sum256([]byte(timestamp.String() + *userName))
	chatID := sha256.Sum256([]byte(*chatName))

	chat := &chatpb.Chat{
		Id:   hex.EncodeToString(chatID[:]),
		Name: *chatName,
	}

	user := &chatpb.User{
		Id:   hex.EncodeToString(userID[:]),
		Name: *userName,
		Chat: chat,
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	client = chatpb.NewBroadcastClient(conn)

	connect(user)

	wait.Add(1)
	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)

		ts := time.Now()
		msgID := sha256.Sum256([]byte(ts.String() + *userName))

		for scanner.Scan() {
			msg := &chatpb.Message{
				Id:        hex.EncodeToString(msgID[:]),
				User:      user,
				Content:   scanner.Text(),
				Timestamp: ts.String(),
			}

			_, err := client.BroadcastMessage(context.Background(), msg)
			if err != nil {
				log.Fatalf("Error Sending Message: %v\n", err)
				break
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
}
