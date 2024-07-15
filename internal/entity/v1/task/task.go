package task

import (
	"github.com/kubackup/kubackup/internal/entity/v1/common"
	"github.com/kubackup/kubackup/internal/model"
)

type Task struct {
	common.BaseModel `storm:"inline"`
	Name             string               `json:"name"`
	Path             string               `json:"path"`   //备份路径或还原路径
	PlanId           int                  `json:"planId"` //计划id
	RepositoryId     int                  `json:"repositoryId"`
	Status           int                  `json:"status"`
	ParentId         string               `json:"parentId"`      //父快照id
	Scanner          *model.VerboseUpdate `json:"scanner"`       //扫描结果
	ScannerError     *model.ErrorUpdate   `json:"scannerError"`  //扫描错误
	ArchivalError    []model.ErrorUpdate  `json:"archivalError"` //备份错误
	Summary          *model.SummaryOutput `json:"summary"`       //备份结果
	Progress         *model.StatusUpdate  `json:"progress"`      //当前进度
	RestoreError     []model.ErrorUpdate  `json:"restoreError"`  //恢复错误
	ReadConcurrency  uint                 //读取并发数量，默认2
}
