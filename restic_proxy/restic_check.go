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
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui"
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
	NoLock         bool
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
//   - if --with-cache is specified, the default cache is used
//   - if the user explicitly requested --no-cache, we don't use any cache
//   - if the user provides --cache-dir, we use a cache in a temporary sub-directory of the specified directory and the sub-directory is deleted after the check
//   - by default, we use a cache in a temporary directory that is deleted after the check
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

func RunCheckSync(ctx context.Context, opts CheckOptions, gopts GlobalOptions, repo *repository.Repository, spr *wsTaskInfo.Sprintf) error {
	err := checkFlags(opts)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})
	if !opts.NoLock {
		lock, err := lockRepoExclusive(ctx, repo)
		if err != nil {
			clean.Cleanup()
			return err
		}
		clean.AddCleanCtx(func() {
			unlockRepo(lock)
		})
	}
	if spr == nil {
		logTask := log.LogInfo{}
		logTask.SetId(0)
		spr = wsTaskInfo.NewSprintf(&logTask)
	}
	err = check(repo, opts, gopts, ctx, spr)
	clean.Cleanup()
	if err != nil {
		return err
	}
	return nil
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
	ctx, cancel := context.WithCancel(context.Background())
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})
	repo := repoHandler.repo
	if !opts.NoLock {
		lock, err := lockRepoExclusive(ctx, repo)
		if err != nil {
			clean.Cleanup()
			return 0, err
		}
		clean.AddCleanCtx(func() {
			unlockRepo(lock)
		})
	}
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

	logTask.SetBound(make(chan string))
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
			spr.Append(wsTaskInfo.Error, err.Error())
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
	err := chkr.LoadSnapshots(ctx)
	if err != nil {
		return err
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("load indexes\n"))
	pro := newProgressMax(true, 0, "files deleted", spr)
	hints, errs := chkr.LoadIndex(ctx, pro)

	errorsFound := false
	suggestIndexRebuild := false
	mixedFound := false
	for _, hint := range hints {
		switch hint.(type) {
		case *checker.ErrDuplicatePacks, *checker.ErrOldIndexFormat:
			spr.AppendByForce(wsTaskInfo.Info, fmt.Sprintf("%v\n", hint), false)
			suggestIndexRebuild = true
		case *checker.ErrMixedPack:
			spr.AppendByForce(wsTaskInfo.Info, fmt.Sprintf("%v\n", hint), false)
			mixedFound = true
		default:
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("error: %v\n", hint))
			errorsFound = true
		}
	}

	if suggestIndexRebuild {
		spr.Append(wsTaskInfo.Info, fmt.Sprint("Duplicate packs/old indexes are non-critical, you can `rebuild index' to correct this."))
	}
	if mixedFound {
		spr.Append(wsTaskInfo.Info, fmt.Sprint("Mixed packs with tree and data blobs are non-critical, you can `prune` to correct this."))
	}

	if len(errs) > 0 {
		for _, err := range errs {
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("error: %v\n", err))
		}
		return errors.Fatal("LoadIndex returned errors")
	}

	orphanedPacks := 0
	errChan := make(chan error)

	spr.Append(wsTaskInfo.Info, fmt.Sprint("check all packs\n"))
	spr.ResetLimitNum()
	go chkr.Packs(ctx, errChan)

	for err := range errChan {
		if checker.IsOrphanedPack(err) {
			orphanedPacks++
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("%v\n", err))
		} else if err == checker.ErrLegacyLayout {
			spr.Append(wsTaskInfo.Error, fmt.Sprint("repository still uses the S3 legacy layout\nPlease run `restic migrate s3legacy` to correct this.\n"))
		} else {
			errorsFound = true
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("%v\n", err))
		}
	}

	if orphanedPacks > 0 {
		spr.Append(wsTaskInfo.Error, fmt.Sprintf("%d additional files were found in the repo, which likely contain duplicate data.\nThis is non-critical, you can `prune` to correct this.\n", orphanedPacks))
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprint("check snapshots, trees and blobs\n"))
	errChan = make(chan error)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		pro = newProgressMax(true, 0, "snapshots", spr)
		defer pro.Done()
		chkr.Structure(ctx, pro, errChan)
	}()

	for err := range errChan {
		errorsFound = true
		if e, ok := err.(*checker.TreeError); ok {
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("error for tree %v:\n", e.ID.Str()))
			for _, treeErr := range e.Errors {
				spr.Append(wsTaskInfo.Info, fmt.Sprintf("  %v\n", treeErr))
			}
		} else {
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("error: %v\n", err))
		}
	}

	// Wait for the progress bar to be complete before printing more below.
	// Must happen after `errChan` is read from in the above loop to avoid
	// deadlocking in the case of errors.
	wg.Wait()

	if opts.CheckUnused {
		for _, id := range chkr.UnusedBlobs(ctx) {
			spr.AppendByForce(wsTaskInfo.Info, fmt.Sprintf("unused blob %v\n", id), false)
			errorsFound = true
		}
	}

	doReadData := func(packs map[restic.ID]int64) {
		packCount := uint64(len(packs))

		p := newProgressMax(true, packCount, "packs", spr)
		errChan := make(chan error)

		go chkr.ReadPacks(ctx, packs, p, errChan)

		var salvagePacks restic.IDs

		for err := range errChan {
			errorsFound = true
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("%v\n", err))
			if err, ok := err.(*checker.ErrPackData); ok {
				if strings.Contains(err.Error(), "wrong data returned, hash is") {
					salvagePacks = append(salvagePacks, err.PackID)
				}
			}
		}
		p.Done()

		if len(salvagePacks) > 0 {
			spr.Append(wsTaskInfo.Error, fmt.Sprintf("\nThe repository contains pack files with damaged blobs. These blobs must be removed to repair the repository. This can be done using the following commands:\n\n"))
			var strIds []string
			for _, id := range salvagePacks {
				strIds = append(strIds, id.String())
			}
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("RESTIC_FEATURES=repair-packs-v1 restic repair packs %v\nrestic repair snapshots --forget\n\n", strings.Join(strIds, " ")))
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("Corrupted blobs are either caused by hardware problems or bugs in restic. Please open an issue at https://github.com/restic/restic/issues/new/choose for further troubleshooting!\n"))
		}
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
		} else if strings.HasSuffix(opts.ReadDataSubset, "%") {
			percentage, err := parsePercentage(opts.ReadDataSubset)
			if err == nil {
				packs = selectRandomPacksByPercentage(chkr.GetPacks(), percentage)
				spr.Append(wsTaskInfo.Info, fmt.Sprintf("read %.1f%% of data packs\n", percentage))
			}
		} else {
			repoSize := int64(0)
			allPacks := chkr.GetPacks()
			for _, size := range allPacks {
				repoSize += size
			}
			if repoSize == 0 {
				return errors.Fatal("Cannot read from a repository having size 0")
			}
			subsetSize, _ := ui.ParseBytes(opts.ReadDataSubset)
			if subsetSize > repoSize {
				subsetSize = repoSize
			}
			packs = selectRandomPacksByFileSize(chkr.GetPacks(), subsetSize, repoSize)
			spr.Append(wsTaskInfo.Info, fmt.Sprintf("read %d bytes of data packs\n", subsetSize))
		}
		if packs == nil {
			return errors.Fatal("internal error: failed to select packs to check")
		}
		doReadData(packs)
	}

	if errorsFound {
		return errors.Fatal("repository contains errors")
	}
	spr.Append(wsTaskInfo.Info, fmt.Sprintf("no errors were found\n"))

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
func CheckRepoStatus(repoid int) *restic.Config {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return nil
	}
	repo := repoHandler.repo
	conf, err := restic.LoadConfig(context.Background(), repo)
	if err != nil {
		return nil
	}
	return &conf
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
				config := CheckRepoStatus(v.Id)
				if config != nil {
					v.Status = repoModel.StatusRun
					v.RepositoryVersion = strconv.Itoa(int(config.Version))
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

func selectRandomPacksByFileSize(allPacks map[restic.ID]int64, subsetSize int64, repoSize int64) map[restic.ID]int64 {
	subsetPercentage := (float64(subsetSize) / float64(repoSize)) * 100.0
	packs := selectRandomPacksByPercentage(allPacks, subsetPercentage)
	return packs
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
