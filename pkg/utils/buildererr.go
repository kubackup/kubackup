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
	
	// Get current language
	lang := i18n.GetLanguage(ctx)
	
	// Try to translate error message
	// If the error message is a translation key, translate it
	translatedErr := i18n.T(errstring, lang)
	
	ctx.StatusCode(code)
	ctx.Values().Set("message", translatedErr)
}

func Errore(ctx *context.Context, err error) {
	ErrorCode(ctx, iris.StatusBadRequest, err)
}

func ErrorStr(ctx *context.Context, err string) {
	// Get current language
	lang := i18n.GetLanguage(ctx)
	
	// Try to translate error message
	translatedErr := i18n.T(err, lang)
	
	Errore(ctx, fmt.Errorf(translatedErr))
}
