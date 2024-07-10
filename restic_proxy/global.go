package resticProxy

import (
	"context"
	"fmt"
	"github.com/kubackup/kubackup/internal/backend/hwobs"
	"github.com/kubackup/kubackup/internal/backend/txcos"
	"github.com/kubackup/kubackup/internal/consts/system_status"
	repoModel "github.com/kubackup/kubackup/internal/entity/v1/repository"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	repositoryDao "github.com/kubackup/kubackup/internal/service/v1/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/azure"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/b2"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/gs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/limiter"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/local"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/location"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/rclone"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/rest"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/retry"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/s3"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/sftp"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/backend/swift"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/cache"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/debug"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/errors"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/fs"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/options"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
	"net/url"
	"path/filepath"
	"sync"
	"time"
)

// TimeFormat is the format used for all timestamps printed by restic.
const TimeFormat = "2006-01-02 15:04:05"

var version = "0.16.5"

type backendWrapper func(r restic.Backend) (restic.Backend, error)

// GlobalOptions hold all global options for restic.
type GlobalOptions struct {
	ctx           context.Context
	Repo          string
	KeyHint       string
	Quiet         bool
	Verbose       int
	NoLock        bool
	RetryLock     time.Duration
	JSON          bool
	CacheDir      string
	NoCache       bool
	CleanupCache  bool
	Compression   repository.CompressionMode
	PackSize      uint
	NoExtraVerify bool

	backend.TransportOptions
	limiter.Limits

	// AWS_ACCESS_KEY_ID
	KeyId string
	// AWS_SECRET_ACCESS_KEY
	Secret string
	// AWS_DEFAULT_REGION
	Region string
	// GOOGLE_PROJECT_ID
	ProjectID string
	// AZURE_ACCOUNT_NAME
	AccountName string
	// AZURE_ACCOUNT_KEY，B2_ACCOUNT_KEY
	AccountKey string
	// B2_ACCOUNT_ID
	AccountID string
	password  string

	backends                              *location.Registry
	backendTestHook, backendInnerTestHook backendWrapper

	// verbosity is set as follows:
	//  0 means: don't print any messages except errors, this is used when --quiet is specified
	//  1 is the default: print essential messages
	//  2 means: print more messages, report minor things, this is used when --verbose is specified
	//  3 means: print very detailed debug messages, this is used when --verbose=2 is specified
	verbosity uint

	Options []string

	extended options.Options

	RepositoryVersion string
}

type Repository struct {
	repoId   int
	repoName string
	repo     *repository.Repository
	cancel   context.CancelFunc
	gopts    GlobalOptions
}

var repositoryService repositoryDao.Service

// GetGlobalOptions 获取仓库配置
func GetGlobalOptions(rep repoModel.Repository) (GlobalOptions, context.CancelFunc) {
	var types string
	switch rep.Type {
	case repoModel.S3:
		types = "s3:"
	case repoModel.Alioos:
		types = "s3:"
	case repoModel.Sftp:
		types = "sftp:"
	case repoModel.Rest:
		types = "rest:"
	case repoModel.HwObs:
		types = "obs:"
	case repoModel.TxCos:
		types = "cos:"
	case repoModel.Local:
		types = ""
	default:
		types = ""
	}
	var repo string
	if rep.Type == repoModel.Rest {
		endpoint, err := url.Parse(rep.Endpoint)
		if err != nil {
			server.Logger().Error(err)
			return GlobalOptions{}, nil
		}
		repo = types + endpoint.Scheme + "://" + rep.KeyId + ":" + rep.Secret + "@" + endpoint.Host + endpoint.Path
	} else {
		repo = types + rep.Endpoint + "/" + rep.Bucket
	}
	var globalOptions = GlobalOptions{
		Repo:              repo,
		KeyId:             rep.KeyId,
		Secret:            rep.Secret,
		Region:            rep.Region,
		CleanupCache:      true,
		Compression:       repository.CompressionOff, //压缩模式
		PackSize:          0,
		NoExtraVerify:     false,
		ProjectID:         rep.ProjectID,
		AccountName:       rep.AccountName,
		AccountKey:        rep.AccountKey,
		AccountID:         rep.AccountID,
		password:          rep.Password,
		RepositoryVersion: rep.RepositoryVersion,
		CacheDir:          server.Config().Data.CacheDir,
		NoCache:           server.Config().Data.NoCache,
		Options:           []string{},
	}
	if rep.RepositoryVersion == "" {
		globalOptions.RepositoryVersion = "2"
	}
	backends := location.NewRegistry()
	backends.Register(azure.NewFactory())
	backends.Register(b2.NewFactory())
	backends.Register(gs.NewFactory())
	backends.Register(local.NewFactory())
	backends.Register(rclone.NewFactory())
	backends.Register(rest.NewFactory())
	backends.Register(s3.NewFactory())
	backends.Register(sftp.NewFactory())
	backends.Register(swift.NewFactory())
	backends.Register(hwobs.NewFactory())
	backends.Register(txcos.NewFactory())

	globalOptions.backends = backends
	var cancel context.CancelFunc
	globalOptions.ctx, cancel = context.WithCancel(context.Background())
	return globalOptions, cancel
}

