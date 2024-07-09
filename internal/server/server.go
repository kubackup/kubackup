package server

import (
	"embed"
	"fmt"
	"github.com/asdine/storm/v3"
	"github.com/fanjindong/go-cache"
	prometheusMiddleware "github.com/iris-contrib/middleware/prometheus"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/middleware/pprof"
	"github.com/kataras/iris/v12/view"
	cf "github.com/kubackup/kubackup/internal/config"
	"github.com/kubackup/kubackup/internal/consts/system_status"
	"github.com/kubackup/kubackup/internal/entity/v1/config"
	fileutil "github.com/kubackup/kubackup/pkg/file"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/utils/docker"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type BackupServer struct {
	app          *iris.Application
	logger       *logrus.Logger
	rootRoute    iris.Party
	config       *config.Config
	db           *storm.DB
	cache        cache.ICache
	systemStatus string
	isDocker     bool //是否运行在docker中
}

var EmbedWebDashboard embed.FS

var bs *BackupServer

func Listen(route func(party iris.Party), host string, port int, path string) error {
	bs = NewBackupServer(host, port, path)
	route(bs.rootRoute)
	return bs.app.Run(iris.Addr(fmt.Sprintf("%s:%d", bs.config.Server.Bind.Host, bs.config.Server.Bind.Port)), iris.WithoutInterruptHandler)
}

func NewBackupServer(host string, port int, path string) *BackupServer {
	server := &BackupServer{systemStatus: system_status.Normal}
	server.isDocker = docker.IsDockerEnv()
	server.app = iris.New()
	c, err := cf.ReadConfig(path)
	if err != nil {
		panic(err)
	}
	server.config = c
	bs.UpdateHost(host)
	bs.UpdatePort(port)
	return server.bootstrap()
}
func UpdateSystemStatus(ok string) {
	bs.systemStatus = ok
}
func (e *BackupServer) bootstrap() *BackupServer {
	e.setUpRootRoute()
	e.setUpStaticFile()
	e.setUpLogger()
	e.setResultHandler()
	e.setUpErrHandler()
	e.setUpPrometheusHandler()
	e.setUpPprofHandler()
	e.setUpDB()
	e.setStop()
	e.setUpCache()
	return e
}

func (e *BackupServer) setUpStaticFile() {
	spaOption := iris.DirOptions{SPA: true, IndexName: "index.html"}
	party := e.rootRoute.Party("/")
	party.Use(iris.Compression)
	dashboardFS := iris.PrefixDir("web/dashboard", http.FS(EmbedWebDashboard))
	party.RegisterView(view.HTML(dashboardFS, ".html"))
	party.HandleDir("/", dashboardFS, spaOption)
}

func (e *BackupServer) setStop() {
	iris.RegisterOnInterrupt(func() {
		fmt.Println("关闭服务")
		os.Exit(1)
	})
}
func (e *BackupServer) UpdateHost(host string) {
	if "" != host {
		e.config.Server.Bind.Host = host
	}
}

func (e *BackupServer) UpdatePort(port int) {
	if 0 != port {
		e.config.Server.Bind.Port = port
	}
}

func (e *BackupServer) setUpRootRoute() {
	e.rootRoute = e.app.Party("/")
}

func (e *BackupServer) setUpDB() {
	dbpath := fileutil.ReplaceHomeDir(e.config.Data.DbDir)
	if !fileutil.Exist(dbpath) {
		if err := os.MkdirAll(dbpath, 0755); err != nil {
			panic(fmt.Errorf("can not create database dir: %s message: %s", e.config.Data.DbDir, err))
		}
	}
	d, err := storm.Open(path.Join(dbpath, string(filepath.Separator), "kubackup.db"))
	if err != nil {
		panic(err)
	}
	e.db = d
}

