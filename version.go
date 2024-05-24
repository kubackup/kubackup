package backup

type Verison struct {
	GitTag    string `json:"gitTag"`
	BuildTime string `json:"buildTime"`
	Verison   string `json:"verison"`
}

var (
	GitTag    string
	BuildTime string
	V         string
)

func GetVersion() *Verison {
	return &Verison{
		GitTag:    GitTag,
		Verison:   V,
		BuildTime: BuildTime,
	}
}
