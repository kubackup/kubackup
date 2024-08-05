package restic

import (
	"fmt"
	"github.com/fanjindong/go-cache"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup/internal/consts"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"github.com/kubackup/kubackup/pkg/utils"
	"github.com/kubackup/kubackup/restic_proxy"
	"os"
	"strconv"
	"strings"
	"time"
)

func lsHandler() iris.Handler {
	return func(ctx *context.Context) {
		snapshotid := ctx.Params().Get("snapshotid")
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		res := model.PageParam(ctx)
		path := ctx.URLParam("path")
		var lsResCache interface{}
		c := server.Cache()
		key := consts.Key("lsHandler", strconv.Itoa(repository), snapshotid, path)
		lsResCache, is := c.Get(key)
		var lsRes *resticProxy.LsRes
		if !is {
			lsRes, err = resticProxy.RunLs(path, repository, snapshotid)
			if err != nil {
				utils.Errore(ctx, err)
				return
			}
			if len(lsRes.Nodes) > 0 {
				c.Set(key, *lsRes, cache.WithEx(10*time.Minute))
			}
		} else {
			lsRes2, ok := lsResCache.(resticProxy.LsRes)
			if !ok {
				utils.ErrorStr(ctx, "缓存读取失败")
				return
			}
			lsRes = &lsRes2
		}
		total, result, err := model.PageFilter(res.PageNum, res.PageSize, lsRes.Nodes)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		res.Total = total
		lsRes.Nodes = result
		res.Items = lsRes
		ctx.Values().Set("data", res)
	}
}

func searchHandler() iris.Handler {
	return func(ctx *context.Context) {
		snapshotid := ctx.Params().Get("snapshotid")
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		res := model.PageParam(ctx)
		path := ctx.URLParam("path")
		lsRes, err := resticProxy.RunFind(path, repository, snapshotid)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		total, result, err := model.PageFilter(res.PageNum, res.PageSize, lsRes.Nodes)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		res.Total = total
		lsRes.Nodes = result
		res.Items = lsRes
		ctx.Values().Set("data", res)
	}
}

func snapshotsHandler() iris.Handler {
	return func(ctx *context.Context) {
		snapshotid := ctx.URLParam("snapshotid")
		groupby := ctx.URLParam("groupby")
		path := ctx.URLParam("path")
		host := ctx.URLParam("host")
		tag := ctx.URLParam("tag")
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		res := model.PageParam(ctx)
		var paths []string
		if path != "" {
			paths = strings.Split(path, ",")
		}
		var hosts []string
		if host != "" {
			hosts = strings.Split(host, ",")
		}
		tags := restic.TagLists{}
		if tag != "" {
			err := tags.Set(tag)
			if err != nil {
				utils.Errore(ctx, err)
				return
			}
		}
		groupBy, err := SplitSnapshotGroupBy(groupby)
		opts := resticProxy.SnapshotOptions{
			SnapshotFilter: restic.SnapshotFilter{Hosts: hosts, Paths: paths, Tags: tags},
			Compact:        false,
			Last:           false,
			Latest:         0,
			GroupBy:        groupBy,
		}
		var snapshotids []string
		if snapshotid != "" {
			snapshotids = strings.Split(snapshotid, ",")
		}
		snapshots, err := resticProxy.RunSnapshots(opts, repository, snapshotids)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		total, p, err := model.PageFilter(res.PageNum, res.PageSize, snapshots)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}

		res.Total = total
		res.Items = p
		ctx.Values().Set("data", res)
	}
}

func parmsHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		host := ctx.URLParam("host")

		var hosts []string
		if host != "" {
			hosts = strings.Split(host, ",")
		}
		paths, err := resticProxy.GetParms(repository, hosts)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", paths)
	}
}

func parmsMyHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		hostname, err := os.Hostname()
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		hosts := []string{hostname}
		paths, err := resticProxy.GetParms(repository, hosts)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ps := make([]string, 0)
		parms := paths.Parms
		for _, p := range parms {
			ps = append(ps, p.Paths...)
		}
		res := struct {
			Paths    []string `json:"paths"`
			Hostname string   `json:"hostname"`
		}{}
		res.Paths = ps
		res.Hostname = hostname
		ctx.Values().Set("data", res)
	}
}

func loadIndexHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		err = resticProxy.RunLoadIndex(repository)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "")
	}

}

func checkHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		opt := resticProxy.CheckOptions{}
		id, err := resticProxy.RunCheck(opt, repository)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", id)
	}
}

func rebuildIndexHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		opt := resticProxy.RebuildIndexOptions{
			ReadAllPacks: false,
		}
		id, err := resticProxy.RunRebuildIndex(opt, repository)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", id)
	}
}

func pruneHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		opt := resticProxy.PruneOptions{
			MaxUnused: "5%",
		}
		id, err := resticProxy.RunPrune(opt, repository)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", id)
	}
}
func migrateHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		force := ctx.Params().GetBoolDefault("force", false)
		action := ctx.Params().GetStringDefault("action", "upgrade_repo_v2")
		opt := resticProxy.MigrateOptions{
			Force: force,
		}
		id, err := resticProxy.RunMigrate(opt, repository, action)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", id)
	}
}

func unlockHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		all := ctx.URLParam("all") == "true"
		locks, err := resticProxy.UnlockRepoById(repository, all)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", locks)
	}
}

func forgetHandler() iris.Handler {
	return func(ctx *context.Context) {
		repository, err := ctx.Params().GetInt("repository")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		snapshotid := ctx.URLParam("snapshotid")
		var snapshotids []string
		if snapshotid != "" {
			snapshotids = strings.Split(snapshotid, ",")
		}
		opt := resticProxy.ForgetOptions{
			Prune: true,
		}
		id, err := resticProxy.RunForget(opt, repository, snapshotids)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", id)
	}
}

func Install(parent iris.Party) {
	// restic 直接操作接口
	sp := parent.Party("/restic")
	// restic ls命令
	sp.Get("/:repository/ls/:snapshotid", lsHandler())
	// restic find命令
	sp.Get("/:repository/search/:snapshotid", searchHandler())
	// 获取仓库快照
	sp.Get("/:repository/snapshots", snapshotsHandler())
	sp.Get("/:repository/parms", parmsHandler())
	sp.Get("/:repository/parmsForMy", parmsMyHandler())
	sp.Get("/:repository/loadIndex", loadIndexHandler())
	sp.Post("/:repository/check", checkHandler())
	sp.Post("/:repository/rebuild-index", rebuildIndexHandler())
	sp.Post("/:repository/prune", pruneHandler())
	sp.Post("/:repository/forget", forgetHandler())
	sp.Post("/:repository/migrate", migrateHandler())
	sp.Post("/:repository/unlock", unlockHandler())
}

func SplitSnapshotGroupBy(s string) (restic.SnapshotGroupByOptions, error) {
	var l restic.SnapshotGroupByOptions
	for _, option := range strings.Split(s, ",") {
		switch option {
		case "host", "hosts":
			l.Host = true
		case "path", "paths":
			l.Path = true
		case "tag", "tags":
			l.Tag = true
		case "":
		default:
			return restic.SnapshotGroupByOptions{}, fmt.Errorf("unknown grouping option: %q", option)
		}
	}
	return l, nil
}
