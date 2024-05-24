package fileutil

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestListDir(t *testing.T) {
	infos, err := ListDir("/Users/tanyi/Documents/gocode/backup")
	if err != nil {
		return
	}
	marshal, err := json.Marshal(infos)
	if err != nil {
		return
	}
	fmt.Println(string(marshal))
}
func TestFixPath(t *testing.T) {
	fmt.Println(FixPath("/Users/tanyi/Documents/gocode/backup/conf"))
}
func TestGetFilePath(t *testing.T) {
	res := GetFilePath("../tanyi/dsdfsdf")
	fmt.Println(res)
}
