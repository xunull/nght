package cmd

import (
	"context"
	"github.com/xunull/nght/pkg/gin_server"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "start gin web server",
	Long:  `start gin web server`,
	Run: func(cmd *cobra.Command, args []string) {
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
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

}
