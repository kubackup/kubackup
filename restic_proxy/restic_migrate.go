package resticProxy

import (
	"context"
	"fmt"
	operationModel "github.com/kubackup/kubackup/internal/entity/v1/operation"
	repoModel "github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/store/log"
	wsTaskInfo "github.com/kubackup/kubackup/internal/store/ws_task_info"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/migrations"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"gopkg.in/tomb.v2"
)

// MigrateOptions bundles all options for the 'check' command.
type MigrateOptions struct {
	Force bool
}

func applyMigrations(repoid int, ctx context.Context, opts MigrateOptions, gopts GlobalOptions, repo restic.Repository, action string, spr *wsTaskInfo.Sprintf) error {
	var firsterr error
	for _, m := range migrations.All {
		if m.Name() == action {
			ok, reason, err := m.Check(ctx, repo)
			if err != nil {
				return err
			}

			if !ok {
				if !opts.Force {
					if reason == "" {
						reason = "check failed"
					}
					spr.Append(wsTaskInfo.Error, fmt.Sprintf("migration %v cannot be applied: %v\nIf you want to apply this migration anyway, re-run with option --force\n", m.Name(), reason))
					continue
				}

				spr.Append(wsTaskInfo.Error, fmt.Sprintf("check for migration %v failed, continuing anyway\n", m.Name()))
			}

			if m.RepoCheck() {
				spr.Append(wsTaskInfo.Info, "checking repository integrity...\n")

				checkOptions := CheckOptions{NoLock: true}
				checkGopts := gopts
				// the repository is already locked
				checkGopts.NoLock = true
				err = RunCheckSync(checkOptions, repoid)
				if err != nil {
					return err
				}
			}

			spr.Append(wsTaskInfo.Info, fmt.Sprintf("applying migration %v...\n", m.Name()))
			if err = m.Apply(ctx, repo); err != nil {
				spr.Append(wsTaskInfo.Error, fmt.Sprintf("migration %v failed: %v\n", m.Name(), err))
				if firsterr == nil {
					firsterr = err
				}
				continue
			}

			spr.Append(wsTaskInfo.Success, fmt.Sprintf("migration %v: success\n", m.Name()))
		}
	}

	return firsterr
}

func RunMigrate(opts MigrateOptions, repoid int, action string) (int, error) {
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
		Type:         operationModel.MIGRATE_TYPE,
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
		err := applyMigrations(repoid, ctx, opts, repoHandler.gopts, repo, action, spr)
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
		InitRepository()
		t.Kill(nil)
		log.LogInfos.Close(oper.Id, "process end", 1)
		return nil
	})
	return oper.Id, nil
}
