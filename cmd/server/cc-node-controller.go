package main

import (
	"cc-node-controller-simple/pkg/sysfeatures"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	ccmetric "github.com/ClusterCockpit/cc-metric-collector/pkg/ccMetric"
	lp "github.com/influxdata/line-protocol" // MIT license

	"github.com/nats-io/nats.go"
)

type NatsConnection struct {
	conn *nats.Conn
	sub  *nats.Subscription
	ch   chan *nats.Msg
}

type NatsConfig struct {
	Hostname      string `json:"hostname"`
	Port          int    `json:"port"`
	SubjectPrefix string `json:"subject_prefix"`
	subject       string
}

func ConnectNats(config NatsConfig) (NatsConnection, error) {
	c := NatsConnection{
		conn: nil,
		sub:  nil,
		ch:   nil,
	}
	uri := fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	fmt.Println("connecting to", uri)
	conn, err := nats.Connect(uri)
	if err != nil {
		return c, err
	}

	ch := make(chan *nats.Msg)
	fmt.Println("subscribing to", config.subject)
	sub, err := conn.ChanSubscribe(config.subject, ch)
	if err != nil {
		return c, err
	}
	c.conn = conn
	c.ch = ch
	c.sub = sub
	return c, nil
}

func DisconnectNats(conn NatsConnection) {
	fmt.Println("disconnecting ...")
	conn.sub.Unsubscribe()
	close(conn.ch)
	conn.conn.Close()
}

func FromLineProtocol(metric string) (ccmetric.CCMetric, error) {
	handler := lp.NewMetricHandler()
	parser := lp.NewParser(handler)

	m, err := parser.Parse([]byte(metric))
	if err != nil {
		return nil, err
	}

	cc := ccmetric.FromInfluxMetric(m[0])
	return cc, nil
}

func ProcessCommand(metric ccmetric.CCMetric) string {
	if metric.HasTag("knob") && metric.HasTag("type") && metric.HasTag("method") {
		fmt.Println("Processing", metric)
		t, ok := metric.GetTag("type")
		if !ok {
			return ""
		}
		stid, ok := metric.GetTag("type-id")
		if !ok {
			return ""
		}
		tid, err := strconv.ParseInt(stid, 10, 64)
		if err != nil {
			return ""
		}
		knob, ok := metric.GetTag("knob")
		if !ok {
			return ""
		}
		method, ok := metric.GetTag("method")
		if !ok {
			return ""
		}
		if method == "PUT" {
			value, ok := metric.GetField("value")
			if !ok {
				return ""
			}
			v := fmt.Sprintf("%d", value)

			dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
			if err != nil {
				return ""
			}
			sysfeatures.SysFeaturesSetDevice(knob, dev, v)
		} else {
			dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
			if err != nil {
				return ""
			}
			value, err := sysfeatures.SysFeaturesGetDevice(knob, dev)
			if err != nil {
				return ""
			}
			fmt.Println(knob, value)
			return value
		}
	}
	return ""
}

func ReadCli() map[string]string {
	var m map[string]string
	cfg := flag.String("config", "./config.json", "Path to configuration file")
	logfile := flag.String("log", "stderr", "Path for logfile")
	debug := flag.Bool("debug", false, "Activate debug output")
	flag.Parse()
	m = make(map[string]string)
	m["configfile"] = *cfg
	m["logfile"] = *logfile
	if *debug {
		m["debug"] = "true"
	} else {
		m["debug"] = "false"
	}
	return m
}

func LoadConfiguration(filename string) (NatsConfig, error) {
	var c NatsConfig = NatsConfig{
		Hostname:      "",
		Port:          0,
		SubjectPrefix: "",
	}
	configFile, err := os.Open(filename)
	if err != nil {
		return c, err
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&c)
	return c, err
}

func real_main() int {

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	cli_opts := ReadCli()
	if len(cli_opts["configfile"]) == 0 {
		cli_opts["configfile"] = "./config.json"
	}

	config, err := LoadConfiguration(cli_opts["configfile"])
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}
	if len(config.SubjectPrefix) > 0 {
		config.subject = fmt.Sprintf("%s/%s", config.SubjectPrefix, hostname)
	} else {
		config.subject = hostname
	}

	err = sysfeatures.SysFeaturesInit()
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	conn, err := ConnectNats(config)
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt)
	signal.Notify(shutdownSignal, syscall.SIGTERM)

global_for:
	for {
		select {
		case <-shutdownSignal:
			fmt.Println("got interrupt, exiting...")
			break global_for
		case msg := <-conn.ch:
			data := string(msg.Data)
			for _, line := range strings.Split(data, "\n") {
				m, err := FromLineProtocol(line)
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				if m.Name() != hostname {
					continue
				}

				ProcessCommand(m)
			}
		}
	}

	DisconnectNats(conn)
	sysfeatures.SysFeaturesClose()
	return 0
}

func main() {
	ret := real_main()
	os.Exit(ret)
}
