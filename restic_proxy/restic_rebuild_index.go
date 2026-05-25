package resticProxy

import (
	"context"
	"fmt"
	"time"

	operationModel "github.com/kubackup/kubackup/internal/entity/v1/operation"
	repoModel "github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/log"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/ui/progress"
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

	ctx, cancel := context.WithCancel(context.Background())
	clean := NewCleanCtx()
	clean.AddCleanCtx(func() {
		cancel()
	})

	unlocker, _, err := lockRepoExclusive(ctx, repo)
	if err != nil {
		clean.Cleanup()
		return 0, err
	}
	clean.AddCleanCtx(func() {
		unlockRepo(unlocker)
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
	repairOpts := repository.RepairIndexOptions{
		ReadAllPacks: opts.ReadAllPacks,
	}

	err := repository.RepairIndex(ctx, repo, repairOpts, &progressPrinter{spr: spr})
	if err != nil {
		return err
	}

	spr.Append(wsTaskInfo.Success, fmt.Sprintf("done\n"))
	return nil
}

type progressPrinter struct {
	spr *wsTaskInfo.Sprintf
}

func (p *progressPrinter) P(format string, args ...interface{}) {
	p.spr.Append(wsTaskInfo.Info, fmt.Sprintf(format, args...))
}

func (p *progressPrinter) E(format string, args ...interface{}) {
	p.spr.Append(wsTaskInfo.Error, fmt.Sprintf(format, args...))
}

func (p *progressPrinter) S(format string, args ...interface{}) {
	p.spr.Append(wsTaskInfo.Info, fmt.Sprintf(format, args...))
}

func (p *progressPrinter) V(format string, args ...interface{}) {
	p.spr.Append(wsTaskInfo.Info, fmt.Sprintf(format, args...))
}

func (p *progressPrinter) VV(format string, args ...interface{}) {
	p.spr.Append(wsTaskInfo.Info, fmt.Sprintf(format, args...))
}

func (p *progressPrinter) NewCounter(name string) *progress.Counter {
	return progress.NewCounter(time.Second, 0, func(v, max uint64, runtime time.Duration, final bool) {
		if max > 0 {
			p.spr.Append(wsTaskInfo.Info, fmt.Sprintf("%s: %d/%d\r", name, v, max))
		} else {
			p.spr.Append(wsTaskInfo.Info, fmt.Sprintf("%s: %d\r", name, v))
		}
	})
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
