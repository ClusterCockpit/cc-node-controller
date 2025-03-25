package sysfeatures

/*
#cgo LDFLAGS: -ldl
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <dlfcn.h>

typedef int LikwidDeviceType;

typedef struct {
    LikwidDeviceType type;
    union {
        struct {
            int id;
        } simple;
        struct {
            uint16_t pci_domain;
            uint8_t pci_bus;
            uint8_t pci_dev;
            uint8_t pci_func;
        } pci;
    } id;
    int internal_id;
} _LikwidDevice;

typedef _LikwidDevice* LikwidDevice_t;

typedef struct {
    char* name;
    char* category;
    char* description;
    LikwidDeviceType type;
    unsigned int readonly:1;
    unsigned int writeonly:1;
} LikwidSysFeature;

typedef struct {
    int num_features;
    LikwidSysFeature* features;
} LikwidSysFeatureList;

static void *cgo_lw_lib;

static int (*likwid_sysft_init_ptr)(void);
static int likwid_sysft_init(void) { return likwid_sysft_init_ptr(); }

static int (*topology_init_ptr)(void);
static int topology_init(void) { return topology_init_ptr(); }

static int (*affinity_init_ptr)(void);
static int affinity_init(void) { return topology_init_ptr(); }

static void (*likwid_sysft_finalize_ptr)(void);
static void likwid_sysft_finalize(void) { likwid_sysft_finalize_ptr(); }

static void (*topology_finalize_ptr)(void);
static void topology_finalize(void) { topology_finalize_ptr(); }

static void (*affinity_finalize_ptr)(void);
static void affinity_finalize(void) { affinity_finalize_ptr(); }

static int (*likwid_sysft_list_ptr)(LikwidSysFeatureList *);
static int likwid_sysft_list(LikwidSysFeatureList *list) { return likwid_sysft_list_ptr(list); }

static void (*likwid_sysft_list_return_ptr)(LikwidSysFeatureList *);
static void likwid_sysft_list_return(LikwidSysFeatureList *list) { likwid_sysft_list_return_ptr(list); }

static int (*likwid_device_create_ptr)(LikwidDeviceType type, int id, LikwidDevice_t *dev);
static int likwid_device_create(LikwidDeviceType type, int id, LikwidDevice_t *dev) {
	return likwid_device_create_ptr(type, id, dev);
}

static int (*likwid_device_create_from_string_ptr)(LikwidDeviceType type, const char *id, LikwidDevice_t *dev);
static int likwid_device_create_from_string(LikwidDeviceType type, const char *id, LikwidDevice_t *dev) {
	return likwid_device_create_from_string_ptr(type, id, dev);
}

static void (*likwid_device_destroy_ptr)(LikwidDevice_t dev);
static void likwid_device_destroy(LikwidDevice_t dev) { likwid_device_destroy_ptr(dev); }

static const char *(*likwid_device_type_name_ptr)(LikwidDeviceType);
static const char *likwid_device_type_name(LikwidDeviceType type) {
	return likwid_device_type_name_ptr(type);
}

static int (*likwid_sysft_getByName_ptr)(const char *, const LikwidDevice_t, char **);
static int likwid_sysft_getByName(const char *name, const LikwidDevice_t dev, char **value) {
	return likwid_sysft_getByName_ptr(name, dev, value);
}

static int (*likwid_sysft_modifyByName_ptr)(const char *, const LikwidDevice_t, const char *);
static int likwid_sysft_modifyByName(const char *name, const LikwidDevice_t dev, const char *value) {
	return likwid_sysft_modifyByName_ptr(name, dev, value);
}

#define INIT_LIKWID_FUNC(func_name) 							\
	do {														\
		func_name##_ptr = dlsym(cgo_lw_lib, #func_name);		\
		if (!func_name##_ptr) {									\
			fprintf(stderr, "[Error] dlsym: %s\n", dlerror());	\
			dlclose(cgo_lw_lib);								\
			cgo_lw_lib = NULL;									\
			return false;										\
		}														\
	} while (0)

static bool cgo_lw_init(void) {
	if (cgo_lw_lib)
		return true;

	cgo_lw_lib = dlopen("liblikwid.so", RTLD_LAZY);
	if (!cgo_lw_lib) {
		fprintf(stderr, "[ERROR] dlopen: %s\n", dlerror());
		return false;
	}

	int (*likwid_getMajorVersion_ptr)(void);
	INIT_LIKWID_FUNC(likwid_getMajorVersion);

	int (*likwid_getMinorVersion_ptr)(void);
	INIT_LIKWID_FUNC(likwid_getMinorVersion);

	const int major = likwid_getMajorVersion_ptr();
	const int minor = likwid_getMinorVersion_ptr();

	// If you run into the error below: After checking that the API is still correct,
	// you can bump the version number to get rid of this warning.
	const int requiredMajor = 5;
	const int requiredMinor = 4;
	if (major != requiredMajor || minor != requiredMinor) {
		fprintf(stderr, "[WARN] Found LIKWID %d.%d.X, but only %d.%d.X is supported. "
			"Malfunction may occur.\n",
			major, minor,
			requiredMajor, requiredMinor);
	}

	int (*likwid_getSysFeaturesSupport_ptr)(void);
	INIT_LIKWID_FUNC(likwid_getSysFeaturesSupport);

	if (!likwid_getSysFeaturesSupport_ptr()) {
		fprintf(stderr, "[ERROR] Found LIKWID, but sysfeatures support is disabled.");
		return false;
	}

	// why are we initalizing it here, if we initialize it anyway later?
	INIT_LIKWID_FUNC(likwid_sysft_init);
	INIT_LIKWID_FUNC(topology_init);
	INIT_LIKWID_FUNC(affinity_init);
	INIT_LIKWID_FUNC(likwid_sysft_finalize);
	INIT_LIKWID_FUNC(topology_finalize);
	INIT_LIKWID_FUNC(affinity_finalize);
	INIT_LIKWID_FUNC(likwid_sysft_list);
	INIT_LIKWID_FUNC(likwid_sysft_list_return);
	INIT_LIKWID_FUNC(likwid_device_create);
	INIT_LIKWID_FUNC(likwid_device_create_from_string);
	INIT_LIKWID_FUNC(likwid_device_destroy);
	INIT_LIKWID_FUNC(likwid_device_type_name);
	INIT_LIKWID_FUNC(likwid_sysft_getByName);

	return true;
}

// TODO remove those two helper functions in the new LIKWID release > 5.4.X, when there are no more bitfields
static int getReadOnly(LikwidSysFeatureList p, int idx) { return (idx > 0 && idx < p.num_features ? p.features[idx].readonly : 0); }
static int getWriteOnly(LikwidSysFeatureList p, int idx) { return (idx > 0 && idx < p.num_features ? p.features[idx].writeonly : 0); }
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"
	"sync"
)

type LikwidDeviceType int

type SysFeature struct {
	Name        string
	Category    string
	Description string
	DevType     LikwidDeviceType
	DevTypeName string
	ReadOnly    bool
	WriteOnly   bool
}

type LikwidDevice struct {
	Id      int64
	DevType LikwidDeviceType
	DevTypeName string
	raw    C.LikwidDevice_t
}

var (
	deviceNameToIdMutex sync.Mutex
	deviceNameToId map[string]LikwidDeviceType
)

func (s *SysFeature) String() string {
	slist := make([]string, 0)
	slist = append(slist, fmt.Sprintf("Category %s Name %s", s.Category, s.Name))
	slist = append(slist, fmt.Sprintf("Description: %s", s.Description))
	slist = append(slist, fmt.Sprintf("For type: %s", s.DevTypeName))
	return strings.Join(slist, "\n")
}

func SysFeaturesInit() error {
	ok := C.cgo_lw_init()
	if !ok {
		return fmt.Errorf("Error while initializing LIKWID sysfeatures")
	}
	if err := C.topology_init(); err != 0 {
		return fmt.Errorf("topology_init: %s", C.GoString(C.strerror(-err)))
	}
	if err := C.affinity_init(); err != 0 {
		return fmt.Errorf("affinity_init: %s", C.GoString(C.strerror(-err)))
	}
	if err := C.likwid_sysft_init(); err != 0 {
		return fmt.Errorf("likwid_sysft_init: %s", C.GoString(C.strerror(-err)))
	}
	return nil
}

func SysFeaturesClose() {
	C.likwid_sysft_finalize()
	C.affinity_finalize()
	C.topology_finalize()
}

func SysFeaturesList() ([]SysFeature, error) {
	sysftList := C.LikwidSysFeatureList{
		num_features: 0,
		features: nil,
	}

	err := C.likwid_sysft_list(&sysftList)
	if err != 0 {
		return nil, fmt.Errorf("likwid_sysft_list")
	}

	defer C.likwid_sysft_list_return(&sysftList)

	retval := make([]SysFeature, 0)

	features := unsafe.Slice(sysftList.features, sysftList.num_features)
	for ftIndex, feature := range features {
		readOnly := false
		if int(C.getReadOnly(sysftList, C.int(ftIndex))) == 1 {
			readOnly = true
		}

		writeOnly := false
		if int(C.getWriteOnly(sysftList, C.int(ftIndex))) == 1 {
			writeOnly = true
		}

		sf := SysFeature{
			Name:        C.GoString(feature.name),
			Category:    C.GoString(feature.category),
			DevType:     LikwidDeviceType(feature._type),
			DevTypeName: C.GoString(C.likwid_device_type_name(feature._type)),
			Description: C.GoString(feature.description),
			ReadOnly:    readOnly,
			WriteOnly:   writeOnly,
		}

		retval = append(retval, sf)
	}

	return retval, nil
}

func SysFeaturesGetByNameAndDevice(name string, dev LikwidDevice) (string, error) {
	var val *C.char
	cName := C.CString(name)
	cerr := C.likwid_sysft_getByName(cName, dev.raw, &val)
	C.free(unsafe.Pointer(cName))
	if cerr != 0 {
		return "", fmt.Errorf("likwid_sysft_getByName() failed (feature=%s, devType=%s, devId=%d): %s", name, dev.DevTypeName, dev.Id, C.GoString(C.strerror(-cerr)))
	}
	defer C.free(unsafe.Pointer(val))
	return C.GoString(val), nil
}

func SysFeaturesGetByNameAndDevId(name string, deviceType LikwidDeviceType, deviceId string) (string, error) {
	dev, err := LikwidDeviceCreate(deviceType, deviceId)
	if err != nil {
		return "", err
	}
	val, err := SysFeaturesGetByNameAndDevice(name, dev)
	LikwidDeviceDestroy(dev)
	if err != nil {
		return "", err
	}
	return val, nil
}

func SysFeaturesSetByNameAndDevice(name string, dev LikwidDevice, value string) error {
	cName := C.CString(name)
	cValue := C.CString(value)
	cerr := C.likwid_sysft_modifyByName(cName, dev.raw, cValue)
	C.free(unsafe.Pointer(cValue))
	C.free(unsafe.Pointer(cName))
	if cerr != 0 {
		return fmt.Errorf("likwid_sysft_modifyByName() failed (feature=%s, devType=%s, devId=%d, value=%s): %s", name, dev.DevTypeName, dev.Id, value, C.GoString(C.strerror(-cerr)))
	}
	return nil
}

func SysFeaturesSetByNameAndDevId(name string, deviceType LikwidDeviceType, deviceId string, value string) error {
	dev, err := LikwidDeviceCreate(deviceType, deviceId)
	if err != nil {
		return err
	}
	err = SysFeaturesSetByNameAndDevice(name, dev, value)
	LikwidDeviceDestroy(dev)
	if err != nil {
		return err
	}
	return nil
}

func LikwidDeviceTypeNameToId(deviceTypeName string) LikwidDeviceType {
	deviceNameToIdMutex.Lock()
	defer deviceNameToIdMutex.Unlock()

	if retval, ok := deviceNameToId[deviceTypeName]; ok {
		return retval
	}

	for i := 1; true; i++ {
		deviceTypeId := LikwidDeviceType(i)
		// Not sure why this function returns "unsupported".
		// "invalid" would be more appropriate to fit with the rest of the LIKWID code.
		ptr := C.likwid_device_type_name(C.LikwidDeviceType(deviceTypeId))
		s := ""
		if ptr != nil {
			s = C.GoString(ptr)
			if s == deviceTypeName {
				deviceNameToId[deviceTypeName] = deviceTypeId
				return deviceTypeId
			}
		}

		if ptr == nil || s == "unsupported" || s == "invalid" {
			break
		}
	}

	deviceNameToId[deviceTypeName] = LikwidDeviceType(0)
	return LikwidDeviceType(0)
}

func LikwidDeviceCreateByTypeName(deviceTypeName string, deviceId string) (LikwidDevice, error) {
	deviceTypeId := LikwidDeviceTypeNameToId(deviceTypeName)
	if deviceTypeId == LikwidDeviceType(0) {
		return LikwidDevice{}, fmt.Errorf("LikwidDeviceCreateByTypeName: Invalid device type %s", deviceTypeName)
	}
	return LikwidDeviceCreate(deviceTypeId, deviceId)
}

func LikwidDeviceDestroy(dev LikwidDevice) {
	C.likwid_device_destroy(dev.raw)
}

func LikwidDeviceCreate(deviceType LikwidDeviceType, deviceId string) (LikwidDevice, error) {
	var cLikwidDevice C.LikwidDevice_t

	err := SysFeaturesInit()
	if err != nil {
		return LikwidDevice{}, err
	}

	cDeviceId := C.CString(deviceId)
	cerr := C.likwid_device_create_from_string(C.LikwidDeviceType(deviceType), cDeviceId, &cLikwidDevice)
	C.free(unsafe.Pointer(cDeviceId))
	if cerr != 0 {
		return LikwidDevice{}, fmt.Errorf("likwid_device_create() failed: (type=%d, idx=%s): %s", deviceType, deviceId, C.GoString(C.strerror(-cerr)))
	}

	id := int64(0)
	for i, d := range cLikwidDevice.id {
		id |= int64(d) << (i * 8)
	}
	dev := LikwidDevice{
		DevType: deviceType,
		DevTypeName: C.GoString(C.likwid_device_type_name(cLikwidDevice._type)),
		Id:      id,
		raw:     cLikwidDevice,
	}
	return dev, nil
}
