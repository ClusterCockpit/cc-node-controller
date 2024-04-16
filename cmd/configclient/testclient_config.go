package main

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	pubsubject := "/cc-control/nuc"
	subsubject := "/cc-events/*"
	commands := []string{
		"controls,hostname=nuc,method=GET,type=node,type-id=0 value=0.0",
	}

	conn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer conn.Close()

	for _, c := range commands {
		fmt.Printf("Publishing to %s: %s\n", pubsubject, c)
		conn.Publish(pubsubject, []byte(c))
	}
	fmt.Printf("Subscribing to %s\n", subsubject)
	_, err = conn.Subscribe(subsubject, func(msg *nats.Msg) {
		fmt.Println(string(msg.Data))
	})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("Waiting for response\n")
	time.Sleep(2 * time.Second)

}