var repositoryLock sync.Mutex

type RepositoryHandler struct {
	rep  map[int]Repository
	lock sync.Mutex
}

var Myrepositorys RepositoryHandler

func (rh *RepositoryHandler) Get(key int) Repository {
	rh.lock.Lock()
	defer rh.lock.Unlock()
	return Myrepositorys.rep[key]
}

func (rh *RepositoryHandler) Set(key int, rep Repository) {
	rh.lock.Lock()
	defer rh.lock.Unlock()
	Myrepositorys.rep[key] = rep
}

func (rh *RepositoryHandler) Remove(key int) {
	rh.lock.Lock()
	defer rh.lock.Unlock()
	delete(Myrepositorys.rep, key)
}

func cleanCtx() {
	for _, rep := range Myrepositorys.rep {
		rep.cancel()
	}
}

func InitRepository() {
	repositoryLock.Lock()
	defer repositoryLock.Unlock()
	server.UpdateSystemStatus(system_status.Loading)
	defer server.UpdateSystemStatus(system_status.Normal)
	reps, err := repositoryService.List(0, "", common.DBOptions{})
	if err != nil && err.Error() != "not found" {
		fmt.Printf("仓库查询失败，%v\n", err)
		return
	}
	cleanCtx()
	ctx := context.Background()
	Myrepositorys = RepositoryHandler{rep: make(map[int]Repository)}
	for _, rep := range reps {
		option, cancel := GetGlobalOptions(rep)
		openRepository, err1 := OpenRepository(ctx, option)
		if err1 != nil {
			fmt.Printf("仓库加载失败：%v\n", err1)
			continue
		}
		repoa := Repository{
			repoId:   rep.Id,
			repoName: rep.Name,
			repo:     openRepository,
			cancel:   cancel,
			gopts:    option,
		}
		err = openRepository.LoadIndex(option.ctx, nil)
		if err != nil {
			fmt.Printf("仓库%s加载索引失败：%v\n", rep.Name, err)
			continue
		}
		Myrepositorys.Set(rep.Id, repoa)
	}
	go GetAllRepoStats()
	fmt.Println("仓库加载完毕！")
}

// GetRepository 获取仓库操作对象
func GetRepository(repoid int) (*Repository, error) {
	if repoid <= 0 {
		return nil, errors.Errorf("仓库id不能为空")
	}
	myrepository := Myrepositorys.rep[repoid]
	if myrepository.repo == nil {
		return nil, fmt.Errorf("仓库不存在！")
	} else {
		return &myrepository, nil
	}
}

func init() {
	repositoryService = repositoryDao.GetService()
}

func ReadRepo(opts GlobalOptions) (string, error) {
	if opts.Repo == "" {
		return "", errors.Errorf("Please specify repository location (-r or --repository-file)")
	}
	repo := opts.Repo
	return repo, nil
}

const maxKeys = 20

