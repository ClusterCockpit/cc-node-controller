package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	cccontrol "github.com/ClusterCockpit/cc-node-controller/pkg/ccControlClient"
)

func ReadCli() map[string]interface{} {
	server := flag.String("server", "127.0.0.1", "IP or hostname of NATS server")
	port := flag.Int("port", 4222, "Port of NATS server")
	debug := flag.Bool("debug", false, "Activate debug output")
	topo := flag.Bool("topology", false, "List topology of remote node")
	config := flag.Bool("list", false, "List controls of remote node")
	get := flag.String("get", "", "Get value of control from remote node (name@type-typeid)")
	set := flag.String("set", "", "Set value of control from remote node (name@type-typeid=value)")
	host := flag.String("host", "", "Hostname of remote node")

	flag.Parse()
	m := make(map[string]interface{})
	m["server"] = *server
	m["get"] = *get
	m["set"] = *set
	m["port"] = *port
	m["host"] = *host
	if *debug {
		m["debug"] = true
	} else {
		m["debug"] = false
	}
	if *topo {
		m["topology"] = true
	} else {
		m["topology"] = false
	}
	if *config {
		m["config"] = true
	} else {
		m["config"] = false
	}
	return m
}

func main() {
	getregex := regexp.MustCompile(`^([a-z0-9\._]+)@([a-z]+)-([0-9]+)`)
	setregex := regexp.MustCompile(`^([a-z0-9\._]+)@([a-z]+)-([0-9]+)=(.+)$`)
	cliopts := ReadCli()

	c, err := cccontrol.NewCCControlClient(cliopts["server"].(string), cliopts["port"].(int), "cc-events", "cc-control")
	if err != nil {
		fmt.Println(err.Error())
	}
	defer c.Close()
	if len(cliopts["host"].(string)) == 0 {
		fmt.Println("-host <hostname> required")
		os.Exit(1)
	}
	if (!cliopts["topology"].(bool)) && (!cliopts["config"].(bool)) && len(cliopts["get"].(string)) == 0 && len(cliopts["set"].(string)) == 0 {
		fmt.Println("Either -topology, -config, -get <control> or -set <control>=<value> required")
		os.Exit(1)
	}

	if cliopts["topology"].(bool) {
		t, err := c.GetTopology(cliopts["host"].(string))
		if err != nil {
			fmt.Printf("Failed to get topology of node %s: %v\n", cliopts["host"].(string), err.Error())
			os.Exit(1)
		}
		clist := make([]string, 0)
		for _, c := range t.HWthreads {
			clist = append(clist, fmt.Sprintf("%d", c.CpuID))
		}
		fmt.Printf("NumHWThreads: %d\n", t.CpuInfo.NumHWthreads)
		fmt.Printf("HWThreads: %s\n", strings.Join(clist, ","))
		os.Exit(0)
	}

	if cliopts["config"].(bool) {
		c, err := c.GetControls(cliopts["host"].(string))
		if err != nil {
			fmt.Printf("Failed to get topology of node %s: %v\n", cliopts["host"].(string), err.Error())
			os.Exit(1)
		}
		for _, ctrl := range c.Controls {
			fmt.Printf("%s.%s for type=%s (%s): %s\n", ctrl.Category, ctrl.Name, ctrl.DeviceType, ctrl.Methods, ctrl.Description)
		}

		os.Exit(0)
	}

	if len(cliopts["get"].(string)) > 0 {
		fmt.Println("Executing GET path")

		rematch := getregex.FindStringSubmatch(cliopts["get"].(string))
		fmt.Println(strings.Join(rematch, " | "))
		if len(rematch) == 4 {
			v, err := c.GetControlValue(cliopts["host"].(string), rematch[1], rematch[2], rematch[3])
			if err != nil {
				fmt.Printf("Failed to get control %s of node %s: %v\n", rematch[1], cliopts["host"].(string), err.Error())
				os.Exit(1)
			}
			fmt.Printf("Control %s at host %s (%s-%s): %s\n", rematch[0], cliopts["host"].(string), rematch[2], rematch[3], v)
			os.Exit(0)
		} else {
			fmt.Printf("Failed to parse control %s\n", cliopts["get"].(string))
			os.Exit(1)
		}
	}
	if len(cliopts["set"].(string)) > 0 {
		fmt.Println("Executing SET path")
		rematch := setregex.FindStringSubmatch(cliopts["set"].(string))
		if len(rematch) == 5 {
			err := c.SetControlValue(cliopts["host"].(string), rematch[1], rematch[2], rematch[3], rematch[4])
			if err != nil {
				fmt.Printf("Failed to set control %s of node %s: %v\n", rematch[1], cliopts["host"].(string), err.Error())
				os.Exit(1)
			}
			fmt.Printf("Control %s set at host %s (%s%s): %s", rematch[0], cliopts["host"].(string), rematch[2], rematch[3], rematch[4])
			os.Exit(0)
		}
	}
	fmt.Println("Nothing executed, something is wrong")
	os.Exit(1)
}
