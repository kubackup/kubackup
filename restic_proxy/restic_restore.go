package resticProxy

import (
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/consts"
	thmodel "github.com/kubackup/kubackup/internal/entity/v1/task"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/task"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/filter"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restorer"
	restoreui "github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui/restore"
	"gopkg.in/tomb.v2"
	"strconv"
	"strings"
	"time"
)

// RestoreOptions collects all options for the restore command.
type RestoreOptions struct {
	Exclude            []string //exclude a `pattern` (can be specified multiple times)
	InsensitiveExclude []string //same as `--exclude` but ignores the casing of filenames
	Include            []string //include a `pattern`, exclude everything else (can be specified multiple times)
	InsensitiveInclude []string //same as `--include` but ignores the casing of filenames
	Target             string   //directory to extract data to
	restic.SnapshotFilter
	Sparse bool //restore files as sparse
	Verify bool //verify restored files content
}

func RunRestore(opts RestoreOptions, repoid int, snapshotid string) error {
	if snapshotid == "" {
		return errors.Fatal("snapshotid不能为空")
	}
	hasExcludes := len(opts.Exclude) > 0 || len(opts.InsensitiveExclude) > 0
	hasIncludes := len(opts.Include) > 0 || len(opts.InsensitiveInclude) > 0

	// Validate provided patterns
	if len(opts.Exclude) > 0 {
		if err := filter.ValidatePatterns(opts.Exclude); err != nil {
			return errors.Fatalf("--exclude: %s", err)
		}
	}
	if len(opts.InsensitiveExclude) > 0 {
		if err := filter.ValidatePatterns(opts.InsensitiveExclude); err != nil {
			return errors.Fatalf("--iexclude: %s", err)
		}
	}
	if len(opts.Include) > 0 {
		if err := filter.ValidatePatterns(opts.Include); err != nil {
			return errors.Fatalf("--include: %s", err)
		}
	}
	if len(opts.InsensitiveInclude) > 0 {
		if err := filter.ValidatePatterns(opts.InsensitiveInclude); err != nil {
			return errors.Fatalf("--iinclude: %s", err)
		}
	}

	for i, str := range opts.InsensitiveExclude {
		opts.InsensitiveExclude[i] = strings.ToLower(str)
	}

	for i, str := range opts.InsensitiveInclude {
		opts.InsensitiveInclude[i] = strings.ToLower(str)
	}
	if opts.Target == "" {
		return errors.Fatal("please specify a directory to restore to (--target)")
	}

	if hasExcludes && hasIncludes {
		return errors.Fatal("exclude and include patterns are mutually exclusive")
	}
	server.Logger().Debugf("restore %s to %s", snapshotid, opts.Target)
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

	ta, err := createRestoreTask(opts.Target, repoid)
	if err != nil {
		return err
	}

	var t tomb.Tomb
	taskInfo := task.TaskInfo{
		Name: ta.Name,
		Path: ta.Path,
	}
	taskInfo.SetId(ta.Id)
	printer := NewRestorePrinter(&taskInfo)
	progressReporter := restoreui.NewProgress(printer, 1)
	// 设置进度发送频率
	printer.SetWeight(4, 1)

	sn, subfolder, err := (&restic.SnapshotFilter{
		Hosts: opts.Hosts,
		Paths: opts.Paths,
		Tags:  opts.Tags,
	}).FindLatest(ctx, repo.Backend(), repo, snapshotid)
	if err != nil {
		return errors.Fatalf("failed to find snapshot: %v", err)
	}

	sn.Tree, err = restic.FindTreeDirectory(ctx, repo, sn.Tree, subfolder)
	if err != nil {
		return err
	}
	start := time.Now()
	// 获取数据总数
	t.Go(func() error {
		stats, err := getStatsForSnapshots(ctx, repo, sn)
		if err != nil {
			return printer.ScannerError(err)
		}
		printer.ReportTotal(start, stats.TotalSize, stats.TotalFileCount)
		return nil
	})

	res := restorer.NewRestorer(repo, sn, opts.Sparse, progressReporter)

	res.Error = func(location string, err error) error {
		return printer.Error(location, err)
	}

	excludePatterns := filter.ParsePatterns(opts.Exclude)
	insensitiveExcludePatterns := filter.ParsePatterns(opts.InsensitiveExclude)
	selectExcludeFilter := func(item string, dstpath string, node *restic.Node) (selectedForRestore bool, childMayBeSelected bool) {
		matched, err := filter.List(excludePatterns, item)
		if err != nil {
			server.Logger().Warnf("error for exclude pattern: %v", err)
		}

		matchedInsensitive, err := filter.List(insensitiveExcludePatterns, strings.ToLower(item))
		if err != nil {
			server.Logger().Warnf("error for iexclude pattern: %v", err)
		}

		// An exclude filter is basically a 'wildcard but foo',
		// so even if a childMayMatch, other children of a dir may not,
		// therefore childMayMatch does not matter, but we should not go down
		// unless the dir is selected for restore
		selectedForRestore = !matched && !matchedInsensitive
		childMayBeSelected = selectedForRestore && node.Type == "dir"

		return selectedForRestore, childMayBeSelected
	}

	includePatterns := filter.ParsePatterns(opts.Include)
	insensitiveIncludePatterns := filter.ParsePatterns(opts.InsensitiveInclude)
	selectIncludeFilter := func(item string, dstpath string, node *restic.Node) (selectedForRestore bool, childMayBeSelected bool) {
		matched, childMayMatch, err := filter.ListWithChild(includePatterns, item)
		if err != nil {
			server.Logger().Warnf("error for include pattern: %v", err)
		}

		matchedInsensitive, childMayMatchInsensitive, err := filter.ListWithChild(insensitiveIncludePatterns, strings.ToLower(item))
		if err != nil {
			server.Logger().Warnf("error for iexclude pattern: %v", err)
		}

		selectedForRestore = matched || matchedInsensitive
		childMayBeSelected = (childMayMatch || childMayMatchInsensitive) && node.Type == "dir"

		return selectedForRestore, childMayBeSelected
	}

	if hasExcludes {
		res.SelectFilter = selectExcludeFilter
	} else if hasIncludes {
		res.SelectFilter = selectIncludeFilter
	}

	server.Logger().Debugf("restoring %s to %s\n", res.Snapshot().ID().Str(), opts.Target)
	taskinfoid := ta.Id
	bound := make(chan error)
	taskInfo.SetBound(bound)
	task.TaskInfos.Set(taskInfo.GetId(), &taskInfo)
	t.Go(func() error {
		for {
			select {
			case <-t.Context(ctx).Done():
				return nil
			case <-task.TaskInfos.Get(taskInfo.GetId()).GetBound():
				info := task.TaskInfos.Get(taskInfo.GetId())
				printer.UpdateTaskInfo(info)
			}
		}
	})
	go func() {
		defer clean.Cleanup()
		err = taskHistoryService.UpdateField(taskinfoid, "Status", task.StatusRunning, common.DBOptions{})
		if err != nil {
			server.Logger().Error(err)
		}
		err = res.RestoreTo(ctx, opts.Target)
		if err != nil {
			server.Logger().Error(err)
			_ = printer.Error("RestoreTo", err)
		}
		progressReporter.Finish()
		if opts.Verify {
			server.Logger().Debugf("verifying files in %s\n", opts.Target)
			t0 := time.Now()
			count, err := res.VerifyFiles(ctx, opts.Target)
			if err != nil {
				_ = printer.Error("VerifyFiles", err)
			}
			server.Logger().Debugf("finished verifying %d files in %s (took %s)\n", count, opts.Target,
				time.Since(t0).Round(time.Millisecond))
			printer.Print(fmt.Sprintf("finished verifying %d files in %s (took %s)\n", count, opts.Target,
				time.Since(t0).Round(time.Millisecond)))
		}
		t.Kill(nil)
		werr := t.Wait()
		if werr != nil {
			server.Logger().Error(werr)
		}
	}()
	return nil

}

