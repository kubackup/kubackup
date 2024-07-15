package utils

import (
	"fmt"
	"github.com/super-l/machine-code/machine"
	"runtime"
)

// GetSN 获取机器序列号
func GetSN() string {
	sn, err := machine.GetBoardSerialNumber()
	if err != nil {
		return ""
	}
	return sn
}

// GetCpuCores 获取cpu核心总数量
func GetCpuCores() int {
	fmt.Println(runtime.GOMAXPROCS(0))
	return runtime.NumCPU()
}
