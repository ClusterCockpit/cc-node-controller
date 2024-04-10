package main

import (
	"cc-node-controller/pkg/sysfeatures"
	"errors"
	"fmt"
	"strconv"
	"time"

	cclog "github.com/ClusterCockpit/cc-metric-collector/pkg/ccLogger"
	ccmetric "github.com/ClusterCockpit/cc-metric-collector/pkg/ccMetric"
)

func ProcessSysfeatures(input ccmetric.CCMetric) (ccmetric.CCMetric, error) {

	createOutput := func(errorString string, tags map[string]string) (ccmetric.CCMetric, error) {
		resp, err := ccmetric.New("knobs", tags, map[string]string{}, map[string]interface{}{"value": errorString}, time.Now())
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
			return createOutput(fmt.Sprintf("Cannot parse 'type-id' tag in %s", input), input.Tags())
		}
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
