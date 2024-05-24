package utils

import (
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/middleware/jwt"
	"github.com/kubackup/kubackup/internal/model"
	"time"
)

func GetCurUser(ctx *context.Context) *model.Userinfo {
	ctx.User()
	var u *model.Userinfo
	if ctx.GetHeader("Authorization") != "" {
		u = jwt.Get(ctx).(*model.Userinfo)
	}
	return u
}
func GetTokenExpires(ctx *context.Context) time.Time {
	if ctx.GetHeader("Authorization") != "" {
		vt := jwt.GetVerifiedToken(ctx)
		if vt != nil {
			return vt.StandardClaims.ExpiresAt()
		}
	}
	return time.Now()
}
