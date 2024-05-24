package resticProxy

import (
	"github.com/fanjindong/go-cache"
	"github.com/kubackup/kubackup/internal/consts"
	"github.com/kubackup/kubackup/internal/server"
	"strconv"
	"time"
)

// BackupLock 备份任务锁
func BackupLock(repo int, path string) bool {
	key := consts.Key("BackupIsRun", strconv.Itoa(repo), path)
	c := server.Cache()
	res, ok := c.Get(key)
	if ok && res == 1 {
		return false
	} else {
		c.Set(key, 1, cache.WithEx(24*time.Hour))
		return true
	}
}

func BackupUnLock(repo int, path string) {
	key := consts.Key("BackupIsRun", strconv.Itoa(repo), path)
	c := server.Cache()
	c.Del(key)
}
