package utils

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"time"
)

var jwtSigner *jwt.Signer
var JwtKey string

func InitJwt() {
	JwtKey = server.Config().Jwt.Key
	t := server.Config().Jwt.MaxAge
	jwtMaxAge := time.Duration(t)
	jwtSigner = jwt.NewSigner(jwt.HS256, JwtKey, jwtMaxAge*time.Second)
}
func GetToken(data interface{}) (*model.TokenInfo, error) {
	token, err := jwtSigner.Sign(data)
	if err != nil {
		return nil, err
	}
	ti := &model.TokenInfo{
		Token:     string(token),
		ExpiresAt: time.Now().Add(jwtSigner.MaxAge),
	}
	return ti, nil
}

func GetJwtVerifier() *jwt.Verifier {
	j := jwt.NewVerifier(jwt.HS256, JwtKey)
	j.ErrorHandler = func(ctx iris.Context, err error) {
		ErrorCode(ctx, iris.StatusUnauthorized, err)
	}
	return j
}
