package resticProxy

import (
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/task"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/internal/ui/backup"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/archiver"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui"
	"github.com/kubackup/kubackup/pkg/utils"
	"os"
	"sort"
	"time"
)

type TaskProgress struct {
	*ui.StdioWrapper
	task           wsTaskInfo.WsTaskInfo
	MinUpdatePause time.Duration
	lastUpdate     time.Time
	errors         []model.ErrorUpdate
}

func (t *TaskProgress) E(msg string, args ...interface{}) {
	errorUpdate := model.ErrorUpdate{
		MessageType: "error",
		Error:       msg,
		During:      "backup",
		Item:        "",
	}
	if len(t.errors) > 20 {
		return
	}
	t.print(errorUpdate, true)
	t.errors = append(t.errors, errorUpdate)
	_ = taskHistoryService.UpdateField(t.task.GetId(), "ArchivalError", t.errors, common.DBOptions{})
}

func (t *TaskProgress) P(msg string, args ...interface{}) {
	t.print(msg, true)
}

func (t *TaskProgress) V(msg string, args ...interface{}) {
	t.print(msg, true)
}

func (t *TaskProgress) VV(msg string, args ...interface{}) {
	t.print(msg, true)
}

var _ backup.ProgressPrinter = &TaskProgress{}

func NewTaskProgress(task wsTaskInfo.WsTaskInfo) *TaskProgress {
	return &TaskProgress{
		task:           task,
		errors:         make([]model.ErrorUpdate, 0),
		MinUpdatePause: time.Second,
	}
}
func (t *TaskProgress) UpdateTaskInfo(task wsTaskInfo.WsTaskInfo) {
	t.task = task
}
func (t *TaskProgress) SetMinUpdatePause(d time.Duration) {
	t.MinUpdatePause = d
}

func (t *TaskProgress) print(status interface{}, forceUpdate bool) {
	//控制发送频率
	if !forceUpdate && (time.Since(t.lastUpdate) < t.MinUpdatePause || t.MinUpdatePause == 0) {
		return
	}
	t.lastUpdate = time.Now()
	t.task.SendMsg(status)
}

func (t *TaskProgress) Update(total, processed backup.Counter, avgSpeed uint64, errors uint, currentFiles map[string]struct{}, start time.Time, secs uint64) {
	status := model.StatusUpdate{
		MessageType:      "status",
		SecondsElapsed:   utils.FormatDuration(time.Since(start)),
		SecondsRemaining: utils.FormatSeconds(secs),
		TotalFiles:       total.Files,
		FilesDone:        processed.Files,
		TotalBytes:       utils.FormatBytes(total.Bytes),
		BytesDone:        utils.FormatBytes(processed.Bytes),
		ErrorCount:       errors,
		AvgSpeed:         utils.FormatBytesSpeed(avgSpeed),
	}

	if total.Bytes > 0 {
		status.PercentDone = float64(processed.Bytes) / float64(total.Bytes)
	}

	for filename := range currentFiles {
		status.CurrentFiles = append(status.CurrentFiles, filename)
	}
	sort.Strings(status.CurrentFiles)
	t.task.(*task.TaskInfo).Progress = &status
	task.TaskInfos.Set(t.task.GetId(), t.task)
	t.print(&status, true)
}

func (t *TaskProgress) ScannerError(item string, fi os.FileInfo, err error) error {
	errorUpdate := &model.ErrorUpdate{
		MessageType: "error",
		Error:       err.Error(),
		During:      "scan",
		Item:        item,
	}
	t.print(errorUpdate, true)
	err1 := taskHistoryService.UpdateField(t.task.GetId(), "ScannerError", errorUpdate, common.DBOptions{})
	if err1 != nil {
		return err1
	}
	return err
}

func (t *TaskProgress) Error(item string, fi os.FileInfo, err error) error {
	errorUpdate := model.ErrorUpdate{
		MessageType: "error",
		Error:       err.Error(),
		During:      "archival",
		Item:        item,
	}
	if len(t.errors) > 20 {
		return err
	}
	t.print(&errorUpdate, true)
	t.errors = append(t.errors, errorUpdate)
	err1 := taskHistoryService.UpdateField(t.task.GetId(), "ArchivalError", t.errors, common.DBOptions{})
	if err1 != nil {
		return err1
	}
	return err
}

