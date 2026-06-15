package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xunull/nght/internal/fiber_server"
	"github.com/xunull/nght/internal/gin_server"
	"github.com/xunull/nght/internal/global"
	"log"
)

var (
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

		switch ServerType {
		case "gin":
			gin_server.Serve(Port)
		case "fiber":
			fiber_server.SetResponseJson(ResponseJson)
			fiber_server.Serve(Port)
		default:
			log.Fatal("server type not support")
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().StringVarP(&ServerType, "type", "t", "gin", "server type")
	serverCmd.PersistentFlags().BoolVar(&ResponseJson, "response-json", false, "response json (fiber only)")
	serverCmd.PersistentFlags().StringVar(&AppName, "app-name", "nght", "app name")
}