// OpenRepository reads the password and opens the repository.
func OpenRepository(ctx context.Context, opts GlobalOptions) (*repository.Repository, error) {
	repo, err := ReadRepo(opts)
	if err != nil {
		return nil, err
	}

	be, err := open(ctx, repo, opts, opts.extended)
	if err != nil {
		return nil, err
	}

	report := func(msg string, err error, d time.Duration) {
		fmt.Printf("%v returned error, retrying after %v: %v\n", msg, d, err)
	}
	success := func(msg string, retries int) {
		fmt.Printf("%v operation successful after %d retries\n", msg, retries)
	}
	be = retry.New(be, 10, report, success)

	// wrap backend if a test specified a hook
	if opts.backendTestHook != nil {
		be, err = opts.backendTestHook(be)
		if err != nil {
			return nil, err
		}
	}

	s, err := repository.New(be, repository.Options{
		Compression:   opts.Compression,
		PackSize:      opts.PackSize * 1024 * 1024,
		NoExtraVerify: opts.NoExtraVerify,
	})
	if err != nil {
		return nil, errors.Fatal(err.Error())
	}

	err = s.SearchKey(opts.ctx, opts.password, maxKeys, opts.KeyHint)
	if err != nil {
		opts.password = ""
		//密码错误
		return nil, errors.Errorf("仓库密码错误")
	}
	id := s.Config().ID
	if len(id) > 8 {
		id = id[:8]
	}
	fmt.Printf("repository %s opened successfully, password is correct\n", id)

	if opts.NoCache {
		return s, nil
	}

	c, err := cache.New(s.Config().ID, opts.CacheDir)
	if err != nil {
		return s, nil
	}

	if c.Created {
		fmt.Printf("created new cache in %v\n", c.Base)
	}

	// start using the cache
	s.UseCache(c)

	oldCacheDirs, err := cache.Old(c.Base)
	if err != nil {
		fmt.Printf("unable to find old cache directories: %v\n", err)
	}

	// nothing more to do if no old cache dirs could be found
	if len(oldCacheDirs) == 0 {
		return s, nil
	}

	// cleanup old cache dirs if instructed to do so
	if opts.CleanupCache {
		fmt.Printf("removing %d old cache dirs from %v\n", len(oldCacheDirs), c.Base)

		for _, item := range oldCacheDirs {
			dir := filepath.Join(c.Base, item.Name())
			err = fs.RemoveAll(dir)
			if err != nil {
				fmt.Printf("unable to remove %v: %v\n", dir, err)
			}
		}
	} else {
		fmt.Printf("found %d old cache directories in %v, run `restic cache --cleanup` to remove them\n",
			len(oldCacheDirs), c.Base)
	}

	return s, nil
}

// parseConfig 配置各个后端特有参数
func parseConfig(loc location.Location, gopts GlobalOptions, opts options.Options) (interface{}, error) {
	// only apply options for a particular backend here
	opts = opts.Extract(loc.Scheme)

	switch loc.Scheme {
	case "local":
		cfg := loc.Config.(*local.Config)
		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}
		debug.Log("opening local repository at %#v", cfg)
		return cfg, nil

	case "sftp":
		cfg := loc.Config.(*sftp.Config)
		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening sftp repository at %#v", cfg)
		return cfg, nil

	case "s3":
		cfg := loc.Config.(*s3.Config)
		if cfg.KeyID == "" {
			cfg.KeyID = gopts.KeyId
		}

		if cfg.Secret.String() == "" {
			cfg.Secret = options.NewSecretString(gopts.Secret)
		}

		if cfg.KeyID == "" && cfg.Secret.String() != "" {
			return nil, errors.Fatalf("unable to open S3 backend: Key ID (KeyId) is empty")
		} else if cfg.KeyID != "" && cfg.Secret.String() == "" {
			return nil, errors.Fatalf("unable to open S3 backend: Secret (Secret) is empty")
		}

		if cfg.Region == "" {
			cfg.Region = gopts.Region
		}

		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening s3 repository at %#v", cfg)
		return cfg, nil

	case "gs":
		cfg := loc.Config.(*gs.Config)
		if cfg.ProjectID == "" {
			cfg.ProjectID = gopts.ProjectID
		}

		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening gs repository at %#v", cfg)
		return cfg, nil

	case "azure":
		cfg := loc.Config.(*azure.Config)
		if cfg.AccountName == "" {
			cfg.AccountName = gopts.AccountName
		}

		if cfg.AccountKey.String() == "" {
			cfg.AccountKey = options.NewSecretString(gopts.AccountKey)
		}

		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening gs repository at %#v", cfg)
		return cfg, nil

	case "swift":
		cfg := loc.Config.(*swift.Config)

		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening swift repository at %#v", cfg)
		return cfg, nil

	case "b2":
		cfg := loc.Config.(*b2.Config)

		if cfg.AccountID == "" {
			cfg.AccountID = gopts.AccountID
		}

		if cfg.AccountID == "" {
			return nil, errors.Fatalf("unable to open B2 backend: Account ID (AccountID) is empty")
		}

		if cfg.Key.String() == "" {
			cfg.Key = options.NewSecretString(gopts.AccountKey)
		}

		if cfg.Key.String() == "" {
			return nil, errors.Fatalf("unable to open B2 backend: Key (AccountKey) is empty")
		}

		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening b2 repository at %#v", cfg)
		return cfg, nil
	case "rest":
		cfg := loc.Config.(*rest.Config)
		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening rest repository at %#v", cfg)
		return cfg, nil
	case "rclone":
		cfg := loc.Config.(*rclone.Config)
		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening rest repository at %#v", cfg)
		return cfg, nil
	case "obs":
		cfg := loc.Config.(*hwobs.Config)
		if cfg.Ak == "" {
			cfg.Ak = gopts.KeyId
		}

		if cfg.Sk == "" {
			cfg.Sk = gopts.Secret
		}

		if cfg.Ak == "" {
			return nil, errors.Fatalf("unable to open OBS backend: Ak is empty")
		}
		if cfg.Sk == "" {
			return nil, errors.Fatalf("unable to open OBS backend: Sk is empty")
		}

		cfg.SslEnable = gopts.InsecureTLS

		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening OBS repository at %#v", cfg)
		return cfg, nil
	case "cos":
		cfg := loc.Config.(*txcos.Config)
		if cfg.SecretID == "" {
			cfg.SecretID = gopts.KeyId
		}

		if cfg.SecretKey == "" {
			cfg.SecretKey = gopts.Secret
		}

		if cfg.SecretID == "" {
			return nil, errors.Fatalf("unable to open COS backend: SecretID is empty")
		}
		if cfg.SecretKey == "" {
			return nil, errors.Fatalf("unable to open COS backend: SecretKey is empty")
		}

		cfg.EnableCRC = gopts.InsecureTLS

		if err := opts.Apply(loc.Scheme, cfg); err != nil {
			return nil, err
		}

		debug.Log("opening COS repository at %#v", cfg)
		return cfg, nil
	}

	return nil, errors.Fatalf("invalid backend: %q", loc.Scheme)
}

