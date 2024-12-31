package sysfeatures

/*
#cgo CFLAGS: -I./likwid -DLIKWID_WITH_SYSFEATURES
#cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-in-object-files
#include <stdlib.h>
#ifndef LIKWID_WITH_SYSFEATURES
#define LIKWID_WITH_SYSFEATURES
#endif
#include <likwid.h>

// Helper functions to access bitfields in SysFeature struct
int getReadOnly(LikwidSysFeatureList p, int idx) { return (idx > 0 && idx < p.num_features ? p.features[idx].readonly : 0); }
int getWriteOnly(LikwidSysFeatureList p, int idx) { return (idx > 0 && idx < p.num_features ? p.features[idx].writeonly : 0); }
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"
)

type SysFeature struct {
	Name        string
	Category    string
	Description string
	Devtype     int
	DevtypeName string
	ReadOnly    bool
	WriteOnly   bool
}

func (s *SysFeature) String() string {
	slist := make([]string, 0)
	slist = append(slist, fmt.Sprintf("Category %s Name %s", s.Category, s.Name))
	slist = append(slist, fmt.Sprintf("Description: %s", s.Description))
	slist = append(slist, fmt.Sprintf("For type: %s", s.DevtypeName))
	return strings.Join(slist, "\n")
}

type SysFeatures struct {
	list         []SysFeature
	inititalized bool
}

var _likwid_sysfeatures SysFeatures = SysFeatures{
	list:         make([]SysFeature, 0),
	inititalized: false,
}

func SysFeaturesInit() error {

	err := OpenLikwidLibrary()
	if err != nil {
		return err
	}
	//fmt.Println("Checking sysFeatures support")
	cerr := C.likwid_getSysFeaturesSupport()
	if cerr == 0 {
		return fmt.Errorf("likwid library built without SysFeatures support")
	}
	//fmt.Println("Getting topology")
	cerr = C.topology_init()
	if cerr != 0 {
		return fmt.Errorf("failed to initialize topology component")
	}
	//fmt.Println("Getting affinity")
	C.affinity_init()
	//fmt.Println("Running likwid_sysft_init")
	cerr = C.likwid_sysft_init()
	if cerr != 0 {
		return fmt.Errorf("failed to initialize SysFeatures component")
	}
	_likwid_sysfeatures.inititalized = true
	return nil
}

func SysFeaturesClose() {
	if _likwid_sysfeatures.inititalized {
		C.likwid_sysft_finalize()
		C.affinity_finalize()
		C.topology_finalize()
		_likwid_sysfeatures.inititalized = false
	}
}

var LikwidDeviceTypeNames map[uint32]string = map[uint32]string{
	0:                              "invalid",
	uint32(C.DEVICE_TYPE_HWTHREAD): "hwthread",
	uint32(C.DEVICE_TYPE_CORE):     "core",
	uint32(C.DEVICE_TYPE_LLC):      "LLC",
	uint32(C.DEVICE_TYPE_NUMA):     "numa",
	uint32(C.DEVICE_TYPE_DIE):      "die",
	uint32(C.DEVICE_TYPE_SOCKET):   "socket",
	uint32(C.DEVICE_TYPE_NODE):     "node",
}

func SysFeaturesList() ([]SysFeature, error) {
	if !_likwid_sysfeatures.inititalized {
		return nil, fmt.Errorf("SysFeatures not initialized")
	}
	if len(_likwid_sysfeatures.list) == 0 {
		var l C.LikwidSysFeatureList
		l.num_features = 0
		l.features = nil

		cerr := C.likwid_sysft_list(&l)
		if cerr != 0 {
			return nil, fmt.Errorf("failed to get list of SysFeatures")
		}
		slice := unsafe.Slice(l.features, l.num_features)
		for i, csf := range slice {
			rdonly := int(C.getReadOnly(l, C.int(i)))
			b_rdonly := false
			wronly := int(C.getWriteOnly(l, C.int(i)))
			b_wronly := false
			if rdonly == 1 {
				b_rdonly = true
			}
			if wronly == 1 {
				b_wronly = true
			}
			sf := SysFeature{
				Name:        C.GoString(csf.name),
				Category:    C.GoString(csf.category),
				Devtype:     int(csf._type),
				DevtypeName: LikwidDeviceTypeNames[uint32(csf._type)],
				Description: C.GoString(csf.description),
				ReadOnly:    b_rdonly,
				WriteOnly:   b_wronly,
			}
			_likwid_sysfeatures.list = append(_likwid_sysfeatures.list, sf)
		}
		C.likwid_sysft_list_return(&l)
	}
	outlist := make([]SysFeature, 0)
	outlist = append(outlist, _likwid_sysfeatures.list...)
	return outlist, nil
}

func SysFeaturesGetDevice(name string, dev LikwidDevice) (string, error) {
	var val *C.char
	if !_likwid_sysfeatures.inititalized {
		return "", fmt.Errorf("SysFeatures not initialized")
	}
	cs := C.CString(name)
	cerr := C.likwid_sysft_getByName(cs, dev._raw, &val)
	C.free(unsafe.Pointer(cs))
	if cerr != 0 {
		return "", fmt.Errorf("failed to get feature '%s' for device (%s, %d)", name, dev.Devname, dev.Id)
	}
	return C.GoString(val), nil
}

func SysFeaturesGet(name string, devicetype int, deviceidx int) (string, error) {
	if !_likwid_sysfeatures.inititalized {
		return "", fmt.Errorf("SysFeatures not initialized")
	}
	dev, err := LikwidDeviceCreate(devicetype, deviceidx)
	if err != nil {
		return "", err
	}
	val, err := SysFeaturesGetDevice(name, dev)
	LikwidDeviceDestroy(dev)
	if err != nil {
		return "", err
	}
	return val, nil
}

func SysFeaturesSetDevice(name string, dev LikwidDevice, value string) error {
	//var val *C.char
	if !_likwid_sysfeatures.inititalized {
		return fmt.Errorf("SysFeatures not initialized")
	}
	cs := C.CString(name)
	cv := C.CString(value)
	cerr := C.likwid_sysft_modifyByName(cs, dev._raw, cv)
	C.free(unsafe.Pointer(cv))
	C.free(unsafe.Pointer(cs))
	if cerr != 0 {
		return fmt.Errorf("failed to set feature '%s' for device (%s, %d): %d", name, dev.Devname, dev.Id, int(cerr))
	}
	return nil
}

func SysFeaturesSet(name string, devicetype int, deviceidx int, value string) error {
	if !_likwid_sysfeatures.inititalized {
		return fmt.Errorf("SysFeatures not initialized")
	}
	dev, err := LikwidDeviceCreate(devicetype, deviceidx)
	if err != nil {
		return err
	}
	err = SysFeaturesSetDevice(name, dev, value)
	LikwidDeviceDestroy(dev)
	if err != nil {
		return err
	}
	return nil
}
