package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ClusterCockpit/cc-node-controller/pkg/sysfeatures"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
)

func ProcessSysfeatures(input lp.CCMessage) (lp.CCMessage, error) {

	createOutput := func(errorString string, tags map[string]string) (lp.CCMessage, error) {
		resp, err := lp.NewLog("knobs", tags, map[string]string{}, errorString, time.Now())
		if err == nil {
			resp.AddTag("level", "ERROR")
			return resp, errors.New(errorString)
		}
		return nil, fmt.Errorf("%s and cannot send response", errorString)
	}
	var tid int64 = 0
	var err error = nil
	knob := input.Name()
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
			return createOutput(fmt.Sprintf("Cannot parse 'type-id' tag in %s: %v", input, err.Error()), input.Tags())
		}
	}
	cclog.ComponentDebug("Sysfeatures", "Getting method", input)
	method, ok := input.GetTag("method")
	if !ok {
		return createOutput(fmt.Sprintf("No 'method' tag in %s", input), input.Tags())
	}
	if method != "PUT" && method != "GET" {
		return createOutput(fmt.Sprintf("Invalid 'method' tag %s in %s", method, input), input.Tags())
	}
	if method == "PUT" {
		value, ok := input.GetField("value")
		if !ok {
			return createOutput(fmt.Sprintf("No 'value' field in %s", input), input.Tags())
		}
		svalue := ""
		switch v := value.(type) {
		case string:
			cclog.ComponentDebug("Sysfeatures", "Value is a string")
			svalue = v
		default:
			cclog.ComponentDebug("Sysfeatures", "Value is a other, use sprintf")
			svalue = fmt.Sprintf("%v", v)
			cclog.ComponentDebug("Sysfeatures", "Value is a other", svalue)
		}

		cclog.ComponentDebug("Sysfeatures", "Creating device", t, " ", int(tid))
		dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d: %v", t, tid, err.Error()), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Set", knob, "for device", t, " ", int(tid), "to", svalue)
		err = sysfeatures.SysFeaturesSetDevice(knob, dev, svalue)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to set %s=%s for device %s%d: %v", knob, svalue, t, tid, err.Error()), input.Tags())
		}
	} else if method == "GET" {
		cclog.ComponentDebug("Sysfeatures", "Creating device", t, " ", int(tid))
		dev, err := sysfeatures.LikwidDeviceCreateByName(t, int(tid))
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d: %v", t, tid, err.Error()), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", t, " ", int(tid))
		value, err := sysfeatures.SysFeaturesGetDevice(knob, dev)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to get %s for device %s%d: %v", knob, t, tid, err.Error()), input.Tags())
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

func ProcessSysfeaturesConfig() ([]CCControlListEntry, error) {
	out := make([]CCControlListEntry, 0)
	sysfList, err := sysfeatures.SysFeaturesList()
	if err != nil {
		return out, err
	}

	getMethods := func(readonly bool, writeonly bool) string {
		if readonly && writeonly {
			return "ERROR"
		} else if readonly && (!writeonly) {
			return "GET"
		} else if (!readonly) && writeonly {
			return "PUT"
		} else {
			return "ALL"
		}
	}

	for _, c := range sysfList {
		out = append(out, CCControlListEntry{
			Category:    c.Category,
			Name:        c.Name,
			DeviceType:  c.DevtypeName,
			Description: c.Description,
			Methods:     getMethods(c.ReadOnly, c.WriteOnly),
		})
	}

	return out, nil
}
