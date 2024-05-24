package cron

import "github.com/robfig/cron/v3"

type BaseJob interface {
	GetType() int
	cron.Job
}

const (
	JOB_TYPE_SYSTEM = 0 // 系统任务，不会被删除
	JOB_TYPE_BACKUP = 1 // 备份任务
	JOB_TYPE_PRUNE  = 2 // 清理任务
)
