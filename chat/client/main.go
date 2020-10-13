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
	var streamerror error

	stream, err := client.CreateStream(context.Background(), &chatpb.Connect{
		User:   user,
		Active: true,
	})

	if err != nil {
		log.Fatalf("Connection filed: %v", err)
	}

	wait.Add(1)
	go func(str chatpb.Broadcast_CreateStreamClient) {
		defer wait.Done()

		for {
			msg, err := str.Recv()

			if err != nil {
				log.Fatalf("Failed recieving msg from Server: %v", err)
				break
			}

			fmt.Printf("%v : %s\n", msg.Id, msg.Message)
		}
	}(stream)

	return streamerror
}

func main() {
	fmt.Println("Hello, I'm a client")

	timestamp := time.Now()
	done := make(chan int)

	name := flag.String("N", "Anon", "The name of the user")
	flag.Parse()

	id := sha256.Sum256([]byte(timestamp.String() + *name))

	cc, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer cc.Close()

	client = chatpb.NewBroadcastClient(cc)
	user := &chatpb.User{
		Id:          hex.EncodeToString(id[:]),
		DisplayName: *name,
	}

	connect(user)

	wait.Add(1)
	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := &chatpb.Message{
				Id:        user.Id,
				Message:   scanner.Text(),
				Timestamp: timestamp.String(),
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
