package model

import "time"

type BoardInfo struct {
	PlanInfo       PlanInfo       `json:"planInfo"`
	RepositoryInfo RepositoryInfo `json:"repositoryInfo"`
	BackupInfo     BackupInfo     `json:"backupInfo"`  //汇总备份数据明细
	BackupInfos    []BackupInfo   `json:"backupInfos"` //每个仓库备份数据明细
}

//PlanInfo 计划信息
type PlanInfo struct {
	Total        int `json:"total"`        // 计划总数
	RunningCount int `json:"runningCount"` // 运行中数量
}

//RepositoryInfo 仓库信息
type RepositoryInfo struct {
	Total        int `json:"total"`        // 仓库总数
	RunningCount int `json:"runningCount"` // 运行中数量
}

//BackupInfo 备份信息
type BackupInfo struct {
	RepositoryName string    `json:"repositoryName"` //仓库名称
	SnapshotsNum   int       `json:"snapshotsNum"`   //快照数量
	FileTotal      int       `json:"fileTotal"`      //总文件数
	DataSize       uint64    `json:"dataSize"`       //总数据量
	DataSizeStr    string    `json:"dataSizeStr"`    //总数据量
	DataDay        string    `json:"dataDay"`        //数据保护天数
	Time           time.Time `json:"time"`           // 统计时间
	Duration       string    `json:"duration"`       //统计耗时
}