func (t *TaskProgress) CompleteItem(messageType, item string, previous, current *restic.Node, s archiver.ItemStats, d time.Duration) {
	var status model.VerboseUpdate
	switch messageType {
	case "dir new":
		status = model.VerboseUpdate{
			MessageType:  "verbose_status",
			Action:       "new",
			Item:         item,
			Duration:     utils.FormatDuration(d),
			DataSize:     utils.FormatBytes(s.DataSize),
			MetadataSize: utils.FormatBytes(s.TreeSize),
		}
	case "dir unchanged":
		status = model.VerboseUpdate{
			MessageType: "verbose_status",
			Action:      "unchanged",
			Item:        item,
		}
	case "dir modified":
		status = model.VerboseUpdate{
			MessageType:  "verbose_status",
			Action:       "modified",
			Item:         item,
			Duration:     utils.FormatDuration(d),
			DataSize:     utils.FormatBytes(s.DataSize),
			MetadataSize: utils.FormatBytes(s.TreeSize),
		}
	case "file new":
		status = model.VerboseUpdate{
			MessageType: "verbose_status",
			Action:      "new",
			Item:        item,
			Duration:    utils.FormatDuration(d),
			DataSize:    utils.FormatBytes(s.DataSize),
		}
	case "file unchanged":
		status = model.VerboseUpdate{
			MessageType: "verbose_status",
			Action:      "unchanged",
			Item:        item,
		}
	case "file modified":
		status = model.VerboseUpdate{
			MessageType: "verbose_status",
			Action:      "modified",
			Item:        item,
			Duration:    utils.FormatDuration(d),
			DataSize:    utils.FormatBytes(s.DataSize),
		}
	}
	t.print(&status, false)
}

func (t *TaskProgress) ReportTotal(item string, start time.Time, s archiver.ScanStats) {
	ver := &model.VerboseUpdate{
		MessageType: "verbose_status",
		Action:      "scan_finished",
		Duration:    utils.FormatDuration(time.Since(start)),
		DataSize:    utils.FormatBytes(s.Bytes),
		TotalFiles:  s.Files,
	}
	t.print(ver, true)
	err := taskHistoryService.UpdateField(t.task.GetId(), "Scanner", ver, common.DBOptions{})
	if err != nil {
		return
	}
}
func (t *TaskProgress) Finish(snapshotID restic.ID, start time.Time, summary *backup.Summary, dryRun bool) {
	var summaryOut *model.SummaryOutput
	var p1 *model.StatusUpdate
	if summary != nil {
		summaryOut = &model.SummaryOutput{
			MessageType:         "summary",
			FilesNew:            summary.Files.New,
			FilesChanged:        summary.Files.Changed,
			FilesUnmodified:     summary.Files.Unchanged,
			DirsNew:             summary.Dirs.New,
			DirsChanged:         summary.Dirs.Changed,
			DirsUnmodified:      summary.Dirs.Unchanged,
			DataBlobs:           summary.ItemStats.DataBlobs,
			TreeBlobs:           summary.ItemStats.TreeBlobs,
			DataAdded:           utils.FormatBytes(summary.ItemStats.DataSize + summary.ItemStats.TreeSize),
			TotalFilesProcessed: summary.Files.New + summary.Files.Changed + summary.Files.Unchanged,
			TotalBytesProcessed: utils.FormatBytes(summary.ProcessedBytes),
			TotalDuration:       utils.FormatDuration(time.Since(start)),
			SnapshotID:          snapshotID.Str(),
			DryRun:              dryRun,
		}
		t.print(summaryOut, true)
		p1 = t.task.(*task.TaskInfo).Progress
		if p1 != nil {
			p1.BytesDone = p1.TotalBytes
			p1.PercentDone = 1
			p1.FilesDone = p1.TotalFiles
			p1.SecondsRemaining = "0"
			p1.SecondsElapsed = summaryOut.TotalDuration
			sec := uint64(time.Since(start) / time.Second)
			if sec > 0 {
				p1.AvgSpeed = utils.FormatBytesSpeed(summary.ProcessedBytes / sec)
			}
			t.print(p1, true)
		}
	}
	taskhis, err3 := taskHistoryService.Get(t.task.GetId(), common.DBOptions{})
	if err3 != nil {
		server.Logger().Error(err3)
		return
	}
	status := task.StatusEnd
	if taskhis.ScannerError != nil || len(taskhis.ArchivalError) > 0 {
		status = task.StatusError
	}
	taskhis.Status = status
	taskhis.Summary = summaryOut
	taskhis.Progress = p1
	_ = taskHistoryService.Update(taskhis, common.DBOptions{})
	task.TaskInfos.Close(t.task.GetId(), "process end", 1)
	go GetAllRepoStats()
}
func (t *TaskProgress) Reset() {

}
