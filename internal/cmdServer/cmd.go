package cmdServer

import (
	"fmt"
	"github.com/asdine/storm/v3"
	cf "github.com/kubackup/kubackup/internal/config"
	"github.com/kubackup/kubackup/internal/entity/v1/config"
	"github.com/kubackup/kubackup/internal/service/v1/common"
	"github.com/kubackup/kubackup/internal/service/v1/user"
	fileutil "github.com/kubackup/kubackup/pkg/file"
	shell "github.com/kubackup/kubackup/pkg/utils/cmd"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

type Cmd struct {
	config *config.Config
	db     *storm.DB
}

func Instance(path string) *Cmd {
	cmd := &Cmd{}
	cmd.setConfig(path)
	cmd.setUpDB()
	return cmd
}

func (cmd *Cmd) setConfig(path string) {
	c, err := cf.ReadConfig(path)
	if err != nil {
		panic(err)
	}
	cmd.config = c
}

func (cmd *Cmd) setUpDB() {
	dbpath := fileutil.ReplaceHomeDir(cmd.config.Data.DbDir)
	if !fileutil.Exist(dbpath) {
		fmt.Println("数据库目录不存在")
		return
	}
	d, err := storm.Open(path.Join(dbpath, string(filepath.Separator), "kubackup.db"))
	if err != nil {
		d = nil
	}
	cmd.db = d
}

// ClearOtp 关闭用户二次认证
// username 用户名
// mode 是否启动服务，若服务启动状态，需临时关闭服务，待任务完成后再启动服务
func (cmd *Cmd) ClearOtp(username string, mode int) {
	userService := user.GetService()
	if cmd.db == nil {
		if runtime.GOOS == "linux" {
			_, _ = shell.ExecWithTimeOut("systemctl stop kubackup.service", 1*time.Minute)
			cmd.setUpDB()
			cmd.ClearOtp(username, 1)
		} else {
			fmt.Println("数据库繁忙，请手动关闭kubackup_server后再试")
		}
		return
	}
	err := userService.ClearOtp(username, common.DBOptions{DB: cmd.db})
	if mode == 1 && runtime.GOOS == "linux" {
		_, _ = shell.ExecWithTimeOut("systemctl start kubackup.service", 1*time.Minute)
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("关闭成功。")

}

// ClearPwd 重置用户密码
// username 用户名
// mode 是否启动服务，若服务启动状态，需临时关闭服务，待任务完成后再启动服务
func (cmd *Cmd) ClearPwd(username string, mode int) {
	userService := user.GetService()
	if cmd.db == nil {
		if runtime.GOOS == "linux" {
			_, _ = shell.ExecWithTimeOut("systemctl stop kubackup.service", 1*time.Minute)
			cmd.setUpDB()
			cmd.ClearPwd(username, 1)
		} else {
			fmt.Println("数据库繁忙，请手动关闭kubackup_server后再试")
		}
		return
	}
	err := userService.ClearPwd(username, common.DBOptions{DB: cmd.db})
	if mode == 1 && runtime.GOOS == "linux" {
		_, _ = shell.ExecWithTimeOut("systemctl start kubackup.service", 1*time.Minute)
	}
	if err != nil {
		fmt.Println(err)
		return
	}
}
