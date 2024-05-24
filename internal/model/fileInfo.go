package model

type FileInfo struct {
	IsDir      bool   `json:"isDir"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Mode       string `json:"mode"`       // 权限
	ModTime    string `json:"modTime"`    // 修改时间
	CreateTime string `json:"createTime"` // 创建时间
	Size       int64  `json:"size"`       // 大小
	Gid        int    `json:"gid"`
	Uid        int    `json:"uid"`
}
