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
	ui "github.com/kubackup/kubackup/internal/ui/restore"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/archiver"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/filter"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restorer"
	"gopkg.in/tomb.v2"
	"os"
	"strconv"
	"strings"
	"time"
)

// RestoreOptions collects all options for the restore command.
type RestoreOptions struct {
	Exclude            []string        //exclude a `pattern` (can be specified multiple times)
	InsensitiveExclude []string        //same as `--exclude` but ignores the casing of filenames
	Include            []string        //include a `pattern`, exclude everything else (can be specified multiple times)
	InsensitiveInclude []string        //same as `--include` but ignores the casing of filenames
	Target             string          //directory to extract data to
	Hosts              []string        //only consider snapshots for this host when the snapshot ID is "latest" (can be specified multiple times)
	Paths              []string        //only consider snapshots which include this (absolute) `path` for snapshot ID "latest"
	Tags               restic.TagLists //only consider snapshots which include this `taglist` for snapshot ID "latest"
	Verify             bool            //verify restored files content
}

func RunRestore(opts RestoreOptions, repoid int, snapshotid string) error {
	if snapshotid == "" {
		return errors.Fatal("snapshotid不能为空")
	}
	hasExcludes := len(opts.Exclude) > 0 || len(opts.InsensitiveExclude) > 0
	hasIncludes := len(opts.Include) > 0 || len(opts.InsensitiveInclude) > 0

	for i, str := range opts.InsensitiveExclude {
		opts.InsensitiveExclude[i] = strings.ToLower(str)
	}

	for i, str := range opts.InsensitiveInclude {
		opts.InsensitiveInclude[i] = strings.ToLower(str)
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

	//err = LoadIndex(ctx, repo)
	//if err != nil {
	//	return err
	//}
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
	progress := NewRestoreProgress(&taskInfo)
	progressReporter := ui.NewProgress(progress)
	// 设置进度发送频率
	progress.SetMinUpdatePause(time.Second)
	progress.SetWeight(4, 1)
	progressReporter.SetMinUpdatePause(time.Second)
	t.Go(func() error { return progressReporter.Run(t.Context(ctx)) })
	clean.AddCleanCtx(func() {
		t.Kill(nil)
	})
	var id restic.ID
	if snapshotid == "latest" {
		id, err = restic.FindLatestSnapshot(ctx, repo, opts.Paths, opts.Tags, opts.Hosts, nil)
		if err != nil {
			clean.Cleanup()
			return fmt.Errorf("latest snapshot for criteria not found: %v Paths:%v Hosts:%v", err, opts.Paths, opts.Hosts)
		}
	} else {
		id, err = restic.FindSnapshot(ctx, repo, snapshotid)
		if err != nil {
			clean.Cleanup()
			return fmt.Errorf("invalid id %q: %v", snapshotid, err)
		}
	}
	// 获取数据总数
	t.Go(func() error {
		stats, err := getStatsForSnapshots(ctx, repo, id)
		if err != nil {
			return progressReporter.ScannerError(err)
		}
		s := ui.Stats{
			TotalSize:      stats.TotalSize,
			TotalFileCount: uint(stats.TotalFileCount),
		}
		progressReporter.ReportTotal("", s)
		return nil
	})

	res, err := restorer.NewRestorer(ctx, repo, id)
	if err != nil {
		clean.Cleanup()
		return fmt.Errorf("creating restorer failed: %v\n", err)
	}
	res.Error = func(location string, err error) error {
		return progressReporter.Error(location, err)
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

	selectByNameFilter := func(item string) bool {
		return true
	}

	selectFilter := func(item string, fi os.FileInfo) bool {
		return true
	}

	var targetFS fs.FS = fs.Local{}
	sc := archiver.NewScanner(targetFS)
	sc.SelectByName = selectByNameFilter
	sc.Select = selectFilter
	sc.Result = progressReporter.CompleteItem
	start := false
	t.Go(func() error {
		tic := time.NewTicker(time.Second)
		defer tic.Stop()
		for {
			select {
			case <-t.Context(ctx).Done():
				return nil
			case <-tic.C:
				if !start {
					continue
				}
			}
			_ = sc.Scan(ctx, res.Snapshot().Paths)
		}
	})

	server.Logger().Debugf("restoring %s to %s\n", res.Snapshot(), opts.Target)
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
				progress.UpdateTaskInfo(info)
			}
		}
	})
	go func() {
		defer clean.Cleanup()
		err = taskHistoryService.UpdateField(taskinfoid, "Status", task.StatusRunning, common.DBOptions{})
		if err != nil {
			server.Logger().Error(err)
		}
		start = true
		err = res.RestoreTo(ctx, opts.Target)
		if err != nil {
			server.Logger().Error(err)
			_ = progressReporter.Error("RestoreTo", err)
		}
		if opts.Verify {
			server.Logger().Debugf("verifying files in %s\n", opts.Target)
			t0 := time.Now()
			count, err := res.VerifyFiles(ctx, opts.Target)
			if err != nil {
				_ = progressReporter.Error("VerifyFiles", err)
			}
			server.Logger().Debugf("finished verifying %d files in %s (took %s)\n", count, opts.Target,
				time.Since(t0).Round(time.Millisecond))
		}
		t.Kill(nil)
		werr := t.Wait()
		if werr != nil {
			server.Logger().Error(werr)
		}
		progressReporter.Finish(snapshotid)
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

func getStatsForSnapshots(ctx context.Context, repo restic.Repository, id restic.ID) (*StatsContainer, error) {
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
	sn, err := restic.LoadSnapshot(ctx, repo, id)
	if err != nil {
		return nil, err
	}
	err = statsWalkSnapshot(opt, ctx, sn, repo, stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
