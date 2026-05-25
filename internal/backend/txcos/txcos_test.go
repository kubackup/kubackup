package txcos

import (
	"context"
	"net/http"
	"testing"

	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/location"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/test"
	rtest "github.com/kubackup/kubackup/pkg/restic_source/rinternal/test"
)

func TestBackendTxCos(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	newTxCosTestSuite(ctx, t).RunTests(t)
}

func BenchmarkBackendTxCos(t *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	newTxCosTestSuite(ctx, t).RunBenchmarks(t)
}

type TxCosTestConfig struct {
	Config
}

func newTxCosTestSuite(ctx context.Context, t testing.TB) *test.Suite[TxCosTestConfig] {
	return &test.Suite[TxCosTestConfig]{
		NewConfig: func() (*TxCosTestConfig, error) {
			cfg := &TxCosTestConfig{}
			rtest.TempDir(t)
			cfg.Config = NewConfig()
			cfg.Config.Endpoint = ""
			cfg.Config.SecretID = ""
			cfg.Config.SecretKey = ""
			cfg.Config.Prefix = "/"
			return cfg, nil
		},

		Factory: location.NewHTTPBackendFactory[TxCosTestConfig, backend.Backend](
			"txcos",
			func(s string) (*TxCosTestConfig, error) {
				return &TxCosTestConfig{}, nil
			},
			func(s string) string {
				return s
			},
			func(ctx context.Context, cfg TxCosTestConfig, rt http.RoundTripper) (backend.Backend, error) {
				return Create(ctx, cfg.Config, rt)
			},
			func(ctx context.Context, cfg TxCosTestConfig, rt http.RoundTripper) (backend.Backend, error) {
				return Open(ctx, cfg.Config, rt)
			},
		),
	}
}
