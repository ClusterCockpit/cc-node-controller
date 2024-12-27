package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ClusterCockpit/cc-node-controller/pkg/sysfeatures"

	lp "github.com/ClusterCockpit/cc-energy-manager/pkg/cc-message"
	cclog "github.com/ClusterCockpit/cc-metric-collector/pkg/ccLogger"
	// MIT license
)

var cc_node_control_hostname string = ""

func FromLineProtocol(metric string) (lp.CCMessage, error) {

	list, err := lp.FromBytes([]byte(metric))
	if err != nil {
		return nil, err
	}
	if len(list) != 1 {
		return nil, errors.New("string contains mutliple metrics")
	}
	return list[0], nil
}

func ProcessCommand(input lp.CCMessage) (lp.CCMessage, error) {

	createOutput := func(errorString string) (lp.CCMessage, error) {
		resp, err := lp.NewLog(input.Name(), input.Tags(), input.Meta(), errorString, time.Now())
		if err == nil {
			resp.AddTag("level", "ERROR")
			return resp, nil
		}
		return nil, fmt.Errorf("%s and cannot send response", errorString)
	}

	var tid int64 = 0
	var err error = nil
	cclog.ComponentDebug("Sysfeatures", "Processing", input.ToLineProtocol(nil))
	knob := input.Name()
	t, ok := input.GetTag("type")
	if !ok {
		return createOutput(fmt.Sprintf("No 'type' tag in %s", input.ToLineProtocol(nil)))
	}
	if t != "node" {
		stid, ok := input.GetTag("type-id")
		if !ok {
			return createOutput(fmt.Sprintf("No 'type-id' tag in %s", input.ToLineProtocol(nil)))
		}
		tid, err = strconv.ParseInt(stid, 10, 64)
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot parse 'type-id' tag in %s", input.ToLineProtocol(nil)))
		}
	}
	method, ok := input.GetTag("method")
	if !ok {
		return createOutput(fmt.Sprintf("No 'method' tag in %s", input.ToLineProtocol(nil)))
	}
	if method != "PUT" && method != "GET" {
		return createOutput(fmt.Sprintf("Invalid 'method' tag in %s", input.ToLineProtocol(nil)))
	}
	if method == "PUT" {
		value, ok := input.GetField("value")
		if !ok {
			return createOutput(fmt.Sprintf("No 'value' field in %s", input.ToLineProtocol(nil)))
		}
		v := fmt.Sprintf("%d", value)
		cclog.ComponentDebug("Sysfeatures", "Creating device", t, " ", int(tid))
		dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d", t, tid))
		}
		cclog.ComponentDebug("Sysfeatures", "Set", knob, "for device", t, " ", int(tid), "to", v)
		err = sysfeatures.SysFeaturesSetDevice(knob, dev, v)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to set %s=%s for device %s%d", knob, v, t, tid))
		}
	} else if method == "GET" {
		cclog.ComponentDebug("Sysfeatures", "Creating device", t, " ", int(tid))
		dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d", t, tid))
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", t, " ", int(tid))
		value, err := sysfeatures.SysFeaturesGetDevice(knob, dev)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to get %s for device %s%d", knob, t, tid))
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", t, " ", int(tid), "returned", value)
		resp, err := createOutput(value)
		if err == nil {
			resp.AddTag("level", "INFO")
		}
		return resp, nil
	}
	return createOutput(fmt.Sprintf("Invalid 'method' tag in %s", input.ToLineProtocol(nil)))
}

func ReadCli() map[string]string {
	var m map[string]string
	cfg := flag.String("config", "./config.json", "Path to configuration file")
	logfile := flag.String("log", "stderr", "Path for logfile")
	debug := flag.Bool("debug", false, "Activate debug output")
	pretend := flag.Bool("pretend", false, "Do not actually do anything")
	flag.Parse()
	m = make(map[string]string)
	m["configfile"] = *cfg
	m["logfile"] = *logfile
	if *debug {
		m["debug"] = "true"
	} else {
		m["debug"] = "false"
	}
	if *pretend {
		m["pretend"] = "true"
	} else {
		m["pretend"] = "false"
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
		OutstandingMessages: 1000,
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
	cc_node_control_hostname = hostname

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
			config.subject = fmt.Sprintf("%s.%s", config.InputSubjectPrefix, hostname)
		} else {
			config.subject = hostname
		}
	}
	cclog.ComponentDebug("CONFIG", "Using input subject", config.subject)
	if len(config.OutputSubject) > 0 {
		config.outSubject = config.OutputSubject
	} else {
		if len(config.OutputSubjectPrefix) > 0 {
			config.outSubject = fmt.Sprintf("%s.%s", config.OutputSubjectPrefix, hostname)
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
			//data := string(msg.Data)
			data, err := lp.FromBytes(msg.Data)
			if err == nil {
				for _, m := range data {
					var r lp.CCMessage = nil
					select {
					case <-shutdownSignal:
						cclog.ComponentDebug("LOOP", "got interrupt, exiting...")
						break global_for
					default:
						if h, ok := m.GetTag("hostname"); ok && h != hostname {
							cclog.ComponentDebug("LOOP", "Non-local command, skipping...")
							continue
						}
						cclog.ComponentDebug("LOOP", "processing", m.String())
						switch m.Name() {
						case "topology":
							cclog.ComponentDebug("LOOP", "Got topology message")
							r, err = ProcessTopologyConfig(m)
							if err != nil {
								cclog.Error(err.Error())
							}
						case "controls":
							cclog.ComponentDebug("LOOP", "Got controls message")
							r, err = ProcessControlsConfig(m)
							if err != nil {
								cclog.Error(err.Error())
							}
						default:
							r, err = ProcessCommand(m)
							if err != nil {
								cclog.Error(err.Error())
							}
						}
						if r != nil {
							r.AddTag("hostname", cc_node_control_hostname)
							cclog.ComponentDebug("LOOP", "sending response", r.ToLineProtocol(nil))
							msg.Respond([]byte(r.ToLineProtocol(nil)))
						}
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
