package user

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup/internal/consts"
	"github.com/kubackup/kubackup/internal/entity/v1/oplog"
	"github.com/kubackup/kubackup/internal/entity/v1/sysuser"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	logser "github.com/kubackup/kubackup/internal/service/v1/oplog"
	"github.com/kubackup/kubackup/internal/service/v1/user"
	"github.com/kubackup/kubackup/pkg/utils"
	"github.com/kubackup/kubackup/pkg/utils/otp"
	"strings"
	"time"
)

var userService user.Service
var logService logser.Service

var pwderrKey = "PwdErrCount:"

// Second
var lockTime = 1800

func init() {
	userService = user.GetService()
	logService = logser.GetService()
}

func loginHandler() iris.Handler {
	return func(ctx *context.Context) {
		var loginData model.LoginData
		err := ctx.ReadJSON(&loginData)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		udb := login(ctx, loginData.Username, loginData.Password)
		if udb == nil {
			return
		}
		udb.LastLogin = time.Now()
		userinfo := &model.Userinfo{
			Id:        udb.Id,
			Username:  udb.Username,
			NickName:  udb.NickName,
			Email:     udb.Email,
			Phone:     udb.Phone,
			Mfa:       false,
			LastLogin: udb.LastLogin.Format(consts.Custom),
		}
		if udb.OtpSecret != "" {
			userinfo.Mfa = true
		}
		// 关闭后设置为 1
		if udb.OtpSecret == "1" {
			userinfo.Mfa = false
		}

		if userinfo.Mfa {
			if loginData.Code == "" {
				ctx.Values().Set("data", userinfo)
				return
			} else {
				if !otp.ValidCode(loginData.Code, udb.OtpInterval, udb.OtpSecret) {
					utils.Errore(ctx, err)
					return
				}
			}
		}
		token, err := utils.GetToken(userinfo)
		if err != nil {
			utils.ErrorStr(ctx, consts.TokenGenErrstr)
			return
		}
		userinfo.Token = token
		ctx.Values().Set("data", userinfo)
		go func() {
			// 更新最后登录日志
			_ = userService.Update(udb, common.DBOptions{})
			var log oplog.OperationLog
			log.Operator = udb.Username
			log.Operation = "post"
			log.Url = "/api/v1/login"
			// 新增登录日志
			_ = logService.Create(&log, common.DBOptions{})
		}()
	}
}

func login(ctx *context.Context, username, password string) *sysuser.SysUser {
	u, err := userService.GetByUserName(username, common.DBOptions{})
	errk := pwderrKey + username
	count, ok := utils.Get(errk)
	if !ok || count == nil {
		count = 0
	}
	errcount := count.(int)
	if errcount >= 3 {
		utils.ErrorStr(ctx, fmt.Sprintf(consts.LockErrstr, errcount, lockTime/60))
		utils.Set(errk, errcount, lockTime)
		return nil
	}
	if err != nil || u == nil {
		utils.ErrorStr(ctx, consts.Pwderrstr)
		utils.Set(errk, errcount+1, lockTime)
		return nil
	}
	if !utils.ComparePwd(password, u.Password) {
		utils.ErrorStr(ctx, consts.Pwderrstr)
		utils.Set(errk, errcount+1, lockTime)
		return nil
	}
	return u
}

func refreshTokenHandler() iris.Handler {
	return func(ctx *context.Context) {
		curuser := utils.GetCurUser(ctx)
		token, err := utils.GetToken(curuser)
		if err != nil {
			utils.ErrorStr(ctx, "token生成失败")
			return
		}
		ctx.Values().Set("data", token)
	}

}

func listHandler() iris.Handler {
	return func(ctx *context.Context) {
		var users []sysuser.SysUser
		users, err := userService.List(common.DBOptions{})
		if err != nil && err.Error() != "not found" {
			utils.Errore(ctx, err)
			return
		}
		for key, sysUser := range users {
			// 清除敏感数据
			sysUser.Password = ""
			sysUser.OtpSecret = ""
			sysUser.OtpInterval = 0
			users[key] = sysUser
		}
		ctx.Values().Set("data", users)
	}
}

