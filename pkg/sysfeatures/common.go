package sysfeatures

/*
#cgo CFLAGS: -I./likwid -DLIKWID_WITH_SYSFEATURES
#cgo LDFLAGS: -L/tmp/likwid-install/lib -Wl,-rpath=/tmp/likwid-install/lib -Wl,--unresolved-symbols=ignore-in-object-files -llikwid
#include <stdlib.h>
#ifndef LIKWID_WITH_SYSFEATURES
#define LIKWID_WITH_SYSFEATURES
#endif
#include <likwid.h>
*/
import "C"
import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/dl"
)

const (
	LIKWID_LIB_NAME     = "liblikwid.so"
	LIKWID_LIB_DL_FLAGS = dl.RTLD_LAZY | dl.RTLD_GLOBAL
)

func OpenLikwidLibrary() error {
	lib := dl.New(LIKWID_LIB_NAME, LIKWID_LIB_DL_FLAGS)
	if lib == nil {
		return fmt.Errorf("error instantiating DynamicLibrary %s", LIKWID_LIB_NAME)
	}
	err := lib.Open()
	if err != nil {
		return fmt.Errorf("error opening %s: %v", lib.Name, err)
	}
	//fmt.Printf("Library %s opened\n", LIKWID_LIB_NAME)
	err = lib.Lookup("likwid_getSysFeaturesSupport")
	if err != nil {
		return fmt.Errorf("LIKWID library %s version 5.4+ required: %v", lib.Name, err)
	}
	//fmt.Println("Found symbol likwid_getSysFeaturesSupport")
	// if C.likwid_getSysFeaturesSupport == nil {
	// 	return errors.New("function likwid_getSysFeaturesSupport is NULL")
	// }
	// fmt.Println("Function likwid_getSysFeaturesSupport valid")
	err = lib.Lookup("likwid_sysft_init")
	if err != nil {
		return fmt.Errorf("LIKWID library %s built without SysFeatures support: %v", lib.Name, err)
	}
	return nil
}
