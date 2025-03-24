package main

import (
	"errors"
	"fmt"
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
	var deviceId string
	knob := input.Name()
	cclog.ComponentDebug("Sysfeatures", "Processing", input)
	deviceType, ok := input.GetTag("type")
	if !ok {
		return createOutput(fmt.Sprintf("No 'type' tag in %s", input), input.Tags())
	}
	if deviceType != "node" {
		var ok bool
		deviceId, ok = input.GetTag("type-id")
		if !ok {
			return createOutput(fmt.Sprintf("No 'type-id' tag in %s", input), input.Tags())
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
		valueRaw, ok := input.GetField("value")
		if !ok {
			return createOutput(fmt.Sprintf("No 'value' field in %s", input), input.Tags())
		}
		value := fmt.Sprintf("%v", valueRaw)
		cclog.ComponentDebug("Sysfeatures", "Creating device type", deviceType, "of id", deviceId)
		dev, err := sysfeatures.LikwidDeviceCreateByTypeName(deviceType, deviceId)
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d: %v", deviceType, deviceId, err.Error()), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Set ", knob, " for device type", deviceType, "of id", deviceId, "to", value)
		err = sysfeatures.SysFeaturesSetByNameAndDevice(knob, dev, value)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to set %s=%s for device %s%d: %v", knob, value, deviceType, deviceId, err.Error()), input.Tags())
		}
	} else if method == "GET" {
		cclog.ComponentDebug("Sysfeatures", "Creating device", deviceType, " ", deviceId)
		dev, err := sysfeatures.LikwidDeviceCreateByTypeName(deviceType, deviceId)
		if err != nil {
			return createOutput(fmt.Sprintf("Cannot create LIKWID device %s%d: %v", deviceType, deviceId, err.Error()), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", deviceType, " ", deviceId)
		value, err := sysfeatures.SysFeaturesGetByNameAndDevice(knob, dev)
		if err != nil {
			return createOutput(fmt.Sprintf("Failed to get %s for device %s%d: %v", knob, deviceType, deviceId, err.Error()), input.Tags())
		}
		cclog.ComponentDebug("Sysfeatures", "Get", knob, "for device", deviceType, " ", deviceId, "returned", value)
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
			DeviceType:  c.DevTypeName,
			Description: c.Description,
			Methods:     getMethods(c.ReadOnly, c.WriteOnly),
		})
	}

	return out, nil
}
