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
		fmt.Sprintf("controls,hostname=%s,method=GET,type=node,type-id=0 value=0.0", hostname),
	}

	conn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer conn.Close()

	for _, c := range commands {
		fmt.Printf("Requesting to %s: %s\n", pubsubject, c)
		msg, err := conn.Request(pubsubject, []byte(c), time.Second * timeout)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("Received reply: %s\n", string(msg.Data))
		}
	}
}
