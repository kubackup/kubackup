package txcos

import (
	"net/url"
	"strings"

	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/options"
)

type Config struct {
	Endpoint     string //https://examplebucket-appid.cos.COS_REGION.myqcloud.com
	SecretID     string //*** Provide your Access Key ***
	SecretKey    string //*** Provide your Secret Key ***
	StorageClass string `option:"storage-class" help:"set OBS storage class (STANDARD, WARM, COLD)"`
	Connections  uint
	Layout       string `option:"layout" help:"use this backend layout (default: auto-detect)"`
	Prefix       string
	EnableCRC    bool //CRC64 校验
}

func init() {
	options.Register("cos", Config{})
}

// NewConfig returns a new Config with the default values filled in.
func NewConfig() Config {
	return Config{
		Connections: 5,
		Prefix:      "",
		EnableCRC:   true,
	}
}

// ParseConfig parses the string s and extracts the cos config. The two
// supported configuration formats are cos://host/bucketname/prefix and
// cos:host/bucketname/prefix. The host can also be a valid obs region
func ParseConfig(s string) (*Config, error) {
	switch {
	case strings.HasPrefix(s, "cos:http"):
		// assume that a URL has been specified, parse it and
		// use the host as the endpoint and the path as the
		// bucket name and prefix
		url, err := url.Parse(s[4:])
		if err != nil {
			return nil, errors.Wrap(err, "url.Parse")
		}
		if url.Path != "" {
			path := strings.SplitN(url.Path[1:], "/", 2)
			return createConfig(url.Scheme+"://"+url.Host, path)
		} else {
			return createConfig(url.Scheme+"://"+url.Host, nil)
		}

	case strings.HasPrefix(s, "cos://"):
		s = s[6:]
	case strings.HasPrefix(s, "cos:"):
		s = s[4:]
	default:
		return nil, errors.New("cos: invalid format")
	}
	// use the first entry of the path as the endpoint and the
	// remainder as bucket name and prefix
	path := strings.SplitN(s, "/", 3)
	return createConfig(path[0], path[1:])
}

func createConfig(endpoint string, p []string) (*Config, error) {
	cfg := NewConfig()
	if len(p) > 0 {
		cfg.Prefix = p[0]
	}
	cfg.Endpoint = endpoint

	return &cfg, nil
}

func (cfg *Config) ApplyEnvironment(prefix string) {

}
