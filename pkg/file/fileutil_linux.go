//go:build linux
// +build linux

package fileutil

import (
	"github.com/kubackup/kubackup/internal/consts"
	"github.com/kubackup/kubackup/internal/model"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func ReplaceHomeDir(path string) string {
	if strings.HasPrefix(FixPath(path), "~") {
		return strings.Replace(path, "~", HomeDir(), -1)
	}
	return path
}

func Exist(name string) bool {
	_, err := os.Stat(FixPath(name))
	if !os.IsNotExist(err) {
		return true
	}
	return false
}

func Mkdir(path string, mode os.FileMode) bool {
	err := os.Mkdir(path, mode)
	if err != nil {
		return false
	}
	return true
}

func HomeDir() string {
	return os.Getenv("HOME")
}

func FixPath(path string) string {
	osType := runtime.GOOS
	if osType == "windows" {
		return strings.ReplaceAll(path, "/", "\\")
	}
	return path
}

func ListDir(path string) ([]*model.FileInfo, error) {
	dirs, err := ioutil.ReadDir(FixPath(path))
	if err != nil {
		return nil, err
	}
	var files []*model.FileInfo
	osType := runtime.GOOS
	for _, dir := range dirs {
		var ct time.Time
		var statT *syscall.Stat_t
		if osType == "linux" {
			statT = dir.Sys().(*syscall.Stat_t)
			ct = time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec)
			sepa := ""
			if !strings.HasSuffix(path, string(filepath.Separator)) {
				sepa = string(filepath.Separator)
			}
			f := &model.FileInfo{
				Name:       dir.Name(),
				Path:       path + sepa + dir.Name(),
				IsDir:      dir.IsDir(),
				Mode:       dir.Mode().String(),
				ModTime:    dir.ModTime().Format(consts.Custom),
				Size:       dir.Size(),
				Gid:        int(statT.Gid),
				Uid:        int(statT.Uid),
				CreateTime: ct.Format(consts.Custom),
			}
			files = append(files, f)
		}
	}
	return files, nil

}

// GetFilePath 获取文件路径
func GetFilePath(file string) string {
	return filepath.Dir(file)
}