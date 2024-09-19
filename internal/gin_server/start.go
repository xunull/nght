package gin_server

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Serve(port int) {
	go StartWeb(port)
	mainSigChan := make(chan os.Signal, 1)
	signal.Notify(mainSigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	<-mainSigChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := HttpServer.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown error:", err)
	}
	log.Println("Server exit")
}

func StartWeb(port int) {
	ginServer = gin.Default()

	AddRoute()

	HttpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        ginServer,
		MaxHeaderBytes: 1 << 20,
	}
	log.Println("start gin server: ", HttpServer.Addr)
	if err := HttpServer.ListenAndServe(); err != nil {
		log.Println("start gin server error!")
		log.Fatal(err)
	}
}
