package main

import (
	"backend/db"
	"backend/router"
	"log"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

func main() {
	// Read the config file
	viper.SetConfigFile("./config/envs/dev.env")
	viper.ReadInConfig()
	dbUrl := viper.Get("DB_URL").(string)

	// Initialize the database
	db.Init(dbUrl)

	// Initialize the router
	r := router.InitRouter()

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Prepare the server start message with colors and emoji
	startMsg := color.New(color.FgGreen).SprintfFunc()
	log.Println(startMsg("🚀 Server starting on %s", srv.Addr))

	// Start the server and log if there's an error
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %s", err)
	}
}
