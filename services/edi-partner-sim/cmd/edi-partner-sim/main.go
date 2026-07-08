package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
)

func main() {
	os.Exit(run())
}

func run() int {
	listen := flag.String("listen", ":9210", "address the simulator listens on")
	as2ID := flag.String("as2-id", "SIMPARTNER", "the simulator's AS2 identifier")
	remoteAS2ID := flag.String("remote-as2-id", "TRENOVA", "the Trenova AS2 identifier")
	trenovaInbound := flag.String(
		"trenova-inbound",
		"http://localhost:3001/api/v1/edi/as2/inbound/",
		"Trenova AS2 inbound receiver URL",
	)
	autoAck := flag.Bool("auto-ack", true, "automatically send a 997 for received documents")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	server, err := sim.NewServer(sim.Options{
		AS2ID:           *as2ID,
		RemoteAS2ID:     *remoteAS2ID,
		TrenovaInbound:  *trenovaInbound,
		AutoAcknowledge: *autoAck,
		Logger:          logger,
	})
	if err != nil {
		logger.Error("failed to initialize simulator", "error", err)
		return 1
	}

	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("EDI partner simulator listening",
			"address", *listen,
			"as2Id", *as2ID,
			"remoteAs2Id", *remoteAS2ID,
			"trenovaInbound", *trenovaInbound,
			"autoAck", *autoAck,
		)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("simulator server failed", "error", err)
			return 1
		}
	case <-shutdown:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error("simulator shutdown failed", "error", err)
			return 1
		}
	}
	return 0
}
