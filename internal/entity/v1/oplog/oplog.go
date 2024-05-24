package oplog

import "github.com/kubackup/kubackup/internal/entity/v1/common"

type OperationLog struct {
	common.BaseModel `storm:"inline"`
	Operator         string `json:"operator"`  // 操作人
	Operation        string `json:"operation"` // 操作
	Url              string `json:"url"`       // 请求url
	Data             string `json:"data"`      // 请求数据
}
