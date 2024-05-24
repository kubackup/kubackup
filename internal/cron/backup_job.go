package cron

import (
	"github.com/kubackup/kubackup/internal/api/v1/task"
	"github.com/kubackup/kubackup/internal/server"
)

type BackupJob struct {
	PlanId int
}

func (b BackupJob) Run() {
	_, err := task.Backup(b.PlanId)
	if err != nil {
		server.Logger().Error(err)
		return
	}
}

func (b BackupJob) GetType() int {
	return JOB_TYPE_BACKUP
}

var _ BaseJob = &BackupJob{}
