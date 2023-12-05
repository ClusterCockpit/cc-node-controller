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
	lib := dl.New("liblikwid.so", dl.RTLD_LAZY|dl.RTLD_GLOBAL)
	if lib == nil {
		return fmt.Errorf("error instantiating DynamicLibrary %s", LIKWID_LIB_NAME)
	}
	err := lib.Open()
	if err != nil {
		return fmt.Errorf("error opening %s: %v", LIKWID_LIB_NAME, err)
	}
	return nil
}
