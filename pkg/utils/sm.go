package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm3"
	"github.com/tjfoc/gmsm/sm4"
	"github.com/tjfoc/gmsm/x509"
	"hash"
)

func Sm2GenerateKey(pwd string) (prikey []byte, pubKey []byte, err error) {
	priv, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return
	}
	var p []byte
	if pwd != "" {
		p = []byte(pwd)
	}
	keyToPem, err := x509.WritePrivateKeyToPem(priv, p)
	if err != nil {
		return nil, nil, err
	}
	publicKeyToPem, err := x509.WritePublicKeyToPem(priv.Public().(*sm2.PublicKey))
	if err != nil {
		return nil, nil, err
	}
	return keyToPem, publicKeyToPem, nil
}

func Sm2ReadPrivateKeyFromPem(key []byte, pwd string) *sm2.PrivateKey {
	privateKey, err := x509.ReadPrivateKeyFromPem(key, []byte(pwd))
	if err != nil {
		return nil
	}
	return privateKey
}

func Sm2ReadPublicKeyFromPem(key []byte) *sm2.PublicKey {
	publicKey, err := x509.ReadPublicKeyFromPem(key)
	if err != nil {
		return nil
	}
	return publicKey
}

func Sm2ReadCertificateFromPem(cert []byte) *x509.Certificate {
	certificate, err := x509.ReadCertificateFromPem(cert)
	if err != nil {
		return nil
	}
	return certificate
}

func Sm2ReadCertificateRequestFromPem(cert []byte) *x509.CertificateRequest {
	certificate, err := x509.ReadCertificateRequestFromPem(cert)
	if err != nil {
		return nil
	}
	return certificate
}

func Sm3Hash() hash.Hash {
	return sm3.New()
}

// Sm3 加密
func Sm3(data []byte) []byte {
	s := Sm3Hash()
	s.Write(data)
	return s.Sum(nil)
}

func SetIV(data []byte) {
	_ = sm4.SetIV(data)
}

// Sm4Encrypt 加密
func Sm4Encrypt(key string, data string) string {
	ecbMsg, err := sm4.Sm4Ecb([]byte(key), []byte(data), true)
	if err != nil {
		fmt.Println(err)
	}
	return hex.EncodeToString(ecbMsg)
}

// Sm4Decrypt 解密
func Sm4Decrypt(key string, sec string) string {
	secb, err := hex.DecodeString(sec)
	if err != nil {
		fmt.Println(err)
	}
	ecbMsg, err := sm4.Sm4Ecb([]byte(key), secb, false)
	if err != nil {
		fmt.Println(err)
	}
	return string(ecbMsg)
}
