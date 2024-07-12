package resticProxy

import (
	"context"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/pkg/errors"
	"sync"
	"time"

	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/debug"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/repository"
	"github.com/kubackup/kubackup/pkg/restic_source/rinternal/restic"
)

var globalLocks struct {
	locks         []*restic.Lock
	cancelRefresh chan struct{}
	refreshWG     sync.WaitGroup
	sync.Mutex
}

// lockRepo ->  lockRepo 获取到锁
// lockRepo ->  lockRepoExclusive 获取不到锁
// lockRepoExclusive ->  lockRepoExclusive 获取不到锁
// lockRepoExclusive ->  lockRepo 获取不到锁

// 非互斥锁
func lockRepo(ctx context.Context, repo *repository.Repository) (*restic.Lock, error) {
	return lockRepository(ctx, repo, false)
}

// 互斥锁
func lockRepoExclusive(ctx context.Context, repo *repository.Repository) (*restic.Lock, error) {
	return lockRepository(ctx, repo, true)
}

func lockRepository(ctx context.Context, repo *repository.Repository, exclusive bool) (*restic.Lock, error) {
	lockFn := restic.NewLock
	if exclusive {
		lockFn = restic.NewExclusiveLock
	}

	lock, err := lockFn(ctx, repo)
	if err != nil {
		return nil, errors.WithMessage(err, "仓库当前已被锁定，请等待其他操作完成，若确定无其他任务，可手动执行清除锁，锁信息：")
	}
	debug.Log("create lock %p (exclusive %v)", lock, exclusive)

	globalLocks.Lock()
	if globalLocks.cancelRefresh == nil {
		debug.Log("start goroutine for lock refresh")
		globalLocks.cancelRefresh = make(chan struct{})
		globalLocks.refreshWG = sync.WaitGroup{}
		globalLocks.refreshWG.Add(1)
		go refreshLocks(&globalLocks.refreshWG, globalLocks.cancelRefresh)
	}

	globalLocks.locks = append(globalLocks.locks, lock)
	globalLocks.Unlock()

	return lock, err
}

var refreshInterval = 5 * time.Minute

func refreshLocks(wg *sync.WaitGroup, done <-chan struct{}) {
	debug.Log("start")
	defer func() {
		wg.Done()
		globalLocks.Lock()
		globalLocks.cancelRefresh = nil
		globalLocks.Unlock()
	}()

	ticker := time.NewTicker(refreshInterval)

	for {
		select {
		case <-done:
			debug.Log("terminate")
			return
		case <-ticker.C:
			debug.Log("refreshing locks")
			globalLocks.Lock()
			for _, lock := range globalLocks.locks {
				err := lock.Refresh(context.TODO())
				if err != nil {
					server.Logger().Warnf("unable to refresh lock: %v\n", err)
				}
			}
			globalLocks.Unlock()
		}
	}
}

func unlockRepo(lock *restic.Lock) {
	if lock == nil {
		return
	}

	globalLocks.Lock()
	defer globalLocks.Unlock()

	for i := 0; i < len(globalLocks.locks); i++ {
		if lock == globalLocks.locks[i] {
			// remove the lock from the repo
			debug.Log("unlocking repository with lock %v", lock)
			if err := lock.Unlock(); err != nil {
				debug.Log("error while unlocking: %v", err)
				server.Logger().Warnf("error while unlocking: %v", err)
				return
			}

			// remove the lock from the list of locks
			globalLocks.locks = append(globalLocks.locks[:i], globalLocks.locks[i+1:]...)
			return
		}
	}

	debug.Log("unable to find lock %v in the global list of locks, ignoring", lock)
}

func UnlockRepoById(repoid int, removeAll bool) (uint, error) {
	repoHandler, err := GetRepository(repoid)
	if err != nil {
		return 0, err
	}
	repo := repoHandler.repo

	fn := restic.RemoveStaleLocks
	if removeAll {
		fn = restic.RemoveAllLocks
	}

	locks, err := fn(repoHandler.gopts.ctx, repo)
	if err != nil {
		return 0, err
	}
	return locks, nil
}

func unlockAll() error {
	globalLocks.Lock()
	defer globalLocks.Unlock()

	debug.Log("unlocking %d locks", len(globalLocks.locks))
	for _, lock := range globalLocks.locks {
		if err := lock.Unlock(); err != nil {
			debug.Log("error while unlocking: %v", err)
			return err
		}
		debug.Log("successfully removed lock")
	}
	globalLocks.locks = globalLocks.locks[:0]

	return nil
}

func GetAllLock() []model.LockInfo {
	res := make([]model.LockInfo, 0)
	for _, lock := range globalLocks.locks {
		l := model.LockInfo{
			Time:      lock.Time,
			Exclusive: lock.Exclusive,
			Hostname:  lock.Hostname,
			Username:  lock.Username,
			PID:       lock.PID,
			UID:       lock.UID,
			GID:       lock.GID,
		}
		res = append(res, l)
	}
	return res
}
