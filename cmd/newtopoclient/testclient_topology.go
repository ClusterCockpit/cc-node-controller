package main

import (
	"fmt"

	cccc "github.com/ClusterCockpit/cc-node-controller/pkg/ccControlClient"
)

func main() {
	conn, err := cccc.NewCCControlClient("localhost", 4222, "cc-events", "cc-control")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer conn.Close()

	topo, err := conn.GetTopology("ivyep1")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("Got topology with %d HW threads\n", len(topo.HWthreads))
	}
}
