package cron

import (
	"fmt"
	"testing"
)

func TestGetNextTimes(t *testing.T) {
	res, err := GetNextTimes("0 */1 * * * *")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(res)
}

func TestCheckCron(t *testing.T) {
	got := CheckCron("0 2 6 2 1 ? *")
	want := "0 2 6 2 1 ? "
	if got != want {
		t.Errorf("单元测试异常，TestCheckCron")
	}
	fmt.Println(got)
}
