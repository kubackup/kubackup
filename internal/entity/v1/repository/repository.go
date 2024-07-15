package repository

import (
	"github.com/kubackup/kubackup/internal/entity/v1/common"
)

// Repository 参考restic
type Repository struct {
	common.BaseModel `storm:"inline"`
	Name             string `json:"name"`
	Type             int    `json:"type"`
	Endpoint         string `json:"endPoint"`
	// AWS_DEFAULT_REGION
	Region string `json:"region"`
	Bucket string `json:"bucket"`
	// AWS_ACCESS_KEY_ID
	KeyId string `json:"keyId"`
	// AWS_SECRET_ACCESS_KEY
	Secret string `json:"secret"`
	// GOOGLE_PROJECT_ID
	ProjectID string `json:"projectId"`
	// AZURE_ACCOUNT_NAME
	AccountName string `json:"accountName"`
	// AZURE_ACCOUNT_KEY，B2_ACCOUNT_KEY
	AccountKey string `json:"accountKey"`
	// B2_ACCOUNT_ID
	AccountID string `json:"accountId"`
	// 密码
	Password          string `json:"password"`
	Status            int    `json:"status"`
	Errmsg            string `json:"errmsg"`
	RepositoryVersion string `json:"repositoryVersion"`
	Compression       int    `json:"compression"` //压缩模式auto:0、off:1、max:2
	PackSize          int    `json:"packSize"`
}

// Type
const (
	S3     = 1
	Alioos = 2
	Sftp   = 3
	Local  = 4
	Rest   = 5
	HwObs  = 6
	TxCos  = 7
)

const (
	StatusNone = 1
	StatusRun  = 2
	StatusErr  = 3
)
