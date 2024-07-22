package resticProxy

import (
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	ser "github.com/kubackup/kubackup/internal/service/v1/task"
	"github.com/kubackup/kubackup/internal/store/task"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/archiver"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui/backup"
	"gopkg.in/tomb.v2"
	"os"
	"runtime"
	"time"
)

var taskHistoryService ser.Service

func init() {
	taskHistoryService = ser.GetService()
}

var ErrInvalidSourceData = errors.New("at least one source file could not be read")

// BackupOptions bundles all options for the backup command.
type BackupOptions struct {
	excludePatternOptions
	Parent            string
	GroupBy           restic.SnapshotGroupByOptions
	Force             bool
	ExcludeOtherFS    bool
	ExcludeIfPresent  []string
	ExcludeCaches     bool
	ExcludeLargerThan string
	Tags              restic.TagLists
	Host              string
	TimeStamp         string // `time` of the backup (ex. '2012-11-01 22:08:41') (default: now)
	WithAtime         bool
	IgnoreInode       bool
	IgnoreCtime       bool
	UseFsSnapshot     bool
	DryRun            bool
	ReadConcurrency   uint //读取并发数量，默认2
}

func RunBackup(opts BackupOptions, repoid int, taskinfo task.TaskInfo) error {
	if opts.Host == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("os.Hostname() returned err: %v", err)
		}
		opts.Host = hostname
	}
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return err
	}
	repo := repoHandler.repo

	ctx, cancel := context.WithCancel(context.Background())
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})

	targets := []string{taskinfo.Path}
	timeStamp := time.Now()

	var t tomb.Tomb
	progressPrinter := NewTaskProgress(&taskinfo, time.Second)
	progressReporter := backup.NewProgress(progressPrinter, time.Second)
	clean.AddCleanCtx(func() {
		progressReporter.Done()
	})
	if opts.DryRun {
		repo.SetDryRun()
	}
	lock, err := lockRepo(ctx, repo)
	if err != nil {
		clean.Cleanup()
		return err
	}
	clean.AddCleanCtx(func() {
		unlockRepo(lock)
	})
	rejectByNameFuncs, err := collectRejectByNameFuncs(opts, repo)
	if err != nil {
		clean.Cleanup()
		return err
	}
	rejectFuncs, err := collectRejectFuncs(opts, targets)
	if err != nil {
		clean.Cleanup()
		return err
	}
	parentSnapshot, err := findParentSnapshot(ctx, repo, opts, targets, timeStamp)
	if err != nil {
		clean.Cleanup()
		return err
	}
	if parentSnapshot != nil {
		err = taskHistoryService.UpdateField(taskinfo.GetId(), "ParentId", parentSnapshot.ID().Str(), common.DBOptions{})
		if err != nil {
			clean.Cleanup()
			return err
		}
	}

	selectByNameFilter := func(item string) bool {
		for _, reject := range rejectByNameFuncs {
			if reject(item) {
				return false
			}
		}
		return true
	}

	selectFilter := func(item string, fi os.FileInfo) bool {
		for _, reject := range rejectFuncs {
			if reject(item, fi) {
				return false
			}
		}
		return true
	}
	var targetFS fs.FS = fs.Local{}
	if runtime.GOOS == "windows" && opts.UseFsSnapshot {
		if err = fs.HasSufficientPrivilegesForVSS(); err != nil {
			return err
		}

		errorHandler := func(item string, err error) error {
			return progressReporter.Error(item, err)
		}

		messageHandler := func(msg string, args ...interface{}) {
			progressPrinter.P(msg, args...)
		}

		localVss := fs.NewLocalVss(errorHandler, messageHandler)
		defer localVss.DeleteSnapshots()
		targetFS = localVss
	}
	sc := archiver.NewScanner(targetFS)
	sc.SelectByName = selectByNameFilter
	sc.Select = selectFilter
	sc.Error = progressPrinter.ScannerError
	sc.Result = progressReporter.ReportTotal

	t.Go(func() error { return sc.Scan(t.Context(ctx), targets) })

	arch := archiver.New(repo, targetFS, archiver.Options{
		ReadConcurrency: opts.ReadConcurrency,
	})
	arch.SelectByName = selectByNameFilter
	arch.Select = selectFilter
	arch.WithAtime = opts.WithAtime
	success := true
	arch.Error = func(item string, err error) error {
		success = false
		return progressReporter.Error(item, err)
	}
	arch.CompleteItem = progressReporter.CompleteItem
	arch.StartFile = progressReporter.StartFile
	arch.CompleteBlob = progressReporter.CompleteBlob

	if opts.IgnoreInode {
		// --ignore-inode implies --ignore-ctime: on FUSE, the ctime is not
		// reliable either.
		arch.ChangeIgnoreFlags |= archiver.ChangeIgnoreCtime | archiver.ChangeIgnoreInode
	}
	if opts.IgnoreCtime {
		arch.ChangeIgnoreFlags |= archiver.ChangeIgnoreCtime
	}

	snapshotOpts := archiver.SnapshotOptions{
		Excludes:       opts.Excludes,
		Tags:           opts.Tags.Flatten(),
		Time:           timeStamp,
		Hostname:       opts.Host,
		ParentSnapshot: parentSnapshot,
		ProgramVersion: "restic " + version,
	}
	bound := make(chan string)
	taskinfo.SetBound(bound)
	task.TaskInfos.Set(taskinfo.GetId(), &taskinfo)
	t.Go(func() error {
		for {
			select {
			case <-t.Context(ctx).Done():
				return nil
			case <-task.TaskInfos.Get(taskinfo.GetId()).GetBound():
				info := task.TaskInfos.Get(taskinfo.GetId())
				progressPrinter.UpdateTaskInfo(info)
			}
		}
	})
	if !BackupLock(repoid, taskinfo.Path) {
		clean.Cleanup()
		return fmt.Errorf("存储库\"%d\"正在备份：%s", repoid, taskinfo.Path)
	}
	clean.AddCleanCtx(func() {
		BackupUnLock(repoid, taskinfo.Path)
	})
	go func() {
		defer clean.Cleanup()
		err = taskHistoryService.UpdateField(taskinfo.GetId(), "Status", task.StatusRunning, common.DBOptions{})
		if err != nil {
			server.Logger().Error(err)
		}
		_, id, err := arch.Snapshot(ctx, targets, snapshotOpts)
		if err != nil {
			progressPrinter.E(fmt.Errorf("unable to save snapshot: %v", err).Error())
		}
		t.Kill(nil)
		werr := t.Wait()
		if werr != nil {
			server.Logger().Error(werr)
		}
		if !success {
			progressPrinter.E(ErrInvalidSourceData.Error())
		}
		progressReporter.Finish(id, opts.DryRun)
	}()
	return nil

}
func findParentSnapshot(ctx context.Context, repo restic.Repository, opts BackupOptions, targets []string, timeStampLimit time.Time) (*restic.Snapshot, error) {
	if opts.Force {
		return nil, nil
	}

	snName := opts.Parent
	if snName == "" {
		snName = "latest"
	}
	f := restic.SnapshotFilter{TimestampLimit: timeStampLimit}
	if opts.GroupBy.Host {
		f.Hosts = []string{opts.Host}
	}
	if opts.GroupBy.Path {
		f.Paths = targets
	}
	if opts.GroupBy.Tag {
		f.Tags = []restic.TagList{opts.Tags.Flatten()}
	}

	sn, _, err := f.FindLatest(ctx, repo.Backend(), repo, snName)
	// Snapshot not found is ok if no explicit parent was set
	if opts.Parent == "" && errors.Is(err, restic.ErrNoSnapshotFound) {
		err = nil
	}
	return sn, err
}
func collectRejectFuncs(opts BackupOptions, targets []string) (fs []RejectFunc, err error) {
	// allowed devices
	if opts.ExcludeOtherFS {
		f, err := rejectByDevice(targets)
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}

	if len(opts.ExcludeLargerThan) != 0 {
		f, err := rejectBySize(opts.ExcludeLargerThan)
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}

	return fs, nil
}

func collectRejectByNameFuncs(opts BackupOptions, repo *repository.Repository) (fs []RejectByNameFunc, err error) {
	// exclude restic cache
	if repo.Cache != nil {
		f, err := rejectResticCache(repo)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
	}

	fsPatterns, err := opts.excludePatternOptions.CollectPatterns()
	if err != nil {
		return nil, err
	}
	fs = append(fs, fsPatterns...)

	if opts.ExcludeCaches {
		opts.ExcludeIfPresent = append(opts.ExcludeIfPresent, "CACHEDIR.TAG:Signature: 8a477f597d28d172789f06886806bc55")
	}

	for _, spec := range opts.ExcludeIfPresent {
		f, err := rejectIfPresent(spec)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
	}

	return fs, nil
}
