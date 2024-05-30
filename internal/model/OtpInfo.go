package model

type OtpInfo struct {
	Secret   string `json:"secret"`
	Code     string `json:"code"`
	Interval int    `json:"interval"`
}
