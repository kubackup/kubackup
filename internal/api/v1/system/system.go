package system

import (
	"encoding/json"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup"
	"github.com/kubackup/kubackup/internal/consts/global"
	sys "github.com/kubackup/kubackup/internal/entity/v1/system"
	"github.com/kubackup/kubackup/internal/service/v1/system"
	fileutil "github.com/kubackup/kubackup/pkg/file"
	"github.com/kubackup/kubackup/pkg/utils"
	"github.com/kubackup/kubackup/pkg/utils/http"
)

func lsHandler() iris.Handler {
	return func(ctx *context.Context) {
		path := ctx.URLParam("path")
		listDir, err := fileutil.ListDir(path)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", listDir)
	}
}

func upgradeVersionHandler() iris.Handler {
	return func(ctx *context.Context) {
		version := ctx.Params().GetString("version")
		err := system.Upgrade(version)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "upgrading")
	}
}

func versionHandler() iris.Handler {
	return func(ctx *context.Context) {
		v := kubackup.GetVersion()
		ctx.Values().Set("data", v)
	}
}

func latestVersionHandler() iris.Handler {
	return func(ctx *context.Context) {
		body, err := http.Get(global.LatestUrl)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		var releases []sys.Release
		err = json.Unmarshal([]byte(body), &releases)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if len(releases) == 0 {
			utils.ErrorStr(ctx, "新版本获取失败")
		}
		latest := releases[0].TagName
		ctx.Values().Set("data", latest)
	}
}

func Install(parent iris.Party) {
	// 系统接口
	sp := parent.Party("/system")
	// 列出文件夹
	sp.Get("/ls", lsHandler())

	sp.Post("/upgradeVersion/:version", upgradeVersionHandler())

	sp.Get("/version", versionHandler())

	sp.Get("/version/latest", latestVersionHandler())
}
