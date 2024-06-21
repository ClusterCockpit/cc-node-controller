package sysfeatures

import (
	"fmt"
	"testing"
)

func TestInit(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Error(err.Error())
	}
	SysFeaturesClose()
}

func TestList(t *testing.T) {
	err := SysFeaturesInit()
	if err != nil {
		t.Error(err.Error())
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
		t.Error(err.Error())
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
		dev, err := LikwidDeviceCreate(l.Devtype, 0)
		if err != nil {
			t.Error(err.Error())
		}
		s, err := SysFeaturesGetDevice(control, dev)
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
	devtype := LikwidDeviceType(Invalid)
	err := SysFeaturesInit()
	if err != nil {
		t.Error(err.Error())
	}
	defer SysFeaturesClose()
	list, err := SysFeaturesList()
	if err != nil {
		t.Error(err.Error())
	}
	for _, l := range list {
		if l.Name == control {
			has_control = true
			devtype = LikwidDeviceType(l.Devtype)
		}
		if l.Name == max_control {
			has_max_control = true
		}
		if has_control && has_max_control {
			break
		}
	}
	if has_control && has_max_control && devtype != Invalid {
		cur, err := SysFeaturesGet(control, Socket, 0)
		if err != nil {
			t.Error(err.Error())
		}
		t.Logf("%s -> %s", control, cur)
		max, err := SysFeaturesGet(max_control, Socket, 0)
		if err != nil {
			t.Error(err.Error())
		}
		t.Logf("%s -> %s", max_control, max)
		max = "2432000000"
		t.Logf("%s (reset) -> %s", max_control, max)

		err = SysFeaturesSet(control, Socket, 0, max)
		if err != nil {
			t.Error(err.Error())
		}
		tmp, err := SysFeaturesGet(control, Socket, 0)
		if err != nil {
			t.Error(err.Error())
		}
		t.Logf("%s -> %s", control, tmp)
		if tmp != max {
			t.Errorf("expected %s but got %s", max, tmp)
		}
		t.Logf("%s (reset) -> %s", control, cur)
		err = SysFeaturesSet(control, int(devtype), 0, cur)
		if err != nil {
			t.Error(err.Error())
		}
	}
}
