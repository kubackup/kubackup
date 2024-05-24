package system

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	fileutil "github.com/kubackup/kubackup/pkg/file"
)

func lsHandler() iris.Handler {
	return func(ctx *context.Context) {
		path := ctx.URLParam("path")
		listDir, err := fileutil.ListDir(path)
		if err != nil {
			return
		}
		ctx.Values().Set("data", listDir)
	}
}

func Install(parent iris.Party) {
	// 服务器 操作接口
	sp := parent.Party("/system")
	// 列出文件夹
	sp.Get("/ls", lsHandler())
}
