package hwobs

import (
	"context"
	"net/http"
	"testing"

	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/location"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/test"
	rtest "github.com/kubackup/kubackup/pkg/restic_source/rinternal/test"
)

func TestBackendHwobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	newHwObsTestSuite(ctx, t).RunTests(t)
}

func BenchmarkBackendHwobs(t *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	newHwObsTestSuite(ctx, t).RunBenchmarks(t)
}

type HwObsTestConfig struct {
	Config
}

func newHwObsTestSuite(ctx context.Context, t testing.TB) *test.Suite[HwObsTestConfig] {
	return &test.Suite[HwObsTestConfig]{
		NewConfig: func() (*HwObsTestConfig, error) {
			cfg := &HwObsTestConfig{}
			rtest.TempDir(t)
			cfg.Config = NewConfig()
			cfg.Config.Endpoint = ""
			cfg.Config.BucketName = ""
			cfg.Config.Ak = ""
			cfg.Config.Sk = ""
			return cfg, nil
		},

		Factory: location.NewHTTPBackendFactory[HwObsTestConfig, backend.Backend](
			"hwobs",
			func(s string) (*HwObsTestConfig, error) {
				return &HwObsTestConfig{}, nil
			},
			func(s string) string {
				return s
			},
			func(ctx context.Context, cfg HwObsTestConfig, rt http.RoundTripper) (backend.Backend, error) {
				return Create(ctx, cfg.Config, rt)
			},
			func(ctx context.Context, cfg HwObsTestConfig, rt http.RoundTripper) (backend.Backend, error) {
				return Open(ctx, cfg.Config, rt)
			},
		),
	}
}
