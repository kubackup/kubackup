package resticProxy

import (
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/task"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	ui "github.com/kubackup/kubackup/internal/ui/restore"
	"github.com/kubackup/kubackup/pkg/utils"
	"math"
	"time"
)

type RestoreProgress struct {
	ui.ProgressPrinter
	task           wsTaskInfo.WsTaskInfo
	minUpdatePause time.Duration
	weightCount    float64 //数量进度权重
	weightSize     float64 //大小进度权重
	lastUpdate     time.Time
	errors         []model.ErrorUpdate
}

func NewRestoreProgress(t wsTaskInfo.WsTaskInfo) *RestoreProgress {
	return &RestoreProgress{
		task:           t,
		errors:         make([]model.ErrorUpdate, 0),
		minUpdatePause: time.Second,
		weightCount:    1,
		weightSize:     1,
	}
}

func (r *RestoreProgress) SetMinUpdatePause(d time.Duration) {
	r.minUpdatePause = d
}

func (r *RestoreProgress) SetWeight(weightCount, weightSize float64) {
	r.weightSize = weightSize
	r.weightCount = weightCount
}

func (r *RestoreProgress) print(status interface{}, forceUpdate bool) {
	//控制发送频率
	if !forceUpdate && (time.Since(r.lastUpdate) < r.minUpdatePause || r.minUpdatePause == 0) {
		return
	}
	r.lastUpdate = time.Now()
	r.task.SendMsg(status)
}

func (r *RestoreProgress) UpdateTaskInfo(task wsTaskInfo.WsTaskInfo) {
	r.task = task
}

func (r *RestoreProgress) Update(total, processed ui.Counter, avgSpeed uint64, errors uint, start time.Time, secs uint64) {
	status := model.StatusUpdate{
		MessageType:      "status",
		SecondsElapsed:   utils.FormatDuration(time.Since(start)),
		SecondsRemaining: utils.FormatSeconds(secs),
		TotalFiles:       uint64(total.Files),
		FilesDone:        uint64(processed.Files),
		TotalBytes:       utils.FormatBytes(total.Bytes),
		BytesDone:        utils.FormatBytes(processed.Bytes),
		ErrorCount:       errors,
		PercentDone:      0,
		AvgSpeed:         utils.FormatBytesSpeed(avgSpeed),
	}

	if total.Bytes > 0 && total.Files > 0 {
		denominator := float64(total.Files)*r.weightCount + float64(total.Bytes)*r.weightSize
		numerator := float64(processed.Files)*r.weightCount + float64(processed.Bytes)*r.weightSize
		status.PercentDone = numerator / denominator
		status.PercentDone = math.Floor(status.PercentDone*100) / 100
	}
	r.task.(*task.TaskInfo).Progress = &status
	task.TaskInfos.Set(r.task.GetId(), r.task)
	r.print(&status, false)
}
func (r *RestoreProgress) Error(item string, err error) error {
	errorUpdate := model.ErrorUpdate{
		MessageType: "error",
		Error:       err.Error(),
		During:      "restore",
		Item:        item,
	}
	if len(r.errors) > 20 {
		return err
	}
	r.print(&errorUpdate, true)
	r.errors = append(r.errors, errorUpdate)
	err1 := taskHistoryService.UpdateField(r.task.GetId(), "RestoreError", r.errors, common.DBOptions{})
	if err1 != nil {
		return err1
	}
	return err
}
func (r *RestoreProgress) ScannerError(err error) error {
	errorUpdate := &model.ErrorUpdate{
		MessageType: "error",
		Error:       err.Error(),
		During:      "scan",
	}
	r.print(&errorUpdate, true)
	err1 := taskHistoryService.UpdateField(r.task.GetId(), "ScannerError", errorUpdate, common.DBOptions{})
	if err1 != nil {
		return err1
	}
	return nil
}
func (r *RestoreProgress) ReportTotal(start time.Time, total ui.Counter) {
	ver := &model.VerboseUpdate{
		MessageType: "verbose_status",
		Action:      "scan_finished",
		Duration:    utils.FormatDuration(time.Since(start)),
		DataSize:    utils.FormatBytes(total.Bytes),
		TotalFiles:  total.Files,
	}
	r.print(ver, false)
	err := taskHistoryService.UpdateField(r.task.GetId(), "Scanner", ver, common.DBOptions{})
	if err != nil {
		return
	}
}
func (r *RestoreProgress) Finish(snapshotID string, start time.Time, summary *ui.Counter) {
	summaryOut := &model.SummaryOutput{
		MessageType:         "summary",
		DataAdded:           utils.FormatBytes(summary.Bytes),
		TotalFilesProcessed: summary.Files,
		TotalBytesProcessed: utils.FormatBytes(summary.Bytes),
		TotalDuration:       utils.FormatDuration(time.Since(start)),
		SnapshotID:          snapshotID,
	}
	r.print(summaryOut, true)
	opt := common.DBOptions{}
	err1 := taskHistoryService.UpdateField(r.task.GetId(), "Summary", summaryOut, opt)
	if err1 != nil {
		server.Logger().Error(err1)
		return
	}
	p1 := r.task.(*task.TaskInfo).Progress
	p := &model.StatusUpdate{}
	p.BytesDone = utils.FormatBytes(summary.Bytes)
	p.PercentDone = 1
	p.FilesDone = p1.TotalFiles
	p.SecondsRemaining = "0"
	p.SecondsElapsed = summaryOut.TotalDuration
	sec := uint64(time.Since(start) / time.Second)
	if sec > 0 {
		p.AvgSpeed = utils.FormatBytesSpeed(summary.Bytes / sec)
	}

	err2 := taskHistoryService.UpdateField(r.task.GetId(), "Progress", p, opt)
	if err2 != nil {
		server.Logger().Error(err2)
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
