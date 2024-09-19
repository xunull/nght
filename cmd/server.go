package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xunull/nght/internal/fiber_server"
	"github.com/xunull/nght/internal/gin_server"
	"github.com/xunull/nght/internal/global"
	"log"
)

var (
	EchoHostname bool
	AppName      string
	ServerType   string
	ResponseJson bool
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start web server",
	Long:  `start web server`,
	Run: func(cmd *cobra.Command, args []string) {

		global.SetAppName(AppName)

		if ServerType == "gin" {
			gin_server.Serve(Port)
		} else if ServerType == "fiber" {
			fiber_server.Serve(Port)
		} else {
			log.Fatal("server type not support")
		}

	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().BoolVar(&EchoHostname, "echo-hostname", false, "echo host name")
	serverCmd.PersistentFlags().StringVarP(&ServerType, "type", "t", "gin", "server type")
	serverCmd.PersistentFlags().BoolVar(&ResponseJson, "response-json", false, "response json")
	serverCmd.PersistentFlags().StringVar(&AppName, "app-name", "nght", "app name")
}
