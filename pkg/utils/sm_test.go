package utils

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSm2GenerateKey(t *testing.T) {
	prikey, pubKey, err := Sm2GenerateKey("")
	assert.NoError(t, err)
	t.Log(string(prikey))
	t.Log(string(pubKey))
}

func TestSm2ReadPrivateKeyFromPem(t *testing.T) {
	key := `-----BEGIN PRIVATE KEY-----
MIGTAgEAMBMGByqGSM49AgEGCCqBHM9VAYItBHkwdwIBAQQgsA2g8Q8Rs85QI/gU
LxICY5UqOl5VM7lVyMUACqRytBugCgYIKoEcz1UBgi2hRANCAASqo4Wb9YEqEt9i
Z6+MLJGWtw+dmITrNppjPwt+zlYkkns7vTG1Kf4ZJDmuww//taDKYgbA1AJ0mEbK
HJ34I8XL
-----END PRIVATE KEY-----`
	privateKey := Sm2ReadPrivateKeyFromPem([]byte(key), "")
	sign, err := privateKey.Sign(rand.Reader, []byte("13"), nil)
	if err != nil {
		return
	}
	t.Log(string(sign))
}

func TestSm3(t *testing.T) {
	mi := Sm3([]byte("123sfadsfafsdfsdfsdfsdf4"))
	t.Log(string(mi))
}

func TestSm4Encrypt(t *testing.T) {
	key := "123321jdieu37dud"
	data := "123456jklhskjdhfklahfkdfds"
	enc := Sm4Encrypt(key, data)
	t.Log(enc)
	decrypt := Sm4Decrypt(key, enc)
	t.Log(decrypt)
	assert.EqualValues(t, decrypt, data)
}
