package main

import (
	"backend/pkg/common/db"
	"backend/pkg/common/middleware"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {

	viper.SetConfigFile("./pkg/common/envs/.env")
	viper.ReadInConfig()

	dbDsn := viper.Get("DB_DSN").(string)

	fmt.Println(viper.Get("DB_DSN"))

	r := gin.Default()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(gin.ErrorLogger())
	r.Use(middleware.ErrorHandlingMiddleware())

	db.Init(dbDsn)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run()
}