func createRestoreTask(target string, repository int) (*thmodel.Task, error) {
	progress := &model.StatusUpdate{
		MessageType:      "status",
		SecondsElapsed:   "0",
		SecondsRemaining: "0",
		TotalFiles:       0,
		FilesDone:        0,
		TotalBytes:       "0",
		BytesDone:        "0",
		ErrorCount:       0,
		PercentDone:      0,
	}
	t := &thmodel.Task{
		Path:         target,
		Name:         "restore_" + strconv.Itoa(repository) + "_" + time.Now().Format(consts.TaskHistoryName),
		Status:       task.StatusNew,
		RepositoryId: repository,
		Progress:     progress,
	}
	err := taskHistoryService.Create(t, common.DBOptions{})
	if err != nil {
		return nil, err
	}
	return t, nil
}

func getStatsForSnapshots(ctx context.Context, repo restic.Repository, sn *restic.Snapshot) (*StatsContainer, error) {
	opt := StatsOptions{
		countMode: countModeRestoreSize,
	}
	stats := &StatsContainer{
		uniqueFiles:    make(map[fileID]struct{}),
		uniqueInodes:   make(map[uint64]struct{}),
		fileBlobs:      make(map[string]restic.IDSet),
		blobs:          restic.NewBlobSet(),
		snapshotsCount: 0,
	}

	err := statsWalkSnapshot(opt, ctx, sn, repo, stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
