package resticProxy

import (
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/task"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui/restore"
	"github.com/kubackup/kubackup/pkg/utils"
	"math"
	"time"
)

type restorePrinter struct {
	task        wsTaskInfo.WsTaskInfo
	weightCount float64 //数量进度权重
	weightSize  float64 //大小进度权重
	lastUpdate  time.Time
	errors      []model.ErrorUpdate
}

func NewRestorePrinter(t wsTaskInfo.WsTaskInfo) *restorePrinter {
	return &restorePrinter{
		task:        t,
		weightCount: 1,
		weightSize:  1,
	}
}

var _ restore.ProgressPrinter = &restorePrinter{}

func (r *restorePrinter) Update(filesFinished, filesTotal, allBytesWritten, allBytesTotal uint64, duration time.Duration) {
	// 预计剩余时间
	todo := float64(allBytesTotal - allBytesWritten)
	secs := uint64(float64(duration/time.Second) / float64(allBytesWritten) * todo)
	avg := ""
	if duration/time.Second > 0 {
		avg = utils.FormatBytesSpeed(allBytesWritten / uint64(duration/time.Second))
	}

	status := model.StatusUpdate{
		MessageType:      "status",
		SecondsElapsed:   utils.FormatDuration(duration),
		SecondsRemaining: utils.FormatSeconds(secs),
		TotalFiles:       filesTotal,
		FilesDone:        filesFinished,
		TotalBytes:       utils.FormatBytes(allBytesTotal),
		BytesDone:        utils.FormatBytes(allBytesWritten),
		PercentDone:      0,
		AvgSpeed:         avg,
	}

	if allBytesTotal > 0 && filesTotal > 0 {
		denominator := float64(filesTotal)*r.weightCount + float64(allBytesTotal)*r.weightSize
		numerator := float64(filesFinished)*r.weightCount + float64(allBytesWritten)*r.weightSize
		status.PercentDone = numerator / denominator
		status.PercentDone = math.Floor(status.PercentDone*100) / 100
	}
	r.task.(*task.TaskInfo).Progress = &status
	task.TaskInfos.Set(r.task.GetId(), r.task)
	r.task.SendMsg(&status)
}

func (r *restorePrinter) Finish(filesFinished, filesTotal, allBytesWritten, allBytesTotal uint64, duration time.Duration) {
	opt := common.DBOptions{}

	p := &model.StatusUpdate{}
	p.BytesDone = utils.FormatBytes(allBytesWritten)
	p.PercentDone = 1
	p.FilesDone = filesFinished
	p.SecondsRemaining = "0"
	p.SecondsElapsed = utils.FormatDuration(duration)
	err2 := taskHistoryService.UpdateField(r.task.GetId(), "Progress", p, opt)
	if err2 != nil {
		server.Logger().Error(err2)
		return
	}

	summaryOut := &model.SummaryOutput{
		MessageType:         "summary",
		FilesNew:            uint(filesFinished),
		DataAdded:           utils.FormatBytes(allBytesWritten),
		TotalFilesProcessed: uint(filesTotal),
		TotalBytesProcessed: utils.FormatBytes(allBytesTotal),
		TotalDuration:       p.SecondsElapsed,
	}
	r.task.SendMsg(summaryOut)
	err1 := taskHistoryService.UpdateField(r.task.GetId(), "Summary", summaryOut, opt)
	if err1 != nil {
		server.Logger().Error(err1)
		return
	}

	taskhis, err3 := taskHistoryService.Get(r.task.GetId(), opt)
	if err3 != nil {
		server.Logger().Error(err3)
		return
	}
	status := task.StatusEnd
	if taskhis.ScannerError != nil || taskhis.RestoreError != nil {
		status = task.StatusError
	}
	_ = taskHistoryService.UpdateField(r.task.GetId(), "Status", status, common.DBOptions{})
	task.TaskInfos.Close(r.task.GetId(), "process end", 1)
}

func (r *restorePrinter) SetWeight(weightCount, weightSize float64) {
	r.weightSize = weightSize
	r.weightCount = weightCount
}

func (r *restorePrinter) UpdateTaskInfo(task wsTaskInfo.WsTaskInfo) {
	r.task = task
}

func (r *restorePrinter) Error(item string, err error) error {
	errorUpdate := model.ErrorUpdate{
		MessageType: "error",
		Error:       err.Error(),
		During:      "restore",
		Item:        item,
	}
	if len(r.errors) > 20 {
		return err
	}
	r.task.SendMsg(&errorUpdate)
	r.errors = append(r.errors, errorUpdate)
	err1 := taskHistoryService.UpdateField(r.task.GetId(), "RestoreError", r.errors, common.DBOptions{})
	if err1 != nil {
		return err1
	}
	return err
}
func (r *restorePrinter) ScannerError(err error) error {
	errorUpdate := &model.ErrorUpdate{
		MessageType: "error",
		Error:       err.Error(),
		During:      "scan",
	}
	r.task.SendMsg(&errorUpdate)
	err1 := taskHistoryService.UpdateField(r.task.GetId(), "ScannerError", errorUpdate, common.DBOptions{})
	if err1 != nil {
		return err1
	}
	return nil
}
func (r *restorePrinter) ReportTotal(start time.Time, totalSize, totalCount uint64) {
	ver := &model.VerboseUpdate{
		MessageType: "verbose_status",
		Action:      "scan_finished",
		Duration:    utils.FormatDuration(time.Since(start)),
		DataSize:    utils.FormatBytes(totalSize),
		TotalFiles:  uint(totalCount),
	}
	r.task.SendMsg(ver)
	err := taskHistoryService.UpdateField(r.task.GetId(), "Scanner", ver, common.DBOptions{})
	if err != nil {
		return
	}
}

func (r *restorePrinter) Print(msg string) {
	r.task.SendMsg(msg)
}

func (r *restorePrinter) ReportVerify(msg string) {
	ver := &model.VerboseUpdate{
		MessageType: "summary",
		Action:      "verify_finished",
		Item:        msg,
	}
	r.task.SendMsg(ver)
}
