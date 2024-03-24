package server

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/tools"
	"github.com/fatih/color"

	"github.com/emoss08/trenova/routes"
	"github.com/gorilla/handlers"
	"golang.org/x/net/context"
)

var trenovaText = `
   ______                   
  /_  __/_____ ___   ____   ____  _   __ ____ _
   / /  / ___// _ \ / __ \ / __ \| | / // __  /
  / /  / /   /  __// / / // /_/ /| |/ // /_/ / 
 /_/  /_/    \___//_/ /_/ \____/ |___/ \__,_/ %s`

func SetupAndRun() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server will gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	// Initialize Gob
	tools.RegisterGob()

	// Initialize router
	r := routes.InitializeRouter()

	mux := http.NewServeMux()
	mux.Handle("/", r)

	// Server Configuration
	srv := &http.Server{
		Handler:      handlers.CompressHandler(mux),
		Addr:         "127.0.0.1:3000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	printStartupMessage(srv.Addr)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Panicf("Server Shutdown Failed:%+v", err)
	}

	log.Println("Server exiting")
}

func printStartupMessage(addr string) {
	version := os.Getenv("TRENOVA_VERSION")

	color.Cyan(trenovaText, version)
	color.Cyan("-----------------------------------------------------\n")
	color.Cyan("Listening on: http://" + addr)
}
