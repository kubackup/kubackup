package route

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/kubackup/kubackup/internal/api"
	v1 "github.com/kubackup/kubackup/internal/api/v1"
	"github.com/kubackup/kubackup/internal/cron"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/user"
	"github.com/kubackup/kubackup/pkg/utils"
	"github.com/kubackup/kubackup/restic_proxy"
)

func InitRoute(party iris.Party) {
	initOthers()
	apiParty := party.Party("/api")
	api.AddPingRoute(apiParty)
	v1.AddV1Route(apiParty)
	ininPrint()
}
func initOthers() {
	go resticProxy.InitRepository()
	initAdmin()
	utils.InitJwt()
	go cron.InitCron()
}

func ininPrint() {
	if server.Config().Prometheus.Enable {
		fmt.Printf("Prometheus is deploy to: http://%s:%d/%s\n",
			server.Config().Server.Bind.Host,
			server.Config().Server.Bind.Port,
			"metrics")
	}
	fmt.Printf("Health Check is deploy to: http://%s:%d/%s\n",
		server.Config().Server.Bind.Host,
		server.Config().Server.Bind.Port,
		"api/ping")
}

// initAdmin 初始化admin账号
func initAdmin() {
	userServer := user.GetService()
	err := userServer.InitAdmin()
	if err != nil {
		fmt.Println("初始化admin账号失败：", err.Error())
		return
	}
}
