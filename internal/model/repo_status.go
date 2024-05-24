package model

type RepoStatus struct {
	Id     int    `json:"id"`
	Status bool   `json:"status"`
	Errmsg string `json:"errmsg"`
}
