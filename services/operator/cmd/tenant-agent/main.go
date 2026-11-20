package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v5"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/credential/dirstore"
	"github.com/emoss08/trenova/operator/internal/emailevents"
	"github.com/emoss08/trenova/operator/internal/escrow/escrowfile"
	"github.com/emoss08/trenova/operator/internal/hosttelemetry"
	"github.com/emoss08/trenova/operator/internal/runlock"
	"github.com/emoss08/trenova/operator/internal/systememail"
	"github.com/emoss08/trenova/operator/internal/telemetryrelay"
	"github.com/emoss08/trenova/operator/internal/tenantagent"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/usage"
)

const (
	envPrefix      = "TRENOVA_TENANT_AGENT_"
	requestTimeout = 30 * time.Second
	startRetry     = 5 * time.Minute
)

type config struct {
	stateDir    string
	enrolment   string
	secretsDir  string
	tmsURL      string
	tokenFile   string
	dataPath    string
	version     string
	healthAddr  string
	poll        time.Duration
	usageEvery  time.Duration
	logLevel    slog.Level
	once        bool
	healthcheck bool
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := load(os.Args[1:])
	if err != nil {
		logger.Error("the tenant agent's configuration", "error", err)
		os.Exit(2)
	}
	if cfg.healthcheck {
		os.Exit(probe(cfg.healthAddr))
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.logLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err = run(ctx, &cfg, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("the tenant agent stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config, logger *slog.Logger) error {
	lock, err := runlock.Acquire(cfg.stateDir)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Release() }()

	store, err := dirstore.New(dirstore.Params{
		Dir:       filepath.Join(cfg.stateDir, "identity"),
		Enrolment: cfg.enrolment,
	})
	if err != nil {
		return err
	}
	transport := &credential.Transport{}
	cp := controlplane.NewLazy(&http.Client{Transport: transport, Timeout: requestTimeout})
	creds := credential.NewManager(&credential.ManagerParams{
		Store:     store,
		Transport: transport,
		Renewer:   cp,
		Logger:    logger,
	})
	cp.Locate(func() string { return creds.Current().APIURL })

	tmsClient, err := tms.New(tms.Params{URL: cfg.tmsURL, Tokens: tms.NewFileTokens(cfg.tokenFile)})
	if err != nil {
		return err
	}
	escrowed, err := escrowfile.New(&escrowfile.Params{
		Dir:          cfg.secretsDir,
		ControlPlane: cp,
		Logger:       logger,
	})
	if err != nil {
		return err
	}
	events, err := emailevents.New(&emailevents.Params{
		ControlPlane: cp,
		TMS:          tmsClient,
		Logger:       logger,
	})
	if err != nil {
		return err
	}
	agent, err := tenantagent.New(tenantagent.Params{
		ControlPlane: cp,
		TMS:          tmsClient,
		Credentials:  creds,
		Escrow:       escrowed,
		SystemEmail: systememail.New(&systememail.Params{
			TMS:          tmsClient,
			ControlPlane: cp,
			Logger:       logger,
		}),
		Usage:       usage.NewShared(usage.SharedParams{ControlPlane: cp, TMS: tmsClient}),
		EmailEvents: events,
		Version:     cfg.version,
		Logger:      logger,
		Poll:        cfg.poll,
		UsageEvery:  cfg.usageEvery,
	})
	if err != nil {
		return err
	}

	srv := healthServer(cfg.healthAddr, agent.Healthy)
	go func() {
		if serr := srv.ListenAndServe(); serr != nil && !errors.Is(serr, http.ErrServerClosed) {
			logger.Error("health endpoint", "error", serr)
		}
	}()
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()

	if err = start(ctx, creds, logger, cfg.once); err != nil {
		return err
	}
	logger.Info("tenant agent started",
		"tms", cfg.tmsURL,
		"stateDir", cfg.stateDir,
		"secretsDir", cfg.secretsDir,
		"dataPath", cfg.dataPath,
		"version", cfg.version,
	)
	if cfg.once {
		return agent.Iterate(ctx)
	}

	relay, err := telemetryrelay.New(&telemetryrelay.Params{
		ControlPlane: cp,
		Source: hosttelemetry.New(&hosttelemetry.Params{
			DataPath: cfg.dataPath,
			Data:     telemetryrelay.NewDataReader(tmsClient, logger),
			Logger:   logger,
		}),
		Logger: logger,
	})
	if err != nil {
		return err
	}
	go func() {
		if rerr := relay.Run(ctx); rerr != nil && !errors.Is(rerr, context.Canceled) {
			logger.Error("telemetry stopped", "error", rerr)
		}
	}()
	return agent.Run(ctx)
}

func start(ctx context.Context, creds *credential.Manager, logger *slog.Logger, once bool) error {
	retry := backoff.NewExponentialBackOff()
	retry.MaxInterval = startRetry
	for {
		err := creds.Start(ctx)
		if err == nil {
			return nil
		}
		if once {
			return fmt.Errorf("enrol with the control plane: %w", err)
		}
		wait := retry.NextBackOff()
		logger.Warn("cannot load or enrol the server's identity; trying again",
			"error", err, "retryIn", wait.Round(time.Second))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func healthServer(addr string, healthy func() error) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if err := healthy(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	return &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}

func probe(addr string) int {
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get("http://" + host + "/healthz")
	if err != nil {
		return 1
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func load(args []string) (config, error) {
	fs := flag.NewFlagSet("tenant-agent", flag.ContinueOnError)
	var cfg config
	var level string
	poll, err := envDuration("POLL", tenantagent.DefaultPoll)
	if err != nil {
		return cfg, err
	}
	usageEvery, err := envDuration("USAGE_EVERY", tenantagent.DefaultUsageEvery)
	if err != nil {
		return cfg, err
	}
	fs.StringVar(&cfg.stateDir, "state-dir", env("STATE_DIR", "/var/lib/trenova-tenant-agent"),
		"directory for the server's identity and the run lock ($"+envPrefix+"STATE_DIR)")
	fs.StringVar(&cfg.enrolment, "enrolment", env("ENROLMENT", ""),
		"the enrolment bundle staff issued (default <state-dir>/identity/enrolment.json)")
	fs.StringVar(&cfg.secretsDir, "secrets-dir", env("SECRETS_DIR", "/run/trenova/escrow"),
		"directory the escrowed secrets are written to, one file each, for the TMS to read")
	fs.StringVar(&cfg.tmsURL, "tms-url", env("TMS_URL", "http://api:8090"),
		"the TMS's managed listener")
	fs.StringVar(
		&cfg.tokenFile,
		"tms-token-file",
		env("TMS_TOKEN_FILE", "/run/secrets/managed_token"),
		"file holding the TMS's managed.token",
	)
	fs.StringVar(&cfg.dataPath, "data-path", env("DATA_PATH", "/"),
		"path on the volume that holds the server's data, for the disk reading")
	fs.StringVar(&cfg.version, "server-version", env("SERVER_VERSION", ""),
		"the Trenova release the server runs, reported to the control plane")
	fs.StringVar(&cfg.healthAddr, "health-addr", env("HEALTH_ADDR", ":8081"),
		"address of /healthz and /livez")
	fs.DurationVar(&cfg.poll, "poll", poll,
		"how often the agent converges")
	fs.DurationVar(&cfg.usageEvery, "usage-every", usageEvery,
		"how often usage is reported")
	fs.StringVar(&level, "log-level", env("LOG_LEVEL", "info"), "debug, info, warn or error")
	fs.BoolVar(&cfg.once, "once", false, "converge once, report, and exit")
	fs.BoolVar(
		&cfg.healthcheck,
		"healthcheck",
		false,
		"ask a running agent's /healthz and exit 0 when healthy",
	)
	if err = fs.Parse(args); err != nil {
		return cfg, err
	}
	if err = cfg.logLevel.UnmarshalText([]byte(level)); err != nil {
		return cfg, fmt.Errorf("log level %q: %w", level, err)
	}
	if cfg.healthcheck {
		return cfg, nil
	}
	switch {
	case cfg.stateDir == "":
		return cfg, errors.New("the state directory is required")
	case cfg.secretsDir == "":
		return cfg, errors.New("the secrets directory is required")
	case cfg.tokenFile == "":
		return cfg, errors.New("the TMS token file is required")
	case cfg.poll < time.Second:
		return cfg, errors.New("poll must be at least a second")
	case cfg.usageEvery < time.Minute || cfg.usageEvery > 15*time.Minute:
		return cfg, errors.New(
			"usage-every must be between a minute and 15 minutes: the control plane bills a window 30 minutes after it closes",
		)
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(envPrefix + key)); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	raw := env(key, "")
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s%s: %w", envPrefix, key, err)
	}
	return d, nil
}
