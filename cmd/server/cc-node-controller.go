package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClusterCockpit/cc-node-controller/pkg/sysfeatures"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
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

	var deviceId string
	cclog.ComponentDebug("Sysfeatures", "Processing", input.ToLineProtocol(nil))
	knob := input.Name()
	deviceType, ok := input.GetTag("type")
	if !ok {
		return createOutput(fmt.Sprintf("No 'type' tag in %s", input.ToLineProtocol(nil)))
	}
	if deviceType != "node" {
		var ok bool
		deviceId, ok = input.GetTag("type-id")
		if !ok {
			return createOutput(fmt.Sprintf("No 'type-id' tag in %s", input.ToLineProtocol(nil)))
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
		valueRaw, ok := input.GetField("value")
		if !ok {
			return createOutput(fmt.Sprintf("No 'value' field in %s", input.ToLineProtocol(nil)))
		}
		value := fmt.Sprintf("%v", valueRaw)
		cclog.ComponentDebug("Sysfeatures", "Creating device", deviceType, " ", deviceId)
		dev, err := sysfeatures.LikwidDeviceCreateByTypeName(deviceType, deviceId)
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s/%s", deviceType, deviceId))
		}
		cclog.ComponentDebug("Sysfeatures", "Set", knob, "for device", deviceType, " ", deviceId, "to", value)
		err = sysfeatures.SysFeaturesSetByNameAndDevice(knob, dev, value)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to set %s=%s for device %s/%s", knob, value, deviceType, deviceId))
		}
	} else if method == "GET" {
		cclog.ComponentDebug("Sysfeatures", "Creating device", deviceType, " ", deviceId)
		dev, err := sysfeatures.LikwidDeviceCreateByTypeName(deviceType, deviceId)
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s/%s", deviceType, deviceId))
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", deviceType, " ", deviceId)
		value, err := sysfeatures.SysFeaturesGetByNameAndDevice(knob, dev)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to get %s for device %s/%s", knob, deviceType, deviceId))
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", deviceType, " ", deviceId, "returned", value)
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

func LoadNatsConfiguration(filename string) (NatsConfig, error) {
	natsConfig := NatsConfig{
		OutstandingMessages: 1000,
	}
	configFile, err := os.Open(filename)
	if err != nil {
		cclog.Error(err.Error())
		return natsConfig, err
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&natsConfig)
	return natsConfig, err
}

func real_main() int {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}
	cc_node_control_hostname = hostname

	err = sysfeatures.SysFeaturesInit()
	if err != nil {
		cclog.Errorf("SysFeaturesInit() failed: %v", err)
		return 1
	}
	defer sysfeatures.SysFeaturesClose()

	cli_opts := ReadCli()
	if len(cli_opts["configfile"]) == 0 {
		cli_opts["configfile"] = "./config.json"
	}
	if cli_opts["debug"] == "true" {
		// TODO this no longer exists, cleanup
		//cclog.SetDebug()
	}
	// TODO this doesn't exist anymore either
	//cclog.SetOutput(cli_opts["logfile"])

	config, err := LoadNatsConfiguration(cli_opts["configfile"])
	if err != nil {
		cclog.Error(err.Error())
		return 1
	}
	if len(config.RequestSubject) == 0 {
		cclog.ComponentError("CONFIG", "Failed to request subject for NATS")
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
