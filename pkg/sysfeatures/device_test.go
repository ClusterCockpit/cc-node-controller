package sysfeatures

import "testing"

var test_types map[LikwidDeviceType]string = map[LikwidDeviceType]string{
	LikwidDeviceType(HWThread):       "hwthread",
	LikwidDeviceType(Core):           "core",
	LikwidDeviceType(LastLevelCache): "LLC",
	LikwidDeviceType(CpuDie):         "die",
	LikwidDeviceType(Socket):         "socket",
	LikwidDeviceType(Node):           "node",
	LikwidDeviceType(NumaNode):       "numa",
	LikwidDeviceType(Invalid):        "invalid",
}

func TestDeviceCreate(t *testing.T) {

	typelist := []LikwidDeviceType{
		LikwidDeviceType(HWThread),
		LikwidDeviceType(NumaNode),
		LikwidDeviceType(CpuDie),
		LikwidDeviceType(Socket),
		LikwidDeviceType(Node),
	}

	for _, i := range typelist {
		d, err := LikwidDeviceCreate(int(i), 0)
		if err != nil {
			t.Errorf("%v", err.Error())
		}
		LikwidDeviceDestroy(d)
	}
}

func TestDeviceCreateName(t *testing.T) {

	typelist := []LikwidDeviceType{
		LikwidDeviceType(HWThread),
		LikwidDeviceType(NumaNode),
		LikwidDeviceType(CpuDie),
		LikwidDeviceType(Socket),
		LikwidDeviceType(Node),
	}

	for _, i := range typelist {
		d, err := LikwidDeviceCreateByName(test_types[i], 0)
		if err != nil {
			t.Errorf("%v", err.Error())
		}
		LikwidDeviceDestroy(d)
	}
}

func TestDeviceCreateShouldFail(t *testing.T) {

	typelist := []LikwidDeviceType{
		LikwidDeviceType(HWThread),
		LikwidDeviceType(NumaNode),
		LikwidDeviceType(CpuDie),
		LikwidDeviceType(Socket),
		LikwidDeviceType(Node),
	}

	for _, i := range typelist {
		d, err := LikwidDeviceCreate(int(i), -1)
		if err == nil {
			t.Errorf("device successfully created despite ID -1")
			LikwidDeviceDestroy(d)
		}

	}
}

func TestDeviceCreateNameShouldFail(t *testing.T) {

	typelist := []LikwidDeviceType{
		LikwidDeviceType(HWThread),
		LikwidDeviceType(NumaNode),
		LikwidDeviceType(CpuDie),
		LikwidDeviceType(Socket),
		LikwidDeviceType(Node),
	}

	for _, i := range typelist {
		d, err := LikwidDeviceCreateByName(test_types[i], -1)
		if err == nil {
			t.Errorf("device successfully created despite ID -1")
			LikwidDeviceDestroy(d)
		}

	}
}

func TestDeviceCreateNotImplemented(t *testing.T) {

	typelist := []LikwidDeviceType{
		LikwidDeviceType(Core),
		LikwidDeviceType(LastLevelCache),
	}

	for _, i := range typelist {
		d, err := LikwidDeviceCreate(int(i), -1)
		if err == nil {
			t.Errorf("device successfully created despite not implemented type %v", i)
			LikwidDeviceDestroy(d)
		}
	}
}

func TestDeviceCreateNameNotImplemented(t *testing.T) {

	typelist := []LikwidDeviceType{
		LikwidDeviceType(Core),
		LikwidDeviceType(LastLevelCache),
	}

	for _, i := range typelist {
		d, err := LikwidDeviceCreateByName(test_types[i], -1)
		if err == nil {
			t.Errorf("device successfully created despite not implemented type %v", i)
			LikwidDeviceDestroy(d)
		}
	}
}
