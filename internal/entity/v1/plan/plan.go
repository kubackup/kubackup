package plan

import (
	"github.com/kubackup/kubackup/internal/entity/v1/common"
)

type Plan struct {
	common.BaseModel `storm:"inline"`
	Name             string `json:"name"`
	Path             string `json:"path"` //备份路径或还原路径
	RepositoryId     int    `json:"repositoryId"`
	Status           int    `json:"status"`
	ExecTimeCron     string `json:"execTimeCron"` //定时执行时间
}

// 计划/策略 状态
const (
	RunningStatus = 1
	StopStatus    = 2
)
