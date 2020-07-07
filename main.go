package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	Port       string
	RemoteHost string
)

func StartWeb() {
	ginServer = gin.Default()

	AddRoute()

	httpServer = &http.Server{
		Addr:           ":" + Port,
		Handler:        ginServer,
		MaxHeaderBytes: 1 << 20,
	}
	log.Println("start gin server: ", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Println("start gin server error!")
		log.Fatal(err)
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "nght",
		Short: "a gin web server for nginx http test",
		Long:  `a gin web server for nginx http test`,
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "start gin web server",
		Long:  "start gin web server",
		Run: func(cmd *cobra.Command, args []string) {
			go StartWeb()
			mainSigChan := make(chan os.Signal, 1)
			signal.Notify(mainSigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
			<-mainSigChan

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(ctx); err != nil {
				log.Fatal("Server Shutdown error:", err)
			}
			log.Println("Server exit")
		},
	}

	var clientCmd = &cobra.Command{
		Use:   "client",
		Short: "start test client",
		Long:  "start test client",
		Run: func(cmd *cobra.Command, args []string) {
			// todo
			log.Println(RemoteHost)
		},
	}
	clientCmd.Flags().StringVar(&RemoteHost, "host", "127.0.0.1", "remote server addr")
	rootCmd.AddCommand(serverCmd, clientCmd)
	rootCmd.PersistentFlags().StringVarP(&Port, "port", "p", "8080", "the port")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

}
