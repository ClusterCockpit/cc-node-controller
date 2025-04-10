package main

import (
	"encoding/json"
	"fmt"
	"time"

	topo "github.com/ClusterCockpit/cc-node-controller/pkg/ccTopology"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
)

type CCControlTopology struct {
	HWthreads []topo.HwthreadEntry `json:"hwthreads"`
	CpuInfo   topo.CpuInformation  `json:"cpu_info"`
}

func ProcessTopologyConfig(input lp.CCMessage) (lp.CCMessage, error) {
	createOutput := func(str string, tags map[string]string) (lp.CCMessage, error) {
		resp, err := lp.NewLog("topology", tags, map[string]string{}, str, time.Now())
		if err == nil {
			resp.AddTag("level", "ERROR")
			return resp, nil
		}
		return nil, fmt.Errorf("%s and cannot send response", str)
	}
	tc := CCControlTopology{
		CpuInfo:   topo.CpuInfo(),
		HWthreads: topo.CpuData(),
	}

	out, err := json.Marshal(tc)
	if err != nil {
		cclog.ComponentError("Config", err.Error())
		return createOutput(err.Error(), input.Tags())
	}
	resp, err := createOutput(string(out), input.Tags())
	if err == nil {
		resp.AddTag("level", "INFO")
	} else {
		cclog.ComponentError("ProcessTopologyConfig", err.Error())
	}
	return resp, err
}

type CCControlListEntry struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	DeviceType  string `json:"device_type"`
	Description string `json:"description"`
	Methods     string `json:"methods"`
}

type CCControlList struct {
	Controls []CCControlListEntry `json:"controls"`
}

func ProcessControlsConfig(input lp.CCMessage) (lp.CCMessage, error) {
	createOutput := func(str string, tags map[string]string) (lp.CCMessage, error) {
		resp, err := lp.NewLog("controls", tags, map[string]string{}, str, time.Now())
		if err == nil {
			resp.AddTag("level", "ERROR")
			return resp, nil
		}
		return nil, fmt.Errorf("%s and cannot send response", str)
	}

	controls := make([]CCControlListEntry, 0)

	sysfeatures_controls, err := ProcessSysfeaturesConfig()
	if err == nil {
		controls = append(controls, sysfeatures_controls...)
	}

	// if we want other sources, add them here

	cl := CCControlList{
		Controls: controls,
	}

	clj, err := json.Marshal(cl)
	if err != nil {
		createOutput("cannot marshal input control list", input.Tags())
	}
	resp, err := createOutput(string(clj), input.Tags())
	if err == nil {
		resp.AddTag("level", "INFO")
		return resp, nil
	} else {
		cclog.ComponentError("ProcessControlsConfig", err.Error())
	}

	return createOutput("not implemented", input.Tags())
}
