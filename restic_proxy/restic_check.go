package resticProxy

import (
	"context"
	"fmt"
	operationModel "github.com/kubackup/kubackup/internal/entity/v1/operation"
	repoModel "github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	operationDao "github.com/kubackup/kubackup/internal/service/v1/operation"
	"github.com/kubackup/kubackup/internal/store/log"
	"github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/checker"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"gopkg.in/tomb.v2"
	"io/ioutil"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var operationService operationDao.Service

func init() {
	operationService = operationDao.GetService()
}

type CheckOptions struct {
	ReadData       bool   //read all data blobs
	ReadDataSubset string //read a `subset` of data packs, specified as 'n/t' for specific subset or either 'x%' or 'x.y%' for random subset
	CheckUnused    bool   //find unused blobs
	WithCache      bool   //use the cache
}

func checkFlags(opts CheckOptions) error {
	if opts.ReadData && opts.ReadDataSubset != "" {
		return errors.Fatal("check flags --read-data and --read-data-subset cannot be used together")
	}
	if opts.ReadDataSubset != "" {
		dataSubset, err := stringToIntSlice(opts.ReadDataSubset)
		argumentError := errors.Fatal("check flag --read-data-subset must have two positive integer values or a percentage, e.g. --read-data-subset=1/2 or --read-data-subset=2.5%%")
		if err == nil {
			if len(dataSubset) != 2 {
				return argumentError
			}
			if dataSubset[0] == 0 || dataSubset[1] == 0 || dataSubset[0] > dataSubset[1] {
				return errors.Fatal("check flag --read-data-subset=n/t values must be positive integers, and n <= t, e.g. --read-data-subset=1/2")
			}
			if dataSubset[1] > totalBucketsMax {
				return errors.Fatalf("check flag --read-data-subset=n/t t must be at most %d", totalBucketsMax)
			}
		} else {
			percentage, err := parsePercentage(opts.ReadDataSubset)
			if err != nil {
				return argumentError
			}

			if percentage <= 0.0 || percentage > 100.0 {
				return errors.Fatal(
					"check flag --read-data-subset=n% n must be above 0.0% and at most 100.0%")
			}
		}
	}

	return nil
}

// See doReadData in runCheck below for why this is 256.
const totalBucketsMax = 256

// stringToIntSlice converts string to []uint, using '/' as element separator
func stringToIntSlice(param string) (split []uint, err error) {
	if param == "" {
		return nil, nil
	}
	parts := strings.Split(param, "/")
	result := make([]uint, len(parts))
	for idx, part := range parts {
		uintval, err := strconv.ParseUint(part, 10, 0)
		if err != nil {
			return nil, err
		}
		result[idx] = uint(uintval)
	}
	return result, nil
}

// ParsePercentage parses a percentage string of the form "X%" where X is a float constant,
// and returns the value of that constant. It does not check the range of the value.
func parsePercentage(s string) (float64, error) {
	if !strings.HasSuffix(s, "%") {
		return 0, errors.Errorf(`parsePercentage: %q does not end in "%%"`, s)
	}
	s = s[:len(s)-1]

	p, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, errors.Errorf("parsePercentage: %v", err)
	}
	return p, nil
}

// prepareCheckCache configures a special cache directory for check.
//
//  * if --with-cache is specified, the default cache is used
//  * if the user explicitly requested --no-cache, we don't use any cache
//  * if the user provides --cache-dir, we use a cache in a temporary sub-directory of the specified directory and the sub-directory is deleted after the check
//  * by default, we use a cache in a temporary directory that is deleted after the check
func prepareCheckCache(opts CheckOptions, gopts GlobalOptions, spr *wsTaskInfo.Sprintf) (cleanup func()) {
	cleanup = func() {}
	if opts.WithCache {
		// use the default cache, no setup needed
		return cleanup
	}

	if gopts.NoCache {
		// don't use any cache, no setup needed
		return cleanup
	}

	cachedir := gopts.CacheDir

	// use a cache in a temporary directory
	tempdir, err := ioutil.TempDir(cachedir, "backup-check-cache-")
	if err != nil {
		// if an error occurs, don't use any cache
		spr.Append(wsTaskInfo.Error, fmt.Sprintf("unable to create temporary directory for cache during check, disabling cache: %v\n", err))
		gopts.NoCache = true
		return cleanup
	}

	gopts.CacheDir = tempdir
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("using temporary cache in %v\n", tempdir))

	cleanup = func() {
		err := fs.RemoveAll(tempdir)
		if err != nil {
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("error：error removing temporary cache directory: %v\n", err))
		}
	}

	return cleanup
}

