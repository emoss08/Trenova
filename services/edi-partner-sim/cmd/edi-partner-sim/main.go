package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	identityDir := flag.String(
		"identity-dir",
		"",
		"directory to persist the AS2 keypair and SFTP host key across restarts (empty = ephemeral)",
	)
	sftpListen := flag.String(
		"sftp-listen",
		":9222",
		"address the SFTP mailbox listens on (empty to disable)",
	)
	sftpUser := flag.String("sftp-user", "trenova", "SFTP username Trenova authenticates with")
	sftpPassword := flag.String(
		"sftp-password",
		"trenova-sim",
		"SFTP password Trenova authenticates with",
	)
	sftpRoot := flag.String("sftp-root", "", "SFTP mailbox root directory (empty = temp dir)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var sftpServer *sim.SFTPServer
	if strings.TrimSpace(*sftpListen) != "" {
		created, err := sim.NewSFTPServer(sim.SFTPOptions{
			Listen:      *sftpListen,
			Username:    *sftpUser,
			Password:    *sftpPassword,
			RootDir:     *sftpRoot,
			IdentityDir: *identityDir,
			Logger:      logger,
		})
		if err != nil {
			logger.Error("failed to initialize SFTP mailbox", "error", err)
			return 1
		}
		sftpServer = created
		go func() {
			logger.Info("SFTP mailbox listening",
				"address", sftpServer.Addr(),
				"username", *sftpUser,
				"inbound", sftpServer.InboundDir(),
				"outbound", sftpServer.OutboundDir(),
			)
			if err := sftpServer.Serve(); err != nil {
				logger.Error("SFTP mailbox failed", "error", err)
			}
		}()
	}

	server, err := sim.NewServer(sim.Options{
		AS2ID:           *as2ID,
		RemoteAS2ID:     *remoteAS2ID,
		TrenovaInbound:  *trenovaInbound,
		AutoAcknowledge: *autoAck,
		IdentityDir:     *identityDir,
		SFTP:            sftpServer,
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
