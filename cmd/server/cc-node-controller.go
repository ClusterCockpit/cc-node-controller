package main

import (
	"cc-node-controller-simple/pkg/sysfeatures"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	cclog "github.com/ClusterCockpit/cc-metric-collector/pkg/ccLogger"
	ccmetric "github.com/ClusterCockpit/cc-metric-collector/pkg/ccMetric"
	lp "github.com/influxdata/line-protocol" // MIT license

	"github.com/nats-io/nats.go"
)

type NatsConnection struct {
	conn       *nats.Conn
	sub        *nats.Subscription
	ch         chan *nats.Msg
	outSubject string
}

type NatsConfig struct {
	Hostname            string `json:"hostname"`
	Port                int    `json:"port"`
	Username            string `json:"username,omitempty"`
	Password            string `json:"password,omitempty"`
	InputSubjectPrefix  string `json:"input_subject_prefix,omitempty"`
	InputSubject        string `json:"input_subject,omitempty"`
	OutputSubjectPrefix string `json:"output_subject_prefix,omitempty"`
	OutputSubject       string `json:"output_subject,omitempty"`
	subject             string
	outSubject          string
}

func ConnectNats(config NatsConfig) (NatsConnection, error) {
	c := NatsConnection{
		conn:       nil,
		sub:        nil,
		ch:         nil,
		outSubject: config.outSubject,
	}
	uri := fmt.Sprintf("%s:%d", config.Hostname, config.Port)
	cclog.ComponentDebug("NATS", "connecting to", uri)
	conn, err := nats.Connect(uri)
	if err != nil {
		return c, err
	}

	ch := make(chan *nats.Msg)
	cclog.ComponentDebug("NATS", "subscribing to", config.subject)
	sub, err := conn.ChanSubscribe(config.subject, ch)
	if err != nil {
		return c, err
	}
	c.conn = conn
	c.ch = ch
	c.sub = sub
	return c, nil
}

func PublishNats(conn NatsConnection, event ccmetric.CCMetric) error {
	cclog.ComponentDebug("NATS", "Publish ", conn.outSubject, ": ", event.String())
	return conn.conn.Publish(conn.outSubject, []byte(event.String()))
}

