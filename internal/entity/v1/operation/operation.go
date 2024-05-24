package operation

import (
	"github.com/kubackup/kubackup/internal/entity/v1/common"
	"github.com/kubackup/kubackup/internal/store/ws_task_info"
)

type Operation struct {
	common.BaseModel `storm:"inline"`
	RepositoryId     int                  `json:"repositoryId"`
	PolicyId         int                  `json:"policyId"`
	Type             int                  `json:"type"`
	Status           int                  `json:"status"`
	Logs             []*wsTaskInfo.Sprint `json:"logs"`
}

const (
	CHECK_TYPE        = 1 // CHECK 检测仓库状态
	REBUILDINDEX_TYPE = 2 // REBUILDINDEX 重建索引
	PRUNE_TYPE        = 3 // PRUNE 清理无用数据
	FORGET_TYPE       = 4 // FORGET 清理过期快照
)
