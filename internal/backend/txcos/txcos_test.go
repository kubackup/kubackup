package txcos

import (
	"context"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/test"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	rtest "github.com/kubackup/kubackup/pkg/restic_source/rinternal/test"
	"net/http"
	"testing"
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
	tempdir       string
	removeTempdir func()
}

func newTxCosTestSuite(ctx context.Context, t testing.TB) *test.Suite {
	tr, err := backend.Transport(backend.TransportOptions{})
	if err != nil {
		t.Fatalf("cannot create transport for tests: %v", err)
	}

	return &test.Suite{
		// NewConfig returns a config for a new temporary backend that will be used in tests.
		NewConfig: func() (interface{}, error) {
			cfg := TxCosTestConfig{}
			cfg.tempdir, cfg.removeTempdir = rtest.TempDir(t)
			cfg.Config = NewConfig()
			cfg.Config.Endpoint = ""
			cfg.Config.SecretID = ""
			cfg.Config.SecretKey = ""
			cfg.Config.Prefix = "/"
			return cfg, nil
		},

		// CreateFn is a function that creates a temporary repository for the tests.
		Create: func(config interface{}) (restic.Backend, error) {
			cfg := config.(TxCosTestConfig)

			be, err := createTxCos(t, cfg, tr)
			if err != nil {
				return nil, err
			}

			exists, err := be.Test(ctx, restic.Handle{Type: restic.ConfigFile})
			if err != nil {
				return nil, err
			}

			if exists {
				return nil, errors.New("config already exists")
			}

			return be, nil
		},

		// OpenFn is a function that opens a previously created temporary repository.
		Open: func(config interface{}) (restic.Backend, error) {
			cfg := config.(TxCosTestConfig)
			return Open(ctx, cfg.Config, tr)
		},

		// CleanupFn removes data created during the tests.
		Cleanup: func(config interface{}) error {
			cfg := config.(TxCosTestConfig)
			if cfg.removeTempdir != nil {
				cfg.removeTempdir()
			}
			return nil
		},
	}
}

func createTxCos(t testing.TB, cfg TxCosTestConfig, tr http.RoundTripper) (be restic.Backend, err error) {
	be, err = Create(context.TODO(), cfg.Config, tr)
	if err != nil {
		t.Logf("TxCos open error %v", err)
	}
	return be, err
}
