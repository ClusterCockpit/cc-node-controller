package sysfeatures

import (
	"fmt"
	"testing"
)

var testTypes []string = []string{
	"hwthread",
	"core",
	"LLC",
	"die",
	"socket",
	"node",
	"numa",
	"invalid",
}

func TestInit(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
		fmt.Printf("aaaaaaaaaaaaaaaaa\n")
	}
	SysFeaturesClose()
}

func TestList(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer SysFeaturesClose()
	list, err := SysFeaturesList()
	if err != nil {
		t.Error(err.Error())
	}
	if len(list) == 0 {
		t.Errorf("empty sysfeatures list")
	}
	for _, l := range list {
		t.Log(l)
	}
}

func TestGet(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer SysFeaturesClose()
	list, err := SysFeaturesList()
	if err != nil {
		t.Error(err.Error())
	}
	for _, l := range list {
		if l.WriteOnly {
			continue
		}
		control := fmt.Sprintf("%s.%s", l.Category, l.Name)
		dev, err := LikwidDeviceCreate(l.DevType, "0")
		if err != nil {
			t.Error(err.Error())
		}
		s, err := SysFeaturesGetByNameAndDevice(control, dev)
		if err != nil {
			t.Errorf("Control %s: %v", control, err.Error())
		}
		t.Logf("Control %s: %s", control, s)

	}
}

func TestSet(t *testing.T) {
	control := "pkg_limit_1"
	has_control := false
	max_control := "pkg_max_limit"
	has_max_control := false
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}
	devtype := LikwidDeviceTypeNameToId("invalid")
	socketType := LikwidDeviceTypeNameToId("socket")
	defer SysFeaturesClose()
	list, err := SysFeaturesList()
	if err != nil {
		t.Error(err.Error())
	}
	for _, l := range list {
		if l.Name == control {
			has_control = true
			devtype = LikwidDeviceType(l.DevType)
		}
		if l.Name == max_control {
			has_max_control = true
		}
		if has_control && has_max_control {
			break
		}
	}
	if has_control && has_max_control && devtype != LikwidDeviceTypeNameToId("invalid") {
		cur, err := SysFeaturesGetByNameAndDevId(control, socketType, "0")
		if err != nil {
			t.Error(err.Error())
		}
		t.Logf("%s -> %s", control, cur)
		max, err := SysFeaturesGetByNameAndDevId(max_control, socketType, "0")
		if err != nil {
			t.Error(err.Error())
		}
		t.Logf("%s -> %s", max_control, max)
		max = "2432000000"
		t.Logf("%s (reset) -> %s", max_control, max)

		err = SysFeaturesSetByNameAndDevId(control, socketType, "0", max)
		if err != nil {
			t.Error(err.Error())
		}
		tmp, err := SysFeaturesGetByNameAndDevId(control, socketType, "0")
		if err != nil {
			t.Error(err.Error())
		}
		t.Logf("%s -> %s", control, tmp)
		if tmp != max {
			t.Errorf("expected %s but got %s", max, tmp)
		}
		t.Logf("%s (reset) -> %s", control, cur)
		err = SysFeaturesSetByNameAndDevId(control, devtype, "0", cur)
		if err != nil {
			t.Error(err.Error())
		}
	}
}

func TestDeviceCreate(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}

	for _, deviceTypeName := range testTypes {
		deviceTypeId := LikwidDeviceTypeNameToId(deviceTypeName)
		d, err := LikwidDeviceCreate(deviceTypeId, "0")
		if err != nil {
			t.Errorf("%v", err.Error())
		}
		LikwidDeviceDestroy(d)
	}

	SysFeaturesClose()
}

func TestDeviceCreateName(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}

	for _, deviceType := range testTypes {
		d, err := LikwidDeviceCreateByTypeName(deviceType, "0")
		if err != nil {
			t.Errorf("%v", err.Error())
		}
		LikwidDeviceDestroy(d)
	}

	SysFeaturesClose()
}

func TestDeviceCreateShouldFail(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}

	for _, deviceTypeName := range testTypes {
		deviceTypeId := LikwidDeviceTypeNameToId(deviceTypeName)
		d, err := LikwidDeviceCreate(deviceTypeId, "-1")
		if err == nil {
			t.Errorf("device successfully created despite ID -1")
			LikwidDeviceDestroy(d)
		}
	}

	SysFeaturesClose()
}

func TestDeviceCreateNameShouldFail(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}

	for _, deviceType := range testTypes {
		d, err := LikwidDeviceCreateByTypeName(deviceType, "-1")
		if err == nil {
			t.Errorf("device successfully created despite ID -1")
			LikwidDeviceDestroy(d)
		}
	}

	SysFeaturesClose()
}

func TestDeviceCreateNotImplemented(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}

	typeList := []string{
		"LLC",
	}

	for _, deviceTypeName := range typeList {
		deviceTypeId := LikwidDeviceTypeNameToId(deviceTypeName)
		d, err := LikwidDeviceCreate(deviceTypeId, "-1")
		if err == nil {
			t.Errorf("device successfully created despite not implemented type %v", deviceTypeName)
			LikwidDeviceDestroy(d)
		}
	}

	SysFeaturesClose()
}

func TestDeviceCreateNameNotImplemented(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Fatal(err.Error())
	}

	typeList := []string{
		"LLC",
	}

	for _, deviceType := range typeList {
		d, err := LikwidDeviceCreateByTypeName(deviceType, "-1")
		if err == nil {
			t.Errorf("device successfully created despite not implemented type %v", deviceType)
			LikwidDeviceDestroy(d)
		}
	}

	SysFeaturesClose()
}
