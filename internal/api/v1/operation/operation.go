package operation

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	operationDao "github.com/kubackup/kubackup/internal/service/v1/operation"
	"github.com/kubackup/kubackup/pkg/utils"
)

var operationService operationDao.Service

func init() {
	operationService = operationDao.GetService()
}

func getLastHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		Type, err := ctx.Params().GetInt("type")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		listLast, err := operationService.ListLast(repository, Type, common.DBOptions{})
		if err != nil && err.Error() != "not found" {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", listLast)
	}
}

func Install(parent iris.Party) {
	// 仓库相关接口
	sp := parent.Party("/operation")
	sp.Get("/last/:type/:repository", getLastHandler())
}
