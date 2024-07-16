package cron

import (
	"fmt"
	"github.com/kubackup/kubackup/internal/api/v1/policy"
	"github.com/kubackup/kubackup/internal/api/v1/task"
	"github.com/kubackup/kubackup/internal/consts"
	resticProxy "github.com/kubackup/kubackup/restic_proxy"
	"github.com/robfig/cron/v3"
	"strings"
	"time"
)

var c *cron.Cron

func InitCron() {
	c = cron.New(cron.WithSeconds())
	initSystemCronJob()
	c.Start()
	defer c.Stop()
	select {}
}

// initSystemCronJob 初始化系统定时任务
func initSystemCronJob() {
	// 准备首页数据
	_, err := c.AddJob("0 0 0 * * *", SystemJob(func() {
		go resticProxy.GetAllRepoStats()
	}))
	if err != nil {
		fmt.Println(fmt.Errorf("GetAllRepoStats 定时任务启动失败：%s", err))
	}
	// 清理进行中任务
	_, err = c.AddJob("0 */10 * * * *", SystemJob(func() {
		go task.ClearTaskRunning()
	}))
	if err != nil {
		fmt.Println(fmt.Errorf("ClearTaskRunning 定时任务启动失败：%s", err))
	}
	// 执行清理策略
	_, err = c.AddJob("15 15 6 * * *", SystemJob(func() {
		go policy.DoPolicy()
	}))
	if err != nil {
		fmt.Println(fmt.Errorf("DoPolicy 定时任务启动失败：%s", err))
	}
}

func AddJob(cronStr string, job BaseJob) error {
	cronStr = CheckCron(cronStr)
	_, err := c.AddJob(cronStr, job)
	if err != nil {
		return err
	}
	return nil
}

func ClearJob() {
	entries := c.Entries()
	for _, entry := range entries {
		if entry.Job.(BaseJob).GetType() == JOB_TYPE_SYSTEM {
			continue
		}
		c.Remove(entry.ID)
	}
}

func CheckCron(cronStr string) string {
	ts := strings.Fields(cronStr)
	if len(ts) > 6 {
		var res string
		for _, s := range ts[:6] {
			res += s + " "
		}
		return res
	} else {
		return cronStr
	}
}

// GetNextTimes 生成下次执行时间列表
func GetNextTimes(cronStr string) ([]string, error) {
	cronStr = CheckCron(cronStr)
	res := make([]string, 0)
	tmpcron := cron.New(cron.WithSeconds())
	entryID, err := tmpcron.AddFunc(cronStr, func() {

	})
	if err != nil {
		return nil, err
	}
	entry := tmpcron.Entry(entryID)
	nexttime := time.Now()
	for i := 0; i < 5; i++ {
		nexttime = entry.Schedule.Next(nexttime)
		res = append(res, nexttime.Format(consts.Custom))
	}
	return res, nil
}
