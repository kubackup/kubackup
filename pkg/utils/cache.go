package utils

import (
	"sync"
	"sync/atomic"
	"time"
)

var globalMap sync.Map
var length int64

func Set(key string, data interface{}, timeout int) {
	globalMap.Store(key, data)
	atomic.AddInt64(&length, 1)
	time.AfterFunc(time.Second*time.Duration(timeout), func() {
		atomic.AddInt64(&length, -1)
		globalMap.Delete(key)
	})
}

func Get(key string) (interface{}, bool) {
	return globalMap.Load(key)
}

func Len() int {
	return int(length)
}
