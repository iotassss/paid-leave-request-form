package main

import (
	"io"
	"os"

	"github.com/iotassss/paid-leave-request-form/config"
	"github.com/iotassss/paid-leave-request-form/internal/controller/leaveController"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	//ログ設定
	f, _ := os.Create(config.Config.LogFile)
	gin.DefaultWriter = io.MultiWriter(os.Stdout, f)

	//templateディレクトリ設定
	r := gin.Default()
	r.LoadHTMLGlob("template/**/*")

	r.GET("/leave/form", leaveController.Form)
	r.POST("/leave/generate", leaveController.Generate)

	return r
}

func main() {

	r := setupRouter()
	r.Run(":8080")
}
