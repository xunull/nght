package gin_server

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func StartWeb(port string) {
	ginServer = gin.Default()

	AddRoute()

	HttpServer = &http.Server{
		Addr:           ":" + port,
		Handler:        ginServer,
		MaxHeaderBytes: 1 << 20,
	}
	log.Println("start gin server: ", HttpServer.Addr)
	if err := HttpServer.ListenAndServe(); err != nil {
		log.Println("start gin server error!")
		log.Fatal(err)
	}
}
