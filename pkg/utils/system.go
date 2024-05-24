package utils

import "github.com/yumaojun03/dmidecode"

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
