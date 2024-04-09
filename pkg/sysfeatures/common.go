package sysfeatures

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/dl"
)

const (
	LIKWID_LIB_NAME     = "liblikwid.so"
	LIKWID_LIB_DL_FLAGS = dl.RTLD_LAZY | dl.RTLD_GLOBAL
)

func OpenLikwidLibrary() error {
	lib := dl.New(LIKWID_LIB_NAME, dl.RTLD_LAZY|dl.RTLD_GLOBAL)
	if lib == nil {
		return fmt.Errorf("error instantiating DynamicLibrary %s", LIKWID_LIB_NAME)
	}
	err := lib.Open()
	if err != nil {
		return fmt.Errorf("error opening %s: %v", lib.Name, err)
	}
	err = lib.Lookup("likwid_getSysFeaturesSupport")
	if err != nil {
		return fmt.Errorf("LIKWID library %s version 5.3+ required: %v", lib.Name, err)
	}
	err = lib.Lookup("sysFeatures_init")
	if err != nil {
		return fmt.Errorf("LIKWID library %s built without SysFeatures support: %v", lib.Name, err)
	}
	return nil
}
