package main

import (
	"backend/db"
	"backend/handlers"
	"backend/middleware"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigFile("./config/envs/dev.env")
	viper.ReadInConfig()

	dbUrl := viper.Get("DB_URL").(string)

	db.Init(dbUrl)

	r := mux.NewRouter()

	r.Use(middleware.LoggingMiddleware)

	r.HandleFunc("/test", handlers.TestEndpoint).Methods("POST")

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	srv.ListenAndServe()
}
