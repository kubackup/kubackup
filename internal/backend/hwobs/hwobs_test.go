package hwobs

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
	tempdir       string
	removeTempdir func()
}

func newHwObsTestSuite(ctx context.Context, t testing.TB) *test.Suite {
	tr, err := backend.Transport(backend.TransportOptions{})
	if err != nil {
		t.Fatalf("cannot create transport for tests: %v", err)
	}

	return &test.Suite{
		// NewConfig returns a config for a new temporary backend that will be used in tests.
		NewConfig: func() (interface{}, error) {
			cfg := HwObsTestConfig{}
			cfg.tempdir, cfg.removeTempdir = rtest.TempDir(t)
			cfg.Config = NewConfig()
			cfg.Config.Endpoint = ""
			cfg.Config.BucketName = ""
			cfg.Config.Ak = ""
			cfg.Config.Sk = ""
			return cfg, nil
		},

		// CreateFn is a function that creates a temporary repository for the tests.
		Create: func(config interface{}) (restic.Backend, error) {
			cfg := config.(HwObsTestConfig)

			be, err := createHwobs(t, cfg, tr)
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
			cfg := config.(HwObsTestConfig)
			return Open(ctx, cfg.Config, tr)
		},

		// CleanupFn removes data created during the tests.
		Cleanup: func(config interface{}) error {
			cfg := config.(HwObsTestConfig)
			if cfg.removeTempdir != nil {
				cfg.removeTempdir()
			}
			return nil
		},
	}
}

func createHwobs(t testing.TB, cfg HwObsTestConfig, tr http.RoundTripper) (be restic.Backend, err error) {
	be, err = Create(context.TODO(), cfg.Config, tr)
	if err != nil {
		t.Logf("Hwobs open error %v", err)
	}
	return be, err
}
