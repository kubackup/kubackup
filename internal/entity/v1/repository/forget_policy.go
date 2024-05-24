package repository

import (
	"github.com/kubackup/kubackup/internal/entity/v1/common"
)

type ForgetPolicy struct {
	common.BaseModel `storm:"inline"`
	RepositoryId     int    `json:"repositoryId"`
	Path             string `json:"path"` // 路径
	Status           int    `json:"status"`
	/**
	类型
	last
	hourly
	daily
	weekly
	monthly
	yearly
	*/
	Type string `json:"type"`
	// 值
	Value int `json:"value"`
}
