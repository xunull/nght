package gin_server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

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
