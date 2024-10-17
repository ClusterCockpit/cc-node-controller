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
		fmt.Sprintf("cpu_freq.cur_cpu_freq,hostname=%s,method=GET,type=hwthread,type-id=0 value=0.0", hostname),
		fmt.Sprintf("rapl.pkg_energy,hostname=%s,method=GET,type=socket,type-id=0 value=0.0", hostname),
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
			fmt.Printf("Got reply: %s\n", msg.Data)
		}
	}
}
