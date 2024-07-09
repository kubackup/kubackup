package hwobs

import (
	"net/url"
	"path"
	"strings"

	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/options"
)

type Config struct {
	Endpoint     string
	Ak           string //*** Provide your Access Key ***
	Sk           string //*** Provide your Secret Key ***
	BucketName   string //bucket-test
	StorageClass string `option:"storage-class" help:"set OBS storage class (STANDARD, WARM, COLD)"`
	Connections  uint
	Layout       string `option:"layout" help:"use this backend layout (default: auto-detect)"`
	Prefix       string
	SslEnable    bool
}

func init() {
	options.Register("obs", Config{})
}

// NewConfig returns a new Config with the default values filled in.
func NewConfig() Config {
	return Config{
		Connections: 5,
		Prefix:      "",
	}
}

// ParseConfig parses the string s and extracts the obs config. The two
// supported configuration formats are obs://host/bucketname/prefix and
// obs:host/bucketname/prefix. The host can also be a valid obs region
func ParseConfig(s string) (*Config, error) {
	switch {
	case strings.HasPrefix(s, "obs:http"):
		// assume that a URL has been specified, parse it and
		// use the host as the endpoint and the path as the
		// bucket name and prefix
		url, err := url.Parse(s[4:])
		if err != nil {
			return nil, errors.Wrap(err, "url.Parse")
		}

		if url.Path == "" {
			return nil, errors.New("obs: bucket name not found")
		}

		path := strings.SplitN(url.Path[1:], "/", 2)
		return createConfig(url.Scheme+"://"+url.Host, path)
	case strings.HasPrefix(s, "obs:"):
		s = s[4:]
	default:
		return nil, errors.New("obs: invalid format")
	}
	// use the first entry of the path as the endpoint and the
	// remainder as bucket name and prefix
	path := strings.SplitN(s, "/", 3)
	return createConfig(path[0], path[1:])
}

func createConfig(endpoint string, p []string) (*Config, error) {
	if len(p) < 1 {
		return nil, errors.New("obs: invalid format, endpoint or bucket name not found")
	}
	cfg := NewConfig()
	cfg.Endpoint = endpoint
	cfg.BucketName = p[0]
	var prefix string
	if len(p) > 1 && p[1] != "" {
		prefix = path.Clean(p[1])
	}
	cfg.Prefix = prefix
	return &cfg, nil
}

func (cfg *Config) ApplyEnvironment(prefix string) {

}
