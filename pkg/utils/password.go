package utils

import "golang.org/x/crypto/bcrypt"

// EncodePWD 密码加密
func EncodePWD(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	return string(hash), err
}

// ComparePwd 比较密码密文与明文是否相等
func ComparePwd(pwd string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	if err != nil {
		return false
	} else {
		return true
	}
}
