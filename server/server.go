package server

import (
	"errors"
	"flag"
	"github.com/emoss08/trenova/tools"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/routes"
	"github.com/rs/cors"
	"github.com/wader/gormstore/v2"
	"golang.org/x/net/context"
	"gorm.io/gorm"
)

var store *gormstore.Store

func SetupAndRun(db *gorm.DB) {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server will gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	// Initialize Gob
	tools.RegisterGob()

	// Initialize session store
	store = gormstore.New(db, []byte(os.Getenv("SESSION_KEY")))
	quit := make(chan struct{})

	// Periodic cleanup of expired sessions
	go store.PeriodicCleanup(1*time.Hour, quit)

	if store == nil {
		log.Fatal("Session store could not be initialized")
	}

	// Initialize router
	r := routes.InitializeRouter(db, store)

	mux := http.NewServeMux()
	mux.Handle("/", r)

	// Apply CORS middleware allowing all origins
	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"}, // Specify the allowed origin
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"}, // Allowing all headers
		AllowCredentials: true,
	}).Handler(mux)

	// Server Configuration
	srv := &http.Server{
		Handler:      handler,
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

	log.Printf("Server is ready to handle requests at https://%s", srv.Addr)

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
