package resticProxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	ser "github.com/kubackup/kubackup/internal/service/v1/task"
	"github.com/kubackup/kubackup/internal/store/task"
	"github.com/kubackup/kubackup/internal/ui/backup"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/archiver"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/textfile"
	"gopkg.in/tomb.v2"
	"os"
	"strings"
	"time"
)

var taskHistoryService ser.Service

func init() {
	taskHistoryService = ser.GetService()
}

var ErrInvalidSourceData = errors.New("at least one source file could not be read")

// BackupOptions bundles all options for the backup command.
type BackupOptions struct {
	Parent                  string
	Force                   bool
	Excludes                []string
	InsensitiveExcludes     []string
	ExcludeFiles            []string
	InsensitiveExcludeFiles []string
	ExcludeOtherFS          bool
	ExcludeIfPresent        []string
	ExcludeCaches           bool
	ExcludeLargerThan       string
	StdinFilename           string
	Tags                    restic.TagLists
	Host                    string
	TimeStamp               string // `time` of the backup (ex. '2012-11-01 22:08:41') (default: now)
	WithAtime               bool
	IgnoreInode             bool
	IgnoreCtime             bool
	UseFsSnapshot           bool
	DryRun                  bool
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

	ctx, cancel := context.WithCancel(repoHandler.gopts.ctx)
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})

	targets := []string{taskinfo.Path}
	timeStamp := time.Now()

	var t tomb.Tomb
	progressPrinter := NewTaskProgress(&taskinfo)
	progressReporter := backup.NewProgress(progressPrinter)
	if opts.DryRun {
		repo.SetDryRun()
		progressReporter.SetDryRun()
	}
	// 设置进度发送频率
	progressReporter.SetMinUpdatePause(time.Second)
	progressPrinter.SetMinUpdatePause(time.Second)
	t.Go(func() error { return progressReporter.Run(t.Context(ctx)) })
	clean.AddCleanCtx(func() {
		t.Kill(nil)
	})
	lock, err := lockRepo(ctx, repo)
	if err != nil {
		clean.Cleanup()
		return err
	}
	clean.AddCleanCtx(func() {
		unlockRepo(lock)
	})
	rejectByNameFuncs, err := collectRejectByNameFuncs(opts, repo, targets)
	if err != nil {
		clean.Cleanup()
		return err
	}
	rejectFuncs, err := collectRejectFuncs(opts, repo, targets)
	if err != nil {
		clean.Cleanup()
		return err
	}
	parentSnapshotID, err := findParentSnapshot(ctx, repo, opts, targets, timeStamp)
	if err != nil {
		clean.Cleanup()
		return err
	}
	err = taskHistoryService.UpdateField(taskinfo.GetId(), "ParentId", parentSnapshotID.Str(), common.DBOptions{})
	if err != nil {
		clean.Cleanup()
		return err
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
	sc := archiver.NewScanner(targetFS)
	sc.SelectByName = selectByNameFilter
	sc.Select = selectFilter
	sc.Error = progressReporter.ScannerError
	sc.Result = progressReporter.ReportTotal

	t.Go(func() error { return sc.Scan(t.Context(ctx), targets) })

	arch := archiver.New(repo, targetFS, archiver.Options{})
	arch.SelectByName = selectByNameFilter
	arch.Select = selectFilter
	arch.WithAtime = opts.WithAtime
	success := true
	arch.Error = func(item string, fi os.FileInfo, err error) error {
		success = false
		return progressReporter.Error(item, fi, err)
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

	if parentSnapshotID == nil {
		parentSnapshotID = &restic.ID{}
	}

	snapshotOpts := archiver.SnapshotOptions{
		Excludes:       opts.Excludes,
		Tags:           opts.Tags.Flatten(),
		Time:           timeStamp,
		Hostname:       opts.Host,
		ParentSnapshot: *parentSnapshotID,
	}
	bound := make(chan error)
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
		progressReporter.Finish(id)
	}()
	return nil

}
func findParentSnapshot(ctx context.Context, repo restic.Repository, opts BackupOptions, targets []string, timeStampLimit time.Time) (parentID *restic.ID, err error) {
	// Force using a parent
	if !opts.Force && opts.Parent != "" {
		id, err := restic.FindSnapshot(ctx, repo, opts.Parent)
		if err != nil {
			return nil, errors.Fatalf("invalid id %q: %v", opts.Parent, err)
		}

		parentID = &id
	}

	// Find last snapshot to set it as parent, if not already set
	if !opts.Force && parentID == nil {
		id, err := restic.FindLatestSnapshot(ctx, repo, targets, []restic.TagList{}, []string{opts.Host}, &timeStampLimit)
		if err == nil {
			parentID = &id
		} else if err != restic.ErrNoSnapshotFound {
			return nil, err
		}
	}

	return parentID, nil
}
func collectRejectFuncs(opts BackupOptions, repo *repository.Repository, targets []string) (fs []RejectFunc, err error) {
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

func collectRejectByNameFuncs(opts BackupOptions, repo *repository.Repository, targets []string) (fs []RejectByNameFunc, err error) {
	// exclude restic cache
	if repo.Cache != nil {
		f, err := rejectResticCache(repo)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
	}

	// add patterns from file
	if len(opts.ExcludeFiles) > 0 {
		excludes, err := readExcludePatternsFromFiles(opts.ExcludeFiles)
		if err != nil {
			return nil, err
		}
		opts.Excludes = append(opts.Excludes, excludes...)
	}

	if len(opts.InsensitiveExcludeFiles) > 0 {
		excludes, err := readExcludePatternsFromFiles(opts.InsensitiveExcludeFiles)
		if err != nil {
			return nil, err
		}
		opts.InsensitiveExcludes = append(opts.InsensitiveExcludes, excludes...)
	}

	if len(opts.InsensitiveExcludes) > 0 {
		fs = append(fs, rejectByInsensitivePattern(opts.InsensitiveExcludes))
	}

	if len(opts.Excludes) > 0 {
		fs = append(fs, rejectByPattern(opts.Excludes))
	}

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

func readExcludePatternsFromFiles(excludeFiles []string) ([]string, error) {
	getenvOrDollar := func(s string) string {
		if s == "$" {
			return "$"
		}
		return os.Getenv(s)
	}

	var excludes []string
	for _, filename := range excludeFiles {
		err := func() (err error) {
			data, err := textfile.Read(filename)
			if err != nil {
				return err
			}

			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())

				// ignore empty lines
				if line == "" {
					continue
				}

				// strip comments
				if strings.HasPrefix(line, "#") {
					continue
				}

				line = os.Expand(line, getenvOrDollar)
				excludes = append(excludes, line)
			}
			return scanner.Err()
		}()
		if err != nil {
			return nil, err
		}
	}
	return excludes, nil
}
