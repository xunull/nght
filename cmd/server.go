package cmd

import (
	"context"
	"github.com/xunull/nght/internal/fiber_server"
	"github.com/xunull/nght/internal/gin_server"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	EchoHostname bool
	ServerType   string
	ResponseJson bool
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start gin web server",
	Long:  `start gin web server`,
	Run: func(cmd *cobra.Command, args []string) {

		if ServerType == "gin" {
			go gin_server.StartWeb(Port)
			mainSigChan := make(chan os.Signal, 1)
			signal.Notify(mainSigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
			<-mainSigChan

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := gin_server.HttpServer.Shutdown(ctx); err != nil {
				log.Fatal("Server Shutdown error:", err)
			}
			log.Println("Server exit")
		} else if ServerType == "fiber" {
			fiber_server.Server(Port)
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
}
