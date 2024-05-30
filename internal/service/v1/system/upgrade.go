package system

import (
	"fmt"
	"github.com/kubackup/kubackup/internal/consts/global"
	"github.com/kubackup/kubackup/internal/consts/system_status"
	"github.com/kubackup/kubackup/internal/server"
	fileutil "github.com/kubackup/kubackup/pkg/file"
	"github.com/kubackup/kubackup/pkg/utils/cmd"
	"github.com/kubackup/kubackup/pkg/utils/http"
	"os"
	"path"
	"runtime"
	"time"
)

func Upgrade(version string) error {
	if version == "" {
		return fmt.Errorf("版本号不能为空")
	}
	timeStr := time.Now().Format("200601021504")
	upgradeDir := path.Join(server.Config().Data.CacheDir, fmt.Sprintf("upgrade/upgrade_%s/downloads", timeStr))
	if err := os.MkdirAll(upgradeDir, os.ModePerm); err != nil {
		return err
	}
	downloadUrl := fmt.Sprintf("%s/%s/kubackup_server_%s_%s_%s", global.DownlodUrl, version, version, runtime.GOOS, runtime.GOARCH)
	server.UpdateSystemStatus(system_status.Upgrade)
	go func() {
		err := http.DownloadFile(downloadUrl, path.Join(upgradeDir, "kubackup_server"))
		if err != nil {
			server.Logger().Errorf("kubackup_server文件下载失败，错误：%v", err)
			server.UpdateSystemStatus(system_status.Normal)
			return
		}
		if runtime.GOOS == "linux" {
			err = http.DownloadFile(global.ServiceFileUrl, path.Join(upgradeDir, "kubackup.service"))
			if err != nil {
				server.Logger().Errorf("kubackup_server文件下载失败，错误：%v", err)
				server.UpdateSystemStatus(system_status.Normal)
				return
			}
		}
		server.Logger().Println("所有文件下载成功")
		defer func() {
			_ = os.Remove(upgradeDir)
		}()
		err = fileutil.CopyFile(path.Join(upgradeDir, "kubackup_server"), path.Join("/usr/local/bin", "kubackup_server"))
		if err != nil {
			server.Logger().Errorf("kubackup_server更新失败，错误：%v", err)
			server.UpdateSystemStatus(system_status.Normal)
			return
		}
		if runtime.GOOS == "linux" {
			err = fileutil.CopyFile(path.Join(upgradeDir, "kubackup.service"), path.Join("/etc/systemd/system", "kubackup.service"))
			if err != nil {
				server.Logger().Errorf("kubackup.service更新失败，错误：%v", err)
				server.UpdateSystemStatus(system_status.Normal)
				return
			}
		}
		server.Logger().Println("更新成功")
		_, _ = cmd.ExecWithTimeOut("chmod +x "+path.Join("/usr/local/bin", "kubackup_server"), 1*time.Minute)
		if runtime.GOOS == "linux" {
			_, _ = cmd.ExecWithTimeOut("systemctl daemon-reload && systemctl restart kubackup.service", 2*time.Minute)
		}
		server.UpdateSystemStatus(system_status.Normal)
	}()
	return nil
}
