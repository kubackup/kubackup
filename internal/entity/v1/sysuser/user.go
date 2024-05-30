package sysuser

import (
	"github.com/kubackup/kubackup/internal/entity/v1/common"
	"time"
)

type SysUser struct {
	common.BaseModel `storm:"inline"`
	Username         string    `json:"userName" storm:"index,unique"` // 账号
	NickName         string    `json:"nickName"`                      //昵称
	Password         string    `json:"password"`                      //密码
	Email            string    `json:"email"`                         //邮箱
	Phone            string    `json:"phone"`                         //手机号码
	OtpSecret        string    `json:"OtpSecret"`                     //otp密钥
	OtpInterval      int       `json:"otpInterval"`                   //otp 步数
	LastLogin        time.Time `json:"lastLogin"`                     //最后登录时间
}
