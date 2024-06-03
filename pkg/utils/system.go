package utils

import (
	"github.com/yumaojun03/dmidecode"
	"io/ioutil"
	"os"
	"strings"
)

// GetSN 获取机器序列号
func GetSN() string {
	dmi, err := dmidecode.New()
	if err != nil {
		return ""
	}
	infos, err := dmi.System()
	if err != nil {
		return ""
	}
	if len(infos) == 0 {
		return ""
	}
	return infos[0].SerialNumber
}

// IsDockerEnv 是否运行于docker环境
func IsDockerEnv() bool {
	_, err := os.Stat("/.dockerenv")
	if err == nil {
		return true
	}
	content, err := ioutil.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, "/docker/") || strings.Contains(line, "/kubepod/") {
			return true
		}
	}

	return false
}