func DisconnectNats(conn NatsConnection) {
	cclog.ComponentDebug("NATS", "disconnecting ...")
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

func ProcessCommand(input ccmetric.CCMetric) (ccmetric.CCMetric, error) {

	createOutput := func(errorString string, tags map[string]string) (ccmetric.CCMetric, error) {
		resp, err := ccmetric.New("cc-node-controller", tags, map[string]string{}, map[string]interface{}{"value": errorString}, time.Now())
		if err == nil {
			resp.AddTag("level", "ERROR")
			return resp, errors.New(errorString)
		}
		return nil, fmt.Errorf("%s and cannot send response", errorString)
	}

	var tid int64 = 0
	var err error = nil
	cclog.ComponentDebug("Sysfeatures", "Processing", input)
	t, ok := input.GetTag("type")
	if !ok {
		return createOutput(fmt.Sprintf("No 'type' tag in %s", input), input.Tags())
	}
	if t != "node" {
		stid, ok := input.GetTag("type-id")
		if !ok {
			return createOutput(fmt.Sprintf("No 'type-id' tag in %s", input), input.Tags())
		}
		tid, err = strconv.ParseInt(stid, 10, 64)
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot parse 'type-id' tag in %s", input), input.Tags())
		}
	}
	knob, ok := input.GetTag("knob")
	if !ok {
		return createOutput(fmt.Sprintf("No 'knob' tag in %s", input), input.Tags())
	}
	method, ok := input.GetTag("method")
	if !ok {
		return createOutput(fmt.Sprintf("No 'method' tag in %s", input), input.Tags())
	}
	if method != "PUT" && method != "GET" {
		return createOutput(fmt.Sprintf("Invalid 'method' tag in %s", input), input.Tags())
	}
	if method == "PUT" {
		value, ok := input.GetField("value")
		if !ok {
			return createOutput(fmt.Sprintf("No 'value' field in %s", input), input.Tags())
		}
		v := fmt.Sprintf("%d", value)
		cclog.ComponentDebug("Sysfeatures", "Creating device", t, " ", int(tid))
		dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d", t, tid), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Set", knob, "for device", t, " ", int(tid), "to", v)
		err = sysfeatures.SysFeaturesSetDevice(knob, dev, v)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to set %s=%s for device %s%d", knob, v, t, tid), input.Tags())
		}
	} else if method == "GET" {
		cclog.ComponentDebug("Sysfeatures", "Creating device", t, " ", int(tid))
		dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d", t, tid), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", t, " ", int(tid))
		value, err := sysfeatures.SysFeaturesGetDevice(knob, dev)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to get %s for device %s%d", knob, t, tid), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", t, " ", int(tid), "returned", value)
		resp, err := createOutput(value, input.Tags())
		if err == nil {
			resp.AddTag("level", "INFO")
		}
		return resp, err
	}
	return createOutput(fmt.Sprintf("Invalid 'method' tag in %s", input), input.Tags())
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
		Hostname:            "",
		Port:                0,
		InputSubjectPrefix:  "",
		InputSubject:        "",
		OutputSubjectPrefix: "",
		OutputSubject:       "",
		subject:             "",
		outSubject:          "",
	}
	configFile, err := os.Open(filename)
	if err != nil {
		cclog.Error(err.Error())
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
	if cli_opts["debug"] == "true" {
		cclog.SetDebug()
	}
	cclog.SetOutput(cli_opts["logfile"])

	config, err := LoadConfiguration(cli_opts["configfile"])
	if err != nil {
		cclog.Error(err.Error())
		return 1
	}
	if len(config.InputSubject) > 0 {
		config.subject = config.InputSubject
	} else {
		if len(config.InputSubjectPrefix) > 0 {
			config.subject = fmt.Sprintf("%s/%s", config.InputSubjectPrefix, hostname)
		} else {
			config.subject = hostname
		}
	}
	cclog.ComponentDebug("CONFIG", "Using input subject", config.subject)
	if len(config.OutputSubject) > 0 {
		config.outSubject = config.OutputSubject
	} else {
		if len(config.OutputSubjectPrefix) > 0 {
			config.outSubject = fmt.Sprintf("%s/%s", config.OutputSubjectPrefix, hostname)
		} else {
			config.outSubject = fmt.Sprintf("%s-out", hostname)
		}
	}
	cclog.ComponentDebug("CONFIG", "Using output subject", config.outSubject)
	if len(config.subject) == 0 || len(config.outSubject) == 0 {
		cclog.ComponentError("CONFIG", "Failed to get input and output subject for NATS")
		return 1
	}
	cclog.ComponentDebug("CONFIG", "Initializing sysfeatures")
	err = sysfeatures.SysFeaturesInit()
	if err != nil {
		cclog.Error(err.Error())
		return 1
	}
	defer sysfeatures.SysFeaturesClose()

	cclog.ComponentDebug("CONFIG", "Connecting NATS")
	conn, err := ConnectNats(config)
	if err != nil {
		cclog.Error(err.Error())
		return 1
	}
	defer DisconnectNats(conn)

	cclog.ComponentDebug("CONFIG", "Configuring signals")
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt)
	signal.Notify(shutdownSignal, syscall.SIGTERM)

	cclog.ComponentDebug("CONFIG", "Starting Loop")
global_for:
	for {
		select {
		case <-shutdownSignal:
			cclog.ComponentDebug("LOOP", "got interrupt, exiting...")
			break global_for
		case msg := <-conn.ch:
			data := string(msg.Data)
			for _, line := range strings.Split(data, "\n") {
				select {
				case <-shutdownSignal:
					cclog.ComponentDebug("LOOP", "got interrupt, exiting...")
					break global_for
				default:
					cclog.ComponentDebug("LOOP", "parsing", line)
					m, err := FromLineProtocol(line)
					if err != nil {
						cclog.Error(err.Error())
						continue
					}
					if m.Name() != hostname {
						cclog.ComponentDebug("LOOP", "Non-local command, skipping...")
						continue
					}
					cclog.ComponentDebug("LOOP", "processing", line)
					r, err := ProcessCommand(m)
					if err != nil {
						cclog.Error(err.Error())
					}
					if r != nil {
						cclog.ComponentDebug("LOOP", "sending response", r)
						PublishNats(conn, r)
					}
				}

			}
		}
	}

	return 0
}

func main() {
	ret := real_main()
	os.Exit(ret)
}
