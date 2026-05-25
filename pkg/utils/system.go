package utils

import (
	"fmt"
	"os"
	"runtime"
)

// GetSN 获取机器序列号
func GetSN() string {
	return getMachineSerialNumber()
}

func getMachineSerialNumber() string {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/etc/machine-id")
		if err == nil && len(data) > 0 {
			return string(data[:len(data)-1]) // remove newline
		}
	case "darwin":
		data, err := os.ReadFile("/var/db/SystemID")
		if err == nil && len(data) > 0 {
			return string(data)
		}
	case "windows":
		// Windows implementation would require WMI or registry access
		// For simplicity, return empty string
	}
	return ""
}

// GetCpuCores 获取cpu核心总数量
func GetCpuCores() int {
	fmt.Println(runtime.GOMAXPROCS(0))
	return runtime.NumCPU()
}