func (e *BackupServer) setUpLogger() {
	e.logger = logrus.New()
	var lstr string
	if e.config.Server.Debug {
		lstr = "debug"
	} else {
		lstr = e.config.Logger.Level
	}
	l, err := logrus.ParseLevel(lstr)
	if err != nil {
		e.logger.Errorf("cant not parse logger level %s, %s,use default level: INFO", e.config.Logger.Level, err)
	}
	e.logger.SetReportCaller(true)
	e.logger.SetLevel(l)
	e.logger.SetFormatter(&logrus.TextFormatter{})
	logpath := e.config.Logger.LogPath
	if !fileutil.Exist(logpath) {
		if err := os.MkdirAll(logpath, 0755); err != nil {
			panic(fmt.Errorf("can not create log dir: %s message: %s", logpath, err))
		}
	}
	logfile, err := fs.OpenFile(path.Join(logpath, string(filepath.Separator), "kubackup.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		e.logger.Error(err)
	}
	mw := io.MultiWriter(os.Stdout, logfile)
	e.logger.SetOutput(mw)
	e.app.Logger().SetOutput(mw)
	e.app.Logger().SetLevel(lstr)
}

func (e *BackupServer) setUpCache() {
	e.cache = cache.NewMemCache(cache.WithClearInterval(1 * time.Minute))
}

func DB() *storm.DB {
	return bs.db
}

func Cache() cache.ICache {
	return bs.cache
}

func Logger() *logrus.Logger {
	return bs.logger
}

func Config() *config.Config {
	return bs.config
}

func IsDocker() bool {
	return bs.isDocker
}

const ContentTypeDownload = "application/download"

// 全局结果处理
func (e *BackupServer) setResultHandler() {
	e.rootRoute.Use(func(ctx *context.Context) {
		if e.systemStatus != system_status.Normal {
			resp := iris.Map{
				"success":      true,
				"systemStatus": e.systemStatus,
				"data":         nil,
				"isDocker":     e.isDocker,
			}
			_ = ctx.JSON(resp, iris.JSON{})
			return
		}
		ctx.Next()
		contentType := ctx.ResponseWriter().Header().Get("Content-Type")
		if contentType == ContentTypeDownload {
			return
		}
		p := ctx.GetCurrentRoute().Path()
		if strings.HasPrefix(p, "/debug/pprof") {
			return
		}
		if strings.HasPrefix(p, "/metrics") {
			return
		}
		ss := strings.Split(p, "/")
		if len(ss) >= 3 {
			for i := range ss {
				if ss[i] == "ws" {
					return
				}
			}
		}
		if ctx.GetStatusCode() >= iris.StatusOK && ctx.GetStatusCode() < iris.StatusBadRequest {
			resp := iris.Map{
				"success":      true,
				"systemStatus": system_status.Normal,
				"data":         ctx.Values().Get("data"),
				"isDocker":     e.isDocker,
			}
			_ = ctx.JSON(resp, iris.JSON{})
		}
	})
}

// 全局异常处理
func (e *BackupServer) setUpErrHandler() {
	e.rootRoute.OnAnyErrorCode(func(ctx iris.Context) {
		if ctx.Values().GetString("message") == "" {
			switch ctx.GetStatusCode() {
			case iris.StatusNotFound:
				ctx.Values().Set("message", "the server could not find the requested resource")
			}
		}
		message := ctx.Values().Get("message")
		er := iris.Map{
			"success":      false,
			"systemStatus": system_status.Normal,
			"code":         ctx.GetStatusCode(),
			"message":      message,
			"isDocker":     e.isDocker,
		}
		ctx.StatusCode(iris.StatusOK)
		_ = ctx.JSON(er, iris.JSON{})
	})
}

func (e *BackupServer) setUpPrometheusHandler() {
	if !e.config.Prometheus.Enable {
		return
	}
	m := prometheusMiddleware.New(e.config.Server.Name, 0.3, 1.2, 5.0)
	e.rootRoute.Use(m.ServeHTTP)
	e.rootRoute.Get("/metrics", iris.FromStd(promhttp.Handler()))
}

func (e *BackupServer) setUpPprofHandler() {
	if !e.config.Server.Debug {
		return
	}
	p := pprof.New()
	e.rootRoute.Any("/debug/pprof", p)
	e.rootRoute.Any("/debug/pprof/{action:path}", p)
}
