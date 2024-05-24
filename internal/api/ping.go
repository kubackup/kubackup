package api

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"time"
)

func pingHandler() iris.Handler {
	return func(ctx *context.Context) {
		ctx.Values().Set("data", time.Now())
	}
}

func AddPingRoute(app iris.Party) {
	// 用于检活
	app.Get("/ping", pingHandler())
}
