package resticProxy

import (
	"context"
	"fmt"
	operationModel "github.com/kubackup/kubackup/internal/entity/v1/operation"
	repoModel "github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/log"
	"github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/index"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/pack"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"gopkg.in/tomb.v2"
)

type RebuildIndexOptions struct {
	ReadAllPacks bool //read all pack files to generate new index from scratch
}

func RunRebuildIndex(opts RebuildIndexOptions, repoid int) (int, error) {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return 0, err
	}
	repo := repoHandler.repo

	ctx, cancel := context.WithCancel(repoHandler.gopts.ctx)
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})

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
		Type:         operationModel.REBUILDINDEX_TYPE,
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
		err := rebuildIndex(opts, ctx, repo, spr)
		status = repoModel.StatusNone
		if err != nil {
			spr.Append(wsTaskInfo.Error, err.Error())
			status = repoModel.StatusErr
		} else {
			status = repoModel.StatusRun
		}
		err = repo.LoadIndex(ctx, nil)
		if err != nil {
			spr.Append(wsTaskInfo.Error, err.Error())
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

func rebuildIndex(opts RebuildIndexOptions, ctx context.Context, repo *repository.Repository, spr *wsTaskInfo.Sprintf) error {

	var obsoleteIndexes restic.IDs
	packSizeFromList := make(map[restic.ID]int64)
	packSizeFromIndex := make(map[restic.ID]int64)
	removePacks := restic.NewIDSet()

	if opts.ReadAllPacks {
		// get list of old index files but start with empty index
		err := repo.List(ctx, restic.IndexFile, func(id restic.ID, size int64) error {
			obsoleteIndexes = append(obsoleteIndexes, id)
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("loading indexes...\n"))
		mi := index.NewMasterIndex()
		err := index.ForAllIndexes(ctx, repo.Backend(), repo, func(id restic.ID, idx *index.Index, oldFormat bool, err error) error {
			if err != nil {
				server.Logger().Warnf("removing invalid index %v: %v\n", id, err)
				obsoleteIndexes = append(obsoleteIndexes, id)
				return nil
			}

			mi.Insert(idx)
			return nil
		})
		if err != nil {
			return err
		}

		err = mi.MergeFinalIndexes()
		if err != nil {
			return err
		}

		err = repo.SetIndex(mi)
		if err != nil {
			return err
		}
		packSizeFromIndex = pack.Size(ctx, repo.Index(), false)
	}

	spr.Append(wsTaskInfo.Info, fmt.Sprintf("getting pack files to read...\n"))
	spr.ResetLimitNum()
	err := repo.List(ctx, restic.PackFile, func(id restic.ID, packSize int64) error {
		size, ok := packSizeFromIndex[id]
		if !ok || size != packSize {
			// Pack was not referenced in index or size does not match
			packSizeFromList[id] = packSize
			removePacks.Insert(id)
		}
		if !ok {
			spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("adding pack file to index %v\n", id))
		} else if size != packSize {
			spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("reindexing pack file %v with unexpected size %v instead of %v\n", id, packSize, size))
		}
		delete(packSizeFromIndex, id)
		return nil
	})
	if err != nil {
		return err
	}
	spr.ResetLimitNum()
	for id := range packSizeFromIndex {
		// forget pack files that are referenced in the index but do not exist
		// when rebuilding the index
		removePacks.Insert(id)
		spr.AppendLimit(wsTaskInfo.Warning, fmt.Sprintf("removing not found pack file %v\n", id))
	}

	if len(packSizeFromList) > 0 {
		spr.Append(wsTaskInfo.Info, fmt.Sprintf("reading pack files\n"))
		max := uint64(len(packSizeFromList))
		pro := newProgressMax(true, max, "packs", spr)
		invalidFiles, err := repo.CreateIndexFromPacks(ctx, packSizeFromList, pro)
		pro.Done()
		if err != nil {
			return err
		}
		spr.ResetLimitNum()
		for _, id := range invalidFiles {
			spr.AppendLimit(wsTaskInfo.Info, fmt.Sprintf("skipped incomplete pack file: %v\n", id))
		}
	}

	err = rebuildIndexFiles(ctx, repo, removePacks, obsoleteIndexes, spr)
	if err != nil {
		return err
	}
	spr.Append(wsTaskInfo.Success, fmt.Sprintf("done\n"))
	return nil
}

func AutoRebuildIndex() {
	opt := RebuildIndexOptions{}
	for _, repo := range Myrepositorys.rep {
		repoid := repo
		listLast, err := operationService.ListLast(repoid.repoId, operationModel.CHECK_TYPE, common.DBOptions{})
		if err != nil {
			continue
		}
		if listLast.Status == repoModel.StatusErr {
			go func() {
				_, _ = RunRebuildIndex(opt, repoid.repoId)
			}()
		}
	}

}
