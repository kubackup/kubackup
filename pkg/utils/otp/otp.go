package otp

import (
	"bytes"
	"encoding/base64"
	"github.com/kubackup/kubackup/internal/consts/global"
	"github.com/skip2/go-qrcode"
	"github.com/xlzd/gotp"
	"strconv"
	"time"
)

type Otp struct {
	Secret   string `json:"secret"`
	QrImg    string `json:"qrImg"`
	Interval int    `json:"interval"`
}

// GetOtp 获取otp二维码
func GetOtp(username, title string, interval int) (otp Otp, err error) {
	secret := gotp.RandomSecret(global.OTPSecretLength)
	otp = Otp{Secret: secret, Interval: interval}
	totp := gotp.NewTOTP(secret, global.OTPDigits, interval, nil)
	uri := totp.ProvisioningUri(username, title)
	subImg, err := qrcode.Encode(uri, qrcode.Medium, 256)
	dist := make([]byte, 3000)
	base64.StdEncoding.Encode(dist, subImg)
	index := bytes.IndexByte(dist, 0)
	baseImage := dist[0:index]
	otp.QrImg = "data:image/png;base64," + string(baseImage)
	return
}

// ValidCode 验证otp
func ValidCode(code string, interval int, secret string) bool {
	totp := gotp.NewTOTP(secret, global.OTPDigits, interval, nil)
	now := time.Now().Unix()
	strInt64 := strconv.FormatInt(now, 10)
	id16, _ := strconv.Atoi(strInt64)
	return totp.Verify(code, int64(id16))
}
