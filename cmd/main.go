package main

import (
	"embed"
	"fmt"
	"github.com/kubackup/kubackup"
	"github.com/kubackup/kubackup/internal/cmdServer"
	"github.com/kubackup/kubackup/internal/route"
	"github.com/kubackup/kubackup/internal/server"
	"github.com/spf13/cobra"
	"os"
	"runtime"
)

var (
	configPath     string
	serverBindHost string
	serverBindPort int
)

//go:embed web/dashboard
var embedWebDashboard embed.FS

var rootCmd = &cobra.Command{
	Use:   "kubackup_server",
	Short: "酷备份-KuBackup 是一个文件备份服务",
	Long:  `酷备份-KuBackup 简单、开源、快速、安全的 服务器文件备份工具`,
	Run: func(cmd *cobra.Command, args []string) {
		server.EmbedWebDashboard = embedWebDashboard
		err := server.Listen(route.InitRoute, serverBindHost, serverBindPort, configPath)
		if err != nil {
			os.Exit(1)
			return
		}
	},
	Version: fmt.Sprintf("%s compiled with %v on %v/%v at %s",
		v.Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, v.BuildTime),
}
var v = kubackup.GetVersion()

var resetOtpCmd = &cobra.Command{
	Use:   "resetOtp [username]",
	Short: "关闭用户otp认证",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := ""
		if len(args) > 0 {
			username = args[0]
		} else {
			fmt.Println("用户名不能为空")
			return
		}
		fmt.Println(fmt.Sprintf("正在关闭：%s 的二次认证", username))
		cmdServer.Instance(configPath).ClearOtp(username, 0)
	},
}

var resetPwdCmd = &cobra.Command{
	Use:   "resetPwd [username]",
	Short: "重置用户密码",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := ""
		if len(args) > 0 {
			username = args[0]
		} else {
			fmt.Println("用户名不能为空")
			return
		}
		fmt.Println(fmt.Sprintf("正在重置：%s 的密码", username))
		cmdServer.Instance(configPath).ClearPwd(username, 0)
	},
}

func init() {
	rootCmd.Flags().StringVar(&serverBindHost, "server-bind-host", "", "bind address")
	rootCmd.Flags().IntVarP(&serverBindPort, "server-bind-port", "p", 0, "bind port")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config-path", "c", "", "config file path")
	rootCmd.AddCommand(resetOtpCmd)
	rootCmd.AddCommand(resetPwdCmd)
}
func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
		return
	}
}
