package docker

import (
	"io/ioutil"
	"os"
	"strings"
)

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
