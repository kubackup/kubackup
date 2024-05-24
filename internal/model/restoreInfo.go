package model

type RestoreInfo struct {
	Exclude  string `json:"exclude"`
	IExclude string `json:"iExclude"`
	Include  string `json:"include"`
	IInclude string `json:"iInclude"`
	Target   string `json:"target"`
	Hosts    string `json:"hosts"`
	Paths    string `json:"paths"`
	Tags     string `json:"tags"`
	Verify   bool   `json:"verify"`
}
