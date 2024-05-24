package utils

import "testing"

func TestGetSN(t *testing.T) {
	sn := GetSN()
	if sn == "" {
		t.Error("机器序列号获取失败")
	}
	t.Log(sn)
}
