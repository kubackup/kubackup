package model

type DumpInfo struct {
	Filename string `json:"filename"`
	Type     string `json:"type"` // dir / file
	Mode     int    `json:"mode"`
}
