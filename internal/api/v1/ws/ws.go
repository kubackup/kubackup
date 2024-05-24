package ws

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup/internal/store/log"
	info "github.com/kubackup/kubackup/internal/store/task"
	task "github.com/kubackup/kubackup/internal/store/ws_task_info"
)

func Install(parent iris.Party) {
	// websocket端点
	wsParty := parent.Party("/ws")
	// 备份任务sockjs端点
	taskh := task.CreateTaskHandler("/task/sockjs", info.TaskInfos)
	wsParty.Any("/task/sockjs/{p:path}", func(context *context.Context) {
		taskh.ServeHTTP(context.ResponseWriter(), context.Request())
	})

	logh := task.CreateTaskHandler("/log/sockjs", log.LogInfos)
	wsParty.Any("/log/sockjs/{p:path}", func(context *context.Context) {
		logh.ServeHTTP(context.ResponseWriter(), context.Request())
	})
}
