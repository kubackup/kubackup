package utils

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/super-l/machine-code/machine"
)

// GetSN 获取机器序列号
func GetSN() string {
	sn, err := machine.GetBoardSerialNumber()
	if err != nil {
		return ""
	}
	return sn
}

// GetCpuThreads 获取cpu线程总数量
func GetCpuThreads() int {
	c, err := cpu.Counts(true)
	if err != nil {
		return 0
	}
	return c
}

// GetCpuCores 获取cpu核心总数量
func GetCpuCores() int {
	c, err := cpu.Counts(false)
	if err != nil {
		return 0
	}
	return c
}
