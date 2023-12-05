package main

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

func main() {

	commands := []string{
		"nuc,knob=cpu_freq.base_freq,method=GET,type=hwthread,type-id=0 value=0.0",
	}

	conn, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer conn.Close()

	for _, c := range commands {
		conn.Publish("/cc-control/nuc", []byte(c))
	}

}
