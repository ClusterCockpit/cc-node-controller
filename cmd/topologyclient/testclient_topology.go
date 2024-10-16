package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	const timeout = 2
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	pubsubject := fmt.Sprintf("cc-control.%s", hostname)
	commands := []string{
		fmt.Sprintf("topology,hostname=%s,method=GET,type=node,type-id=0 value=0.0", hostname),
	}

	conn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer conn.Close()

	for _, c := range commands {
		fmt.Printf("Publishing to %s: %s\n", pubsubject, c)
		msg, err := conn.Request(pubsubject, []byte(c), time.Second * timeout)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("Got reply: %s\n", msg.Data)
		}
	}
}
