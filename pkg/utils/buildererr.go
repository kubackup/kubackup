package utils

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kubackup/kubackup/internal/i18n"
)

func ErrorCode(ctx *context.Context, code int, err error) {
	if err == nil {
		return
	}
	errstring := err.Error()
	
	// 获取当前语言
	lang := i18n.GetLanguage(ctx)
	
	// 尝试翻译错误信息
	// 如果错误信息是一个翻译键，则翻译它
	translatedErr := i18n.T(errstring, lang)
	
	ctx.StatusCode(code)
	ctx.Values().Set("message", translatedErr)
}

func Errore(ctx *context.Context, err error) {
	ErrorCode(ctx, iris.StatusBadRequest, err)
}

func ErrorStr(ctx *context.Context, err string) {
	// 获取当前语言
	lang := i18n.GetLanguage(ctx)
	
	// 尝试翻译错误信息
	translatedErr := i18n.T(err, lang)
	
	Errore(ctx, fmt.Errorf(translatedErr))
}