func RunCheck(opts CheckOptions, repoid int) (int, error) {

	err := checkFlags(opts)
	if err != nil {
		return 0, err
	}
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return 0, err
	}
	gopts := repoHandler.gopts
	ctx, cancel := context.WithCancel(gopts.ctx)
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})
	repo := repoHandler.repo
	lock, err := lockRepoExclusive(ctx, repo)
	if err != nil {
		clean.Cleanup()
		return 0, err
	}
	clean.AddCleanCtx(func() {
		unlockRepo(lock)
	})
	status := repoModel.StatusNone
	oper := operationModel.Operation{
		RepositoryId: repoid,
		Type:         operationModel.CHECK_TYPE,
		Status:       status,
		Logs:         make([]*wsTaskInfo.Sprint, 0),
	}
	err = operationService.Create(&oper, common.DBOptions{})
	if err != nil {
		clean.Cleanup()
		return 0, err
	}
	var t tomb.Tomb
	logTask := log.LogInfo{}
	logTask.SetId(oper.Id)
	spr := wsTaskInfo.NewSprintf(&logTask)

	logTask.SetBound(make(chan error))
	log.LogInfos.Set(oper.Id, &logTask)
	t.Go(func() error {
		for {
			select {
			case <-t.Context(ctx).Done():
				return nil
			case <-log.LogInfos.Get(oper.Id).GetBound():
				info := log.LogInfos.Get(oper.Id)
				spr.UpdateTaskInfo(info)
				spr.SendAllLog()
			}
		}
	})
	t.Go(func() error {
		defer clean.Cleanup()
		err := check(repo, opts, gopts, ctx, spr)
		status = repoModel.StatusNone
		if err != nil {
			status = repoModel.StatusErr
		} else {
			status = repoModel.StatusRun
		}
		oper.Status = status
		oper.Logs = spr.Sprints
		err = operationService.Update(&oper, common.DBOptions{})
		if err != nil {
			server.Logger().Error(err)
		}
		t.Kill(nil)
		log.LogInfos.Close(oper.Id, "process end", 1)
		return nil
	})
	return oper.Id, nil
}

func check(repo *repository.Repository, opts CheckOptions, gopts GlobalOptions, ctx context.Context, spr *wsTaskInfo.Sprintf) error {
	cleanup := prepareCheckCache(opts, gopts, spr)
	defer cleanup()

	chkr := checker.New(repo, opts.CheckUnused)

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("load indexes\n"))
	hints, errs := chkr.LoadIndex(ctx)

	dupFound := false
	for _, hint := range hints {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("%v\n", hint))
		if _, ok := hint.(checker.ErrDuplicatePacks); ok {
			dupFound = true
		}
	}

	if dupFound {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("This is non-critical,will run 'rebuildIndex' to correct this\n"))
		_ = rebuildIndex(RebuildIndexOptions{}, ctx, repo, spr)
		err := LoadIndex(ctx, repo)
		if err != nil {
			spr.Append(wsTaskInfo.Error, err.Error())
		}
	}
	if len(errs) > 0 {
		for _, err := range errs {
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("error: %v\n", err))
		}
		return errors.Fatal("LoadIndex returned errors")
	}

	errorsFound := false
	orphanedPacks := 0
	errChan := make(chan error)

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("check all packs\n"))
	go chkr.Packs(ctx, errChan)
	spr.ResetLimitNum()
	for err := range errChan {
		if checker.IsOrphanedPack(err) {
			orphanedPacks++
			spr.AppendLimit(wsTaskInfo.Error, fmt.Sprintf("%v\n", err))
			continue
		}
		errorsFound = true
		spr.AppendLimit(wsTaskInfo.Error, fmt.Sprintf("%v\n", err))
	}

	if orphanedPacks > 0 {
		spr.Append(wsTaskInfo.Error, fmt.Sprintf("%d additional files were found in the repo, which likely contain duplicate data.\nYou can run `restic prune` to correct this.\n", orphanedPacks))
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("check snapshots, trees and blobs\n"))
	errChan = make(chan error)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		pro := newProgressMax(true, 0, "snapshots", spr)
		defer pro.Done()
		chkr.Structure(ctx, pro, errChan)
	}()
	spr.ResetLimitNum()
	for err := range errChan {
		errorsFound = true
		if e, ok := err.(checker.TreeError); ok {
			spr.AppendLimit(wsTaskInfo.Error, fmt.Sprintf("error for tree %v:\n", e.ID.Str()))
			for _, treeErr := range e.Errors {
				spr.AppendLimit(wsTaskInfo.Error, fmt.Sprintf("%v\n", treeErr))
			}
		} else {
			spr.AppendLimit(wsTaskInfo.Error, fmt.Sprintf("%v\n", err))
		}
	}
	// Wait for the progress bar to be complete before printing more below.
	// Must happen after `errChan` is read from in the above loop to avoid
	// deadlocking in the case of errors.
	wg.Wait()
	if opts.CheckUnused {
		for _, id := range chkr.UnusedBlobs(ctx) {
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("unused blob %v\n", id))
			errorsFound = true
		}
	}

	doReadData := func(packs map[restic.ID]int64) {
		packCount := uint64(len(packs))
		pro := newProgressMax(true, packCount, "packs", spr)
		errChan := make(chan error)
		go chkr.ReadPacks(ctx, packs, pro, errChan)
		spr.ResetLimitNum()
		for err := range errChan {
			errorsFound = true
			spr.AppendLimit(wsTaskInfo.Error, fmt.Sprintf("%v\n", err))
		}
		pro.Done()
	}

	switch {
	case opts.ReadData:
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("read all data\n"))
		doReadData(selectPacksByBucket(chkr.GetPacks(), 1, 1))
	case opts.ReadDataSubset != "":
		var packs map[restic.ID]int64
		dataSubset, err := stringToIntSlice(opts.ReadDataSubset)
		if err == nil {
			bucket := dataSubset[0]
			totalBuckets := dataSubset[1]
			packs = selectPacksByBucket(chkr.GetPacks(), bucket, totalBuckets)
			packCount := uint64(len(packs))
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("read group #%d of %d data packs (out of total %d packs in %d groups)\n", bucket, packCount, chkr.CountPacks(), totalBuckets))
		} else {
			percentage, _ := parsePercentage(opts.ReadDataSubset)
			packs = selectRandomPacksByPercentage(chkr.GetPacks(), percentage)
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("read %.1f%% of data packs\n", percentage))
		}
		if packs == nil {
			return errors.Fatal("internal error: failed to select packs to check")
		}
		doReadData(packs)
	}

	if errorsFound {
		spr.Append(wsTaskInfo.Error, fmt.Sprint("repository contains errors"))
		return errors.Fatal("repository contains errors")
	}

	spr.Append(wsTaskInfo.Success, fmt.Sprintf("no errors were found\n"))
	return nil
}

