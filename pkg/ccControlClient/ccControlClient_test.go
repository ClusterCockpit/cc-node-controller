package cccontrolclient

import (
	"fmt"
	"testing"
)

var (
	natsConfig NatsConfig = NatsConfig{
		Server:         "127.0.0.1",
		Port:           4222,
		RequestSubject: "cc-control",
	}
)

func TestGetControls(t *testing.T) {
	c, err := NewCCControlClient(natsConfig)
	if err != nil {
		t.Fatal(err.Error())
	}
	control, err := c.GetControls("nuc")
	if err != nil {
		t.Error(err.Error())
	}
	if len(control.Controls) == 0 {
		t.Error("empty response")
	}
	for _, ctrl := range control.Controls {
		t.Log(ctrl)
	}

	c.Close()
}

func TestGetTopology(t *testing.T) {
	target := "nuc"
	c, err := NewCCControlClient(natsConfig)
	if err != nil {
		t.Fatal(err.Error())
	}
	topo, err := c.GetTopology(target)
	if err != nil {
		t.Error(err.Error())
	}
	if len(topo.HWthreads) == 0 {
		t.Error("empty response")
	}
	t.Logf("Target host %s has %d HWThreads", target, len(topo.HWthreads))

	c.Close()
}

func TestGetControlValue(t *testing.T) {
	target := "nuc"
	control := "rapl.pkg_max_limit"
	device := "socket"
	deviceID := "0"

	c, err := NewCCControlClient(natsConfig)
	if err != nil {
		t.Fatal(err.Error())
	}
	value, err := c.GetControlValue(target, control, device, deviceID)
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(value)

	c.Close()
}

func TestSetControlValue(t *testing.T) {
	target := "nuc"
	control := "rapl.pkg_limit_1"
	max_control := "rapl.pkg_max_limit"
	device := "socket"
	deviceID := "0"
	var outerr error = nil

	c, err := NewCCControlClient(natsConfig)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer c.Close()
	cur, err := c.GetControlValue(target, control, device, deviceID)
	if err != nil {
		t.Error(err.Error())
	}
	max, err := c.GetControlValue(target, max_control, device, deviceID)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("current value %s max value %s", cur, max)
	err = c.SetControlValue(target, control, device, deviceID, max)
	if err != nil {
		t.Error(err.Error())
	}
	test, err := c.GetControlValue(target, control, device, deviceID)
	if err != nil {
		t.Error(err.Error())
	}
	if test != max {
		outerr = fmt.Errorf("Setting %s failed. Expected %s but is %s", control, max, test)
	}

	err = c.SetControlValue(target, control, device, deviceID, cur)
	if err != nil {
		t.Error(err.Error())
	}
	if outerr != nil {
		t.Error(err.Error())
	}
}
