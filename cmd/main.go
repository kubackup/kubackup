package main

import (
	"embed"
	"fmt"
	"github.com/kubackup/kubackup"
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
	Short: "酷备份-KuBackup",
	Long:  `酷备份-KuBackup`,
	Run: func(cmd *cobra.Command, args []string) {
		server.EmbedWebDashboard = embedWebDashboard
		err := server.Listen(route.InitRoute, serverBindHost, serverBindPort, configPath)
		if err != nil {
			os.Exit(1)
			return
		}
	},
	Version: fmt.Sprintf("%s compiled with %v on %v/%v at %s",
		v.Verison, runtime.Version(), runtime.GOOS, runtime.GOARCH, v.BuildTime),
}
var v = kubackup.GetVersion()

func init() {
	rootCmd.Flags().StringVar(&serverBindHost, "server-bind-host", "", "bind address")
	rootCmd.Flags().IntVarP(&serverBindPort, "server-bind-port", "p", 0, "bind port")
	rootCmd.Flags().StringVarP(&configPath, "config-path", "c", "", "config file path")
}
func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
		return
	}
}
