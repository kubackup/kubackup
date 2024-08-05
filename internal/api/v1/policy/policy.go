package policy

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	policyDao "github.com/kubackup/kubackup/internal/service/v1/policy"
	repositoryDao "github.com/kubackup/kubackup/internal/service/v1/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/utils"
	resticProxy "github.com/kubackup/kubackup/restic_proxy"
)

var policyService policyDao.Service
var repositoryService repositoryDao.Service

func init() {
	policyService = policyDao.GetService()
	repositoryService = repositoryDao.GetService()
}

func createHandler() iris.Handler {
	return func(ctx *context.Context) {
		var policy repository.ForgetPolicy
		err := ctx.ReadJSON(&policy)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if policy.RepositoryId <= 0 {
			utils.ErrorStr(ctx, "仓库id不能为空")
			return
		}
		if policy.Path == "" {
			utils.ErrorStr(ctx, "path不能为空")
			return
		}
		err = policyService.Create(&policy, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", policy.Id)
	}
}

func updateHandler() iris.Handler {
	return func(ctx *context.Context) {
		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		var policy repository.ForgetPolicy
		err = ctx.ReadJSON(&policy)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if policy.RepositoryId <= 0 {
			utils.ErrorStr(ctx, "仓库id不能为空")
			return
		}
		forgetPolicy, err := policyService.Get(id, common.DBOptions{})
		if err != nil {
			return
		}
		forgetPolicy.Value = policy.Value
		forgetPolicy.Type = policy.Type
		forgetPolicy.RepositoryId = policy.RepositoryId
		err = policyService.Update(forgetPolicy, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", policy.Id)
	}
}

func listHandler() iris.Handler {
	return func(ctx *context.Context) {
		repoid, err := ctx.URLParamInt("repository")
		if err != nil {
			repoid = 0
		}
		path := ctx.URLParam("path")
		policies, err := policyService.Search(repoid, path, common.DBOptions{})
		if err != nil && err.Error() != "not found" {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", policies)
	}
}

func delHanlder() iris.Handler {
	return func(ctx *context.Context) {
		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		err = policyService.Delete(id, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "")
	}
}

func doHanlder() iris.Handler {
	return func(ctx *context.Context) {
		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		policy, err := policyService.Get(id, common.DBOptions{})
		if err != nil {
			return
		}
		opt := resticProxy.ForgetOptions{
			Prune:          true,
			SnapshotFilter: restic.SnapshotFilter{Paths: []string{policy.Path}},
		}
		setType(policy.Type, policy.Value, &opt)
		operid, err := resticProxy.RunForget(opt, policy.RepositoryId, []string{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", operid)
	}
}

func Install(parent iris.Party) {
	// 仓库相关接口
	sp := parent.Party("/policy")
	// 新增
	sp.Post("", createHandler())
	// 修改
	sp.Put("/:id", updateHandler())
	// 列表
	sp.Get("", listHandler())
	// 删除
	sp.Delete("/:id", delHanlder())
	// 立即执行
	sp.Post("/do/:id", doHanlder())
}

func DoPolicy() {
	reps, err := repositoryService.List(0, "", common.DBOptions{})
	if err != nil {
		return
	}
	for _, rep := range reps {
		policys, err := policyService.Search(rep.Id, "", common.DBOptions{})
		if err != nil {
			return
		}
		for i, policy := range policys {
			opt := resticProxy.ForgetOptions{
				Prune:          i == (len(policys) - 1),
				SnapshotFilter: restic.SnapshotFilter{Paths: []string{policy.Path}},
			}
			setType(policy.Type, policy.Value, &opt)
			err = resticProxy.RunForgetSync(opt, policy.RepositoryId, []string{})
			if err != nil {
				server.Logger().Error(err)
			}
			server.Logger().Infof("清理 %s下%s， 保留最新 %d %s 快照", rep.Name, policy.Path, policy.Value, policy.Type)
		}
	}
}

func setType(t string, value int, opt *resticProxy.ForgetOptions) {
	v := resticProxy.ForgetPolicyCount(value)
	switch t {
	case "last":
		opt.Last = v
		break
	case "hourly":
		opt.Hourly = v
		break
	case "daily":
		opt.Daily = v
		break
	case "weekly":
		opt.Weekly = v
		break
	case "monthly":
		opt.Monthly = v
		break
	case "yearly":
		opt.Yearly = v
		break
	}
}
