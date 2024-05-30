package kubackup

type Version struct {
	BuildTime string `json:"buildTime"`
	Version   string `json:"version"`
}

var (
	BuildTime string
	V         string
)

func GetVersion() *Version {
	return &Version{
		Version:   V,
		BuildTime: BuildTime,
	}
}
