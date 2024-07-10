package utils

import (
	"bytes"
	"encoding/json"
)

// ToJSONString 这个函数接受一个接口作为参数并返回一个字符串
func ToJSONString(status interface{}) string {
	if status == nil {
		return ""
	}
	// 创建一个新的缓冲区
	buf := new(bytes.Buffer)
	// 将状态编码到缓冲区
	err := json.NewEncoder(buf).Encode(status)
	// 如果出现错误，panic
	if err != nil {
		panic(err)
	}
	// 返回缓冲区的字符串表示形式
	return buf.String()
}
