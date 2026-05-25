package resticProxy

import (
	"context"
	"sync"

	"github.com/kubackup/kubackup/internal/i18n"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/pkg/errors"

	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/debug"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
)

var globalLocks struct {
	unlockers     []*repository.Unlocker
	cancelRefresh chan struct{}
	refreshWG     sync.WaitGroup
	sync.Mutex
}

// 当前语言，默认为英文
var currentLanguage = i18n.DefaultLanguage

// SetCurrentLanguage 设置当前语言
func SetCurrentLanguage(lang string) {
	if lang == i18n.ZH_CN || lang == i18n.EN_US {
		currentLanguage = lang
	}
}

// lockRepo ->  lockRepo 获取到锁
// lockRepo ->  lockRepoExclusive 获取不到锁
// lockRepoExclusive ->  lockRepoExclusive 获取不到锁
// lockRepoExclusive ->  lockRepo 获取不到锁

// 非互斥锁
func lockRepo(ctx context.Context, repo *repository.Repository) (*repository.Unlocker, context.Context, error) {
	return lockRepository(ctx, repo, false)
}

// 互斥锁
func lockRepoExclusive(ctx context.Context, repo *repository.Repository) (*repository.Unlocker, context.Context, error) {
	return lockRepository(ctx, repo, true)
}

func lockRepository(ctx context.Context, repo *repository.Repository, exclusive bool) (*repository.Unlocker, context.Context, error) {
	unlocker, ctx, err := repository.Lock(ctx, repo, exclusive, 0, func(msg string) {}, func(format string, args ...interface{}) {})
	if err != nil {
		// 使用当前语言翻译错误信息
		errorMsg := i18n.T("error.repositoryLocked", currentLanguage)
		return nil, ctx, errors.WithMessage(err, errorMsg)
	}
	debug.Log("create lock (exclusive %v)", exclusive)

	globalLocks.Lock()
	globalLocks.unlockers = append(globalLocks.unlockers, unlocker)
	globalLocks.Unlock()

	return unlocker, ctx, err
}

func unlockRepo(unlocker *repository.Unlocker) {
	if unlocker == nil {
		return
	}

	globalLocks.Lock()
	defer globalLocks.Unlock()

	for i := 0; i < len(globalLocks.unlockers); i++ {
		if unlocker == globalLocks.unlockers[i] {
			// remove the lock from the repo
			debug.Log("unlocking repository")
			unlocker.Unlock()

			// remove the lock from the list of locks
			globalLocks.unlockers = append(globalLocks.unlockers[:i], globalLocks.unlockers[i+1:]...)
			return
		}
	}

	debug.Log("unable to find unlocker in the global list of locks, ignoring")
}

func UnlockRepoById(repoid int, removeAll bool) (uint, error) {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return 0, err
	}
	repo := repoHandler.repo

	if removeAll {
		return repository.RemoveAllLocks(repoHandler.gopts.ctx, repo)
	}
	return repository.RemoveStaleLocks(repoHandler.gopts.ctx, repo)
}

func unlockAll() error {
	globalLocks.Lock()
	defer globalLocks.Unlock()

	debug.Log("unlocking %d locks", len(globalLocks.unlockers))
	for _, unlocker := range globalLocks.unlockers {
		unlocker.Unlock()
		debug.Log("successfully removed lock")
	}
	globalLocks.unlockers = globalLocks.unlockers[:0]

	return nil
}

func GetAllLock() []model.LockInfo {
	res := make([]model.LockInfo, 0)
	// 在 restic 0.18.1 中，锁的信息不再直接暴露
	// 返回空列表
	return res
}
