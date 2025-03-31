package main

import (
	"encoding/json"
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

func ProcessPutGet(request lp.CCMessage) (lp.CCMessage, error) {
	cclog.ComponentDebug("Sysfeatures", "Processing", request.ToLineProtocol(nil))

	makeReply := func(level, fmtStr string, args ...any) (lp.CCMessage, error) {
		logMsg := fmt.Sprintf(fmtStr, args...)
		resp, err := lp.NewLog(request.Name(), request.Tags(), request.Meta(), logMsg, time.Now())
		if err != nil {
			return nil, fmt.Errorf("Unable to create log message: %w", err)
		}
		resp.AddTag("level", level)
		return resp, nil
	}

	makeErrorReply := func(fmtStr string, args ...any) (lp.CCMessage, error) {
		return makeReply("ERROR", fmtStr, args...)
	}

	if !request.IsControl() {
		return makeErrorReply("Received message is not a control message: %v", request)
	}

	deviceType, ok := request.GetTag("type")
	if !ok {
		return makeErrorReply("No 'type' tag in request: %v", request)
	}

	var deviceId string
	if deviceType != "node" {
		var ok bool
		deviceId, ok = request.GetTag("type-id")
		if !ok {
			return makeErrorReply("No 'type-id' tag in request: %v", request)
		}
	}

	knob := request.Name()
	if method := request.GetControlMethod(); method == "PUT" {
		cclog.ComponentDebug("Sysfeatures", "Creating LIKWID device", deviceType, " ", deviceId)
		dev, err := sysfeatures.LikwidDeviceCreateByTypeName(deviceType, deviceId)
		if err != nil {
			return makeErrorReply(fmt.Sprintf("Cannot create LIKWID device %s/%s", deviceType, deviceId))
		}
		defer sysfeatures.LikwidDeviceDestroy(dev)

		value := request.GetControlValue()
		cclog.ComponentDebug("Sysfeatures", "Set", knob, "for device", deviceType, " ", deviceId, "to", value)
		err = sysfeatures.SysFeaturesSetByNameAndDevice(knob, dev, value)
		if err != nil {
			return makeErrorReply(fmt.Sprintf("Failed to set %s=%s for device %s/%s", knob, value, deviceType, deviceId))
		}

		return makeReply("INFO", "Set '%s' for device '%s:%s': SUCCESS!", knob, deviceType, deviceId)
	} else if method == "GET" {
		cclog.ComponentDebug("Sysfeatures", "Creating LIKWID device", deviceType, " ", deviceId)
		dev, err := sysfeatures.LikwidDeviceCreateByTypeName(deviceType, deviceId)
		if err != nil {
			return makeErrorReply(fmt.Sprintf("Cannot create LIKWID device %s/%s", deviceType, deviceId))
		}
		defer sysfeatures.LikwidDeviceDestroy(dev)

		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", deviceType, " ", deviceId)
		value, err := sysfeatures.SysFeaturesGetByNameAndDevice(knob, dev)
		if err != nil {
			return makeErrorReply(fmt.Sprintf("Failed to get %s for device %s/%s", knob, deviceType, deviceId))
		}

		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", deviceType, " ", deviceId, "returned", value)
		return makeReply("INFO", "%s", value)
	} else {
		return makeReply("Invalid method '%s' in control request: %v", method, request)
	}
}

func ReadCli() map[string]string {
	var m map[string]string
	cfg := flag.String("config", "./config.json", "Path to configuration file")
	loglevel := flag.String("loglevel", "warn", "Activate debug output")
	pretend := flag.Bool("pretend", false, "Do not actually do anything")
	flag.Parse()
	m = make(map[string]string)
	m["configfile"] = *cfg
	m["loglevel"] = *loglevel
	m["pretend"] = fmt.Sprintf("%v", *pretend)
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

	cclog.Init(cli_opts["loglevel"], false)

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
							// In this case, name corresponds to the control, that is to be read/written
							r, err = ProcessPutGet(m)
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