// Open the backend specified by a location config.
func open(ctx context.Context, s string, gopts GlobalOptions, opts options.Options) (restic.Backend, error) {
	debug.Log("parsing location %v", location.StripPassword(gopts.backends, s))
	loc, err := location.Parse(gopts.backends, s)
	if err != nil {

		return nil, errors.Fatalf("parsing repository location failed: %v", err)
	}

	var be restic.Backend

	cfg, err := parseConfig(loc, gopts, opts)
	if err != nil {
		return nil, err
	}

	rt, err := backend.Transport(gopts.TransportOptions)
	if err != nil {
		return nil, errors.Fatal(err.Error())
	}

	lim := limiter.NewStaticLimiter(gopts.Limits)
	rt = lim.Transport(rt)

	factory := gopts.backends.Lookup(loc.Scheme)
	if factory == nil {
		return nil, errors.Fatalf("invalid backend: %q", loc.Scheme)
	}

	be, err = factory.Open(ctx, cfg, rt, lim)
	if err != nil {
		return nil, errors.Fatalf("unable to open repository at %v: %v", location.StripPassword(gopts.backends, s), err)
	}

	// wrap backend if a test specified an inner hook
	if gopts.backendInnerTestHook != nil {
		be, err = gopts.backendInnerTestHook(be)
		if err != nil {
			return nil, err
		}
	}

	// check if config is there
	fi, err := be.Stat(gopts.ctx, restic.Handle{Type: restic.ConfigFile})
	if err != nil {
		return nil, errors.Fatalf("unable to open config file: %v\nIs there a repository at the following location?\n%v", err, location.StripPassword(gopts.backends, s))
	}

	if fi.Size == 0 {
		return nil, errors.New("config file has zero size, invalid repository?")
	}

	return be, nil
}

// Create the backend specified by URI.
func create(ctx context.Context, s string, gopts GlobalOptions, opts options.Options) (restic.Backend, error) {
	debug.Log("parsing location %v", s)
	loc, err := location.Parse(gopts.backends, s)
	if err != nil {
		return nil, err
	}

	cfg, err := parseConfig(loc, gopts, opts)
	if err != nil {
		return nil, err
	}

	rt, err := backend.Transport(gopts.TransportOptions)
	if err != nil {
		return nil, errors.Fatal(err.Error())
	}

	factory := gopts.backends.Lookup(loc.Scheme)
	if factory == nil {
		return nil, errors.Fatalf("invalid backend: %q", loc.Scheme)
	}

	be, err := factory.Create(ctx, cfg, rt, nil)
	if err != nil {
		return nil, err
	}

	return be, nil
}
