package sysfeatures

/*
#cgo CFLAGS: -I./likwid -DLIKWID_WITH_SYSFEATURES
#cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-in-object-files
#include <stdlib.h>
#ifndef LIKWID_WITH_SYSFEATURES
#define LIKWID_WITH_SYSFEATURES
#endif
#include <likwid.h>
*/
import "C"
import (
	"fmt"
)

type LikwidDeviceType int

const (
	Invalid        LikwidDeviceType = iota
	HWThread                        = int(C.DEVICE_TYPE_HWTHREAD)
	Core                            = int(C.DEVICE_TYPE_CORE)
	LastLevelCache                  = int(C.DEVICE_TYPE_LLC)
	NumaNode                        = int(C.DEVICE_TYPE_NUMA)
	CpuDie                          = int(C.DEVICE_TYPE_DIE)
	Socket                          = int(C.DEVICE_TYPE_SOCKET)
	Node                            = int(C.DEVICE_TYPE_NODE)
)

type LikwidDevice struct {
	Id      int64
	Devtype int
	Devname string
	_raw    C.LikwidDevice_t
}

func LikwidDeviceCreate(devtype int, devidx int) (LikwidDevice, error) {
	var ld C.LikwidDevice_t
	var dev LikwidDevice

	err := OpenLikwidLibrary()
	if err != nil {
		return LikwidDevice{
			Devtype: int(Invalid),
			Devname: "invalid",
			Id:      -1,
			_raw:    nil,
		}, err
	}

	cerr := C.likwid_getSysFeaturesSupport()
	if cerr == 0 {
		return LikwidDevice{
			Devtype: int(Invalid),
			Devname: "invalid",
			Id:      -1,
			_raw:    nil,
		}, fmt.Errorf("likwid library built without SysFeatures support")
	}

	cerr = C.likwid_device_create(C.LikwidDeviceType(devtype), C.int(devidx), &ld)
	if cerr != 0 {
		return LikwidDevice{
			Devtype: int(Invalid),
			Devname: "invalid",
			Id:      -1,
			_raw:    nil,
		}, fmt.Errorf("failed to create device (type %d, idx %d)", devtype, devidx)
	}

	id := int64(0)
	for i, d := range ld.id {
		id |= int64(d) << (i * 8)
	}
	dev = LikwidDevice{
		Devtype: devtype,
		Devname: C.GoString(C.device_type_name(ld._type)),
		Id:      id,
		_raw:    ld,
	}
	return dev, nil
}

func LikwidDeviceCreateByName(devtype string, devidx int) (LikwidDevice, error) {
	for i := int(C.MIN_DEVICE_TYPE); i < int(C.MAX_DEVICE_TYPE); i++ {
		s := C.GoString(C.device_type_name(C.LikwidDeviceType(i)))
		if s == devtype {
			return LikwidDeviceCreate(i, devidx)
		}
	}
	return LikwidDevice{
		Devtype: int(Invalid),
		Devname: "invalid",
		Id:      -1,
		_raw:    nil,
	}, fmt.Errorf("invalid device type %s", devtype)
}

func LikwidDeviceDestroy(dev LikwidDevice) {
	C.likwid_device_destroy(dev._raw)
}