// selectPacksByBucket selects subsets of packs by ranges of buckets.
func selectPacksByBucket(allPacks map[restic.ID]int64, bucket, totalBuckets uint) map[restic.ID]int64 {
	packs := make(map[restic.ID]int64)
	for pack, size := range allPacks {
		// If we ever check more than the first byte
		// of pack, update totalBucketsMax.
		if (uint(pack[0]) % totalBuckets) == (bucket - 1) {
			packs[pack] = size
		}
	}
	return packs
}

// selectRandomPacksByPercentage selects the given percentage of packs which are randomly choosen.
func selectRandomPacksByPercentage(allPacks map[restic.ID]int64, percentage float64) map[restic.ID]int64 {
	packCount := len(allPacks)
	packsToCheck := int(float64(packCount) * (percentage / 100.0))
	if packCount > 0 && packsToCheck < 1 {
		packsToCheck = 1
	}
	timeNs := time.Now().UnixNano()
	r := rand.New(rand.NewSource(timeNs))
	idx := r.Perm(packCount)

	var keys []restic.ID
	for k := range allPacks {
		keys = append(keys, k)
	}

	packs := make(map[restic.ID]int64)

	for i := 0; i < packsToCheck; i++ {
		id := keys[idx[i]]
		packs[id] = allPacks[id]
	}

	return packs
}

// CheckRepoStatus 检测仓库状态
func CheckRepoStatus(repoid int) bool {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return false
	}
	gopts := repoHandler.gopts
	repo := repoHandler.repo
	_, err = restic.LoadConfig(gopts.ctx, repo)
	if err != nil {
		return false
	}
	return true
}

// GetAllRepoWithStatus 获取仓库列表并带状态信息
func GetAllRepoWithStatus(repotype int, name string) ([]repoModel.Repository, error) {
	reps, err := repositoryService.List(repotype, name, common.DBOptions{})
	if err != nil && err.Error() != "not found" {
		return nil, err
	}
	ctxx := context.Background()
	var t tomb.Tomb
	ch := make(chan repoModel.Repository)
	defer close(ch)
	ress := make([]repoModel.Repository, 0)
	var lock sync.Mutex
	for i := 0; i < 4; i++ {
		t.Go(func() error {
			for {
				var v repoModel.Repository
				select {
				case <-t.Context(ctxx).Done():
					return nil
				case v = <-ch:
				}
				res := CheckRepoStatus(v.Id)
				if res {
					v.Status = repoModel.StatusRun
				} else {
					v.Status = repoModel.StatusErr
					v.Errmsg = "仓库连接超时"
				}
				v.Password = "******"
				v.Secret = ""
				v.KeyId = ""
				lock.Lock()
				ress = append(ress, v)
				lock.Unlock()
			}
		})
	}
	for _, v := range reps {
		ch <- v
	}
	t.Kill(nil)
	_ = t.Wait()
	sort.SliceStable(ress, func(i, j int) bool {
		return ress[i].Id > ress[j].Id
	})
	return ress, nil
}

func AutoCheck() {
	opt := CheckOptions{}
	for _, repo := range Myrepositorys.rep {
		repoid := repo
		go func() {
			_, _ = RunCheck(opt, repoid.repoId)
		}()
	}
}
