package repository

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	repositoryDao "github.com/kubackup/kubackup/internal/service/v1/repository"
	"github.com/kubackup/kubackup/pkg/utils"
	resticProxy "github.com/kubackup/kubackup/restic_proxy"
	"strconv"
)

var repositoryService repositoryDao.Service

func init() {
	repositoryService = repositoryDao.GetService()
}

func createHandler() iris.Handler {
	return func(ctx *context.Context) {
		var rep repository.Repository
		err := ctx.ReadJSON(&rep)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if rep.Password == "" {
			utils.ErrorStr(ctx, "请输入密码")
			return
		}
		option, _ := resticProxy.GetGlobalOptions(rep)
		repo, err1 := resticProxy.OpenRepository(ctx, option)
		if err1 != nil {
			//仓库异常，重新初始化
			version, err := resticProxy.RunInit(ctx, option)
			if err != nil {
				utils.Errore(ctx, err)
				return
			}
			rep.RepositoryVersion = strconv.Itoa(int(version))
		}
		rep.RepositoryVersion = strconv.Itoa(int(repo.Config().Version))
		rep.Status = repository.StatusNone
		err = repositoryService.Create(&rep, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		go resticProxy.InitRepository()
		ctx.Values().Set("data", rep.Id)
	}
}

func listHandler() iris.Handler {
	return func(ctx *context.Context) {
		repotype, err := ctx.URLParamInt("type")
		if err != nil {
			repotype = 0
		}
		name := ctx.URLParam("name")
		ress, err := resticProxy.GetAllRepoWithStatus(repotype, name)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", ress)
	}
}

func delHanlder() iris.Handler {
	return func(ctx *context.Context) {
		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		err = repositoryService.Delete(id, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "")
	}
}
func updateHandler() iris.Handler {
	return func(ctx *context.Context) {
		var rep repository.Repository
		err := ctx.ReadJSON(&rep)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		// 仅以下字段允许更改
		rep2, err := repositoryService.Get(id, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if rep.Name != "" {
			rep2.Name = rep.Name
		}
		if rep.KeyId != "" {
			rep2.KeyId = rep.KeyId
		}
		if rep.Region != "" {
			rep2.Region = rep.Region
		}
		if rep.Bucket != "" {
			rep2.Bucket = rep.Bucket
		}
		if rep.Secret != "" {
			rep2.Secret = rep.Secret
		}
		if rep.Endpoint != "" {
			rep2.Endpoint = rep.Endpoint
		}
		rep2.PackSize = rep.PackSize
		option, _ := resticProxy.GetGlobalOptions(*rep2)
		_, err = resticProxy.OpenRepository(ctx, option)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		err = repositoryService.Update(rep2, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		go resticProxy.InitRepository()
		ctx.Values().Set("data", "")
	}
}

func getHandler() iris.Handler {
	return func(ctx *context.Context) {

		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		resp, err := repositoryService.Get(id, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		config := resticProxy.CheckRepoStatus(resp.Id)
		if config != nil {
			resp.Status = repository.StatusRun
			resp.RepositoryVersion = strconv.Itoa(int(config.Version))
		} else {
			resp.Status = repository.StatusErr
			resp.Errmsg = "仓库连接超时"
		}
		resp.Password = "******"
		ctx.Values().Set("data", resp)
	}
}

func Install(parent iris.Party) {
	// 仓库相关接口
	sp := parent.Party("/repository")
	// 新增
	sp.Post("", createHandler())
	// 列表
	sp.Get("", listHandler())
	// 删除
	sp.Delete("/:id", delHanlder())
	// 修改
	sp.Put("/:id", updateHandler())

	sp.Get("/:id", getHandler())
}
