package ccTopology

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestJsonCpuInfo(t *testing.T) {
	icopy := CpuInformation{}
	info := CpuInfo()

	infoTest := func(info CpuInformation) {
		if info.NumHWthreads == 0 {
			t.Errorf("system has zero HW threads")
		}
		if info.SMTWidth == 0 {
			t.Errorf("system has an SMT width of zero")
		}
		if info.NumSockets == 0 {
			t.Errorf("system has zero CPU sockets")
		}
		if info.NumDies == 0 {
			t.Errorf("system has zero CPU dies")
		}
		if info.NumCores == 0 {
			t.Errorf("system has zero cores")
		}
		if info.NumNumaDomains == 0 {
			t.Errorf("system has zero NUMA domains")
		}
	}
	infoTest(info)

	j, err := json.Marshal(info)
	if err != nil {
		t.Errorf("cannot marshal CpuInfo into JSON: %v", err.Error())
	}

	err = json.Unmarshal(j, &icopy)
	if err != nil {
		t.Errorf("cannot unmarshal JSON to CpuInfo: %v", err.Error())

	}
	infoTest(icopy)

	if info.NumHWthreads != icopy.NumHWthreads {
		t.Errorf("Input and parsed CpuInfo do not match")
	}
	if info.SMTWidth != icopy.SMTWidth {
		t.Errorf("Input and parsed CpuInfo do not match")
	}
	if info.NumSockets != icopy.NumSockets {
		t.Errorf("Input and parsed CpuInfo do not match")
	}
	if info.NumDies != icopy.NumDies {
		t.Errorf("Input and parsed CpuInfo do not match")
	}
	if info.NumNumaDomains != icopy.NumNumaDomains {
		t.Errorf("Input and parsed CpuInfo do not match")
	}
	if info.NumCores != icopy.NumCores {
		t.Errorf("Input and parsed CpuInfo do not match")
	}
}

func TestJsonCpuData(t *testing.T) {
	dcopy := []HwthreadEntry{}
	data := CpuData()
	if len(data) == 0 {
		t.Errorf("system has zero HW threads defined")
	}
	fmt.Println("cpuid/smt/core/socket/die/numa")
	for _, t := range data {
		fmt.Printf("%d/%d/%d/%d/%d/%d\n", t.CpuID, t.SMT, t.Core, t.Socket, t.Die, t.NumaDomain)
	}

	j, err := json.Marshal(data)
	if err != nil {
		t.Errorf("cannot marshal CpuData into JSON: %v", err.Error())
	}

	err = json.Unmarshal(j, &dcopy)
	if err != nil {
		t.Errorf("cannot unmarshal JSON to CpuData: %v", err.Error())
	}
	fmt.Println("After JSON")
	fmt.Println("cpuid/smt/core/socket/die/numa")
	for _, t := range dcopy {
		fmt.Printf("%d/%d/%d/%d/%d/%d\n", t.CpuID, t.SMT, t.Core, t.Socket, t.Die, t.NumaDomain)
	}
}
