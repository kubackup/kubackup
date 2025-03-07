package i18n

import (
	"github.com/kataras/iris/v12/context"
)

// 语言上下文键
const LanguageContextKey = "language"

// LanguageMiddleware 语言中间件
func LanguageMiddleware() context.Handler {
	return func(ctx *context.Context) {
		// 从请求头中获取语言设置
		lang := ctx.GetHeader("Accept-Language")
		
		// 如果请求头中没有语言设置，尝试从查询参数中获取
		if lang == "" {
			lang = ctx.URLParam("lang")
		}
		
		// 如果查询参数中没有语言设置，尝试从 Cookie 中获取
		if lang == "" {
			lang = ctx.GetCookie("lang")
		}
		
		// 如果没有找到语言设置，使用默认语言
		if lang != ZH_CN && lang != EN_US {
			lang = DefaultLanguage
		}
		
		// 将语言设置存储在上下文中
		ctx.Values().Set(LanguageContextKey, lang)
		
		ctx.Next()
	}
}

// GetLanguage 从上下文中获取语言设置
func GetLanguage(ctx *context.Context) string {
	lang := ctx.Values().GetString(LanguageContextKey)
	if lang == "" {
		return DefaultLanguage
	}
	return lang
} 