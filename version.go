package kubackup

type Verison struct {
	BuildTime string `json:"buildTime"`
	Verison   string `json:"verison"`
}

var (
	BuildTime string
	V         string
)

func GetVersion() *Verison {
	return &Verison{
		Verison:   V,
		BuildTime: BuildTime,
	}
}