func updateHanlder() iris.Handler {
	return func(ctx *context.Context) {
		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		var muser model.Userinfo
		err = ctx.ReadJSON(&muser)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}

		sysUser, err := userService.Get(id, common.DBOptions{})
		if err != nil {
			return
		}
		sysUser.NickName = muser.NickName
		sysUser.Email = muser.Email
		sysUser.Phone = muser.Phone
		err = userService.Update(sysUser, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "")
	}
}

func createHanlder() iris.Handler {
	return func(ctx *context.Context) {
		var muser sysuser.SysUser
		err := ctx.ReadJSON(&muser)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if strings.TrimSpace(muser.Password) == "" {
			utils.ErrorStr(ctx, "密码不能为空")
			return
		}
		if len(muser.Password) < 6 {
			utils.ErrorStr(ctx, "密码长度不能少于6位")
			return
		}
		encodePWD, err := utils.EncodePWD(muser.Password)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		muser.Password = encodePWD
		err = userService.Create(&muser, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "")
	}
}

func delHanlder() iris.Handler {
	return func(ctx *context.Context) {
		id, err := ctx.Params().GetInt("id")
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		sysUser, err := userService.Get(id, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if sysUser.Username == "admin" {
			utils.ErrorStr(ctx, "admin账号不能删除！")
			return
		}
		curu := utils.GetCurUser(ctx)
		if curu.Id == sysUser.Id {
			utils.ErrorStr(ctx, "不能删除自己！")
			return
		}
		err = userService.DeleteStruct(sysUser, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "")
	}
}
func repwdHanlder() iris.Handler {
	return func(ctx *context.Context) {
		var pwdData model.RePwdData
		err := ctx.ReadJSON(&pwdData)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if strings.TrimSpace(pwdData.Password) == "" {
			utils.ErrorStr(ctx, "密码不能为空")
			return
		}
		if len(pwdData.Password) < 6 {
			utils.ErrorStr(ctx, "密码长度不能少于6位")
			return
		}
		curuser := utils.GetCurUser(ctx)
		sysUser, err := userService.Get(curuser.Id, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		if !utils.ComparePwd(pwdData.OldPassword, sysUser.Password) {
			utils.ErrorStr(ctx, consts.RePwderrstr)
			return
		}
		newPwd, err := utils.EncodePWD(pwdData.Password)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		sysUser.Password = newPwd
		err = userService.Update(sysUser, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "修改成功")
	}
}

func otpHandler() iris.Handler {
	return func(ctx *context.Context) {
		curu := utils.GetCurUser(ctx)
		getOtp, err := otp.GetOtp(curu.Username, "Kubackup", 30)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", getOtp)
	}
}

func bindOtpHandler() iris.Handler {
	return func(ctx *context.Context) {
		var otpInfo model.OtpInfo
		err := ctx.ReadJSON(&otpInfo)
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		res := otp.ValidCode(otpInfo.Code, otpInfo.Interval, otpInfo.Secret)
		if !res {
			utils.ErrorStr(ctx, "验证码错误！")
			return
		}
		curu := utils.GetCurUser(ctx)
		sysUser, err := userService.Get(curu.Id, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		sysUser.OtpSecret = otpInfo.Secret
		sysUser.OtpInterval = otpInfo.Interval
		err = userService.Update(sysUser, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "绑定成功")
	}
}

func putOtpHandler() iris.Handler {
	return func(ctx *context.Context) {
		curu := utils.GetCurUser(ctx)
		err := userService.ClearOtp(curu.Username, common.DBOptions{})
		if err != nil {
			utils.Errore(ctx, err)
			return
		}
		ctx.Values().Set("data", "关闭成功")
	}
}

func Install(parent iris.Party) {
	// 用户相关接口
	sp := parent.Party("/")
	// 登录
	sp.Post("/login", loginHandler())
	// 刷新token
	sp.Post("/refreshToken", refreshTokenHandler())
	// 获取用户列表
	sp.Get("/user", listHandler())
	// 删除用户
	sp.Delete("/user/:id", delHanlder())
	sp.Put("/user/:id", updateHanlder())
	sp.Post("/user", createHanlder())
	// 修改密码
	sp.Post("/repwd", repwdHanlder())
	// 获取otp二维码
	sp.Get("/otp", otpHandler())
	//绑定otp
	sp.Post("/otp", bindOtpHandler())

	sp.Put("/otp", putOtpHandler())

}
