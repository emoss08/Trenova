package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/emoss08/trenova/operator/internal/backup"
	"github.com/emoss08/trenova/operator/internal/commands"
	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/escrow"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/release"
	"github.com/emoss08/trenova/operator/internal/storage"
	"github.com/emoss08/trenova/operator/internal/systememail"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/usage"
	"github.com/emoss08/trenova/operator/internal/values"
)

const chartName = "trenova"

type config struct {
	namespace       string
	targetNamespace string
	release         string
	enrolmentSecret string
	identitySecret  string
	chartRef        string
	chartPath       string
	plainHTTP       bool
	verifyChart     bool
	valuesFile      string
	poll            time.Duration
	report          time.Duration
	usage           time.Duration
	tmsURL          string
	healthAddr      string
	logLevel        slog.Level
	storagePrivate  string
	capacity        values.Capacity
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("the operator stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := load()
	if err != nil {
		return err
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.logLevel}))
	base, err := baseValues(cfg.valuesFile)
	if err != nil {
		return err
	}
	rc, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("kubernetes: %w", err)
	}
	kube, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return fmt.Errorf("kubernetes: %w", err)
	}
	dyn, err := dynamic.NewForConfig(rc)
	if err != nil {
		return fmt.Errorf("kubernetes: %w", err)
	}
	helmCfg, err := release.NewConfiguration(release.ConfigurationParams{
		Namespace: cfg.targetNamespace,
		PlainHTTP: cfg.plainHTTP,
		Logger:    logger,
	})
	if err != nil {
		return err
	}

	store := credential.NewStore(credential.StoreParams{
		Client:    kube,
		Namespace: cfg.namespace,
		Enrolment: cfg.enrolmentSecret,
		Identity:  cfg.identitySecret,
	})
	transport := &credential.Transport{}
	httpClient := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	cp := &lazyClient{doer: httpClient}
	creds := credential.NewManager(&credential.ManagerParams{
		Store:     store,
		Transport: transport,
		Renewer:   cp,
		Logger:    logger,
	})
	cp.creds = creds

	tmsClient, err := tms.New(tms.Params{
		URL: cfg.tmsURL,
		Tokens: tms.NewSecretTokens(tms.SecretTokensParams{
			Client:    kube,
			Namespace: cfg.targetNamespace,
			Name:      values.AppSecret(cfg.release, base),
		}),
	})
	if err != nil {
		return err
	}

	loop := reconcile.New(&reconcile.Params{
		ControlPlane: cp,
		Credentials:  creds,
		Charts: release.NewSource(release.SourceParams{
			Ref:        cfg.chartRef,
			Path:       cfg.chartPath,
			Registry:   helmCfg.RegistryClient,
			Provenance: chartProvenance(cfg.verifyChart, logger),
		}),
		Releases: release.NewReleaser(release.ReleaserParams{
			Config:    helmCfg,
			Name:      cfg.release,
			Namespace: cfg.targetNamespace,
			Logger:    logger,
		}),
		Health: health.NewChecker(&health.CheckerParams{
			Kube:      kube,
			Dynamic:   dyn,
			Namespace: cfg.targetNamespace,
			Release:   cfg.release,
			Chart:     chartName,
		}),
		Escrow: escrow.New(&escrow.Params{
			Client:       kube,
			ControlPlane: cp,
			Namespace:    cfg.targetNamespace,
			Name:         values.EscrowSecret,
			Logger:       logger,
		}),
		Usage:          usage.New(usage.Params{ControlPlane: cp, TMS: tmsClient}),
		Administrators: tmsClient,
		Features:       tmsClient,
		SystemEmail: systememail.New(&systememail.Params{
			TMS:          tmsClient,
			ControlPlane: cp,
			Logger:       logger,
		}),
		Storage: storage.New(&storage.Params{
			PrivateEndpoint: cfg.storagePrivate,
			Client:          kube,
			ControlPlane:    cp,
			CORS:            storage.S3CORS{},
			Namespace:       cfg.targetNamespace,
			Logger:          logger,
		}),
		Backups: backup.New(&backup.Params{
			Dynamic:   dyn,
			Namespace: cfg.targetNamespace,
			Release:   cfg.release,
			Logger:    logger,
		}),
		Commands: commands.New(&commands.Params{
			Kube:      kube,
			Namespace: cfg.targetNamespace,
			Release:   cfg.release,
			Chart:     chartName,
			Logger:    logger,
		}),
		Capacity:   cfg.capacity,
		Base:       base,
		Namespace:  cfg.namespace,
		Logger:     logger,
		Poll:       cfg.poll,
		Report:     cfg.report,
		UsageEvery: cfg.usage,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	srv := healthServer(cfg.healthAddr, loop)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health endpoint", "error", err)
		}
	}()
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	logger.Info("operator started",
		"targetNamespace", cfg.targetNamespace,
		"release", cfg.release,
		"chart", firstNonEmpty(cfg.chartPath, cfg.chartRef),
		"tms", cfg.tmsURL,
		"storagePrivateEndpoint", cfg.storagePrivate,
		"capacityCapped", cfg.capacity.Set(),
	)
	return loop.Run(ctx)
}

func chartProvenance(verify bool, logger *slog.Logger) release.Provenance {
	if verify {
		return release.NewSigstore()
	}
	logger.Warn("chart signatures are not checked: this operator runs whatever the registry serves")
	return release.Unverified{}
}

func healthServer(addr string, loop *reconcile.Loop) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if err := loop.Healthy(); err != nil {
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

type lazyClient struct {
	creds *credential.Manager
	doer  *http.Client
	api   *controlplane.Client
	url   string
}

func (c *lazyClient) client() (*controlplane.Client, error) {
	url := c.creds.Current().APIURL
	if url == "" {
		return nil, credential.ErrNotEnrolled
	}
	if c.api != nil && c.url == url {
		return c.api, nil
	}
	api, err := controlplane.New(url, c.doer)
	if err != nil {
		return nil, err
	}
	c.api, c.url = api, url
	return api, nil
}

func (c *lazyClient) Spec(ctx context.Context, etag string) (controlplane.Desired, bool, error) {
	api, err := c.client()
	if err != nil {
		return controlplane.Desired{}, false, err
	}
	return api.Spec(ctx, etag)
}

func (c *lazyClient) ReportStatus(ctx context.Context, r instanceapi.StatusReport) error {
	api, err := c.client()
	if err != nil {
		return err
	}
	return api.ReportStatus(ctx, r)
}

func (c *lazyClient) ReportUsage(
	ctx context.Context,
	r instanceapi.UsageReport,
) (instanceapi.UsageAck, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.UsageAck{}, err
	}
	return api.ReportUsage(ctx, r)
}

func (c *lazyClient) Secrets(ctx context.Context) (instanceapi.InstanceSecrets, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.InstanceSecrets{}, err
	}
	return api.Secrets(ctx)
}

func (c *lazyClient) Storage(ctx context.Context) (instanceapi.InstanceStorage, error) {
	api, err := c.client()
	if err != nil {
		return instanceapi.InstanceStorage{}, err
	}
	return api.Storage(ctx)
}

func (c *lazyClient) Renew(ctx context.Context, csrPEM []byte) (credential.Issued, error) {
	api, err := c.client()
	if err != nil {
		return credential.Issued{}, err
	}
	return api.Renew(ctx, csrPEM)
}

func load() (config, error) {
	cfg := config{
		namespace:       env("TRENOVA_OPERATOR_NAMESPACE", ""),
		targetNamespace: env("TRENOVA_OPERATOR_TARGET_NAMESPACE", "trenova"),
		release:         env("TRENOVA_OPERATOR_RELEASE", "trenova"),
		enrolmentSecret: env("TRENOVA_OPERATOR_ENROLMENT_SECRET", "trenova-operator-enrolment"),
		identitySecret:  env("TRENOVA_OPERATOR_IDENTITY_SECRET", "trenova-operator-identity"),
		chartRef:        env("TRENOVA_OPERATOR_CHART", "oci://ghcr.io/emoss08/charts/trenova"),
		chartPath:       env("TRENOVA_OPERATOR_CHART_PATH", ""),
		valuesFile:      env("TRENOVA_OPERATOR_VALUES_FILE", ""),
		healthAddr:      env("TRENOVA_OPERATOR_HEALTH_ADDR", ":8081"),
	}
	if cfg.namespace == "" {
		return config{}, errors.New(
			"TRENOVA_OPERATOR_NAMESPACE is required (the operator's own namespace)",
		)
	}
	if !strings.HasPrefix(cfg.chartRef, "oci://") && cfg.chartPath == "" {
		return config{}, fmt.Errorf(
			"TRENOVA_OPERATOR_CHART %q must be an oci:// reference",
			cfg.chartRef,
		)
	}
	var err error
	if cfg.plainHTTP, err = booleanOr("TRENOVA_OPERATOR_CHART_PLAIN_HTTP", false); err != nil {
		return config{}, err
	}
	if cfg.verifyChart, err = booleanOr("TRENOVA_OPERATOR_CHART_VERIFY", true); err != nil {
		return config{}, err
	}
	if cfg.poll, err = duration("TRENOVA_OPERATOR_POLL_INTERVAL", 15*time.Second); err != nil {
		return config{}, err
	}
	if cfg.report, err = duration("TRENOVA_OPERATOR_REPORT_INTERVAL", time.Minute); err != nil {
		return config{}, err
	}
	if cfg.report > time.Minute {
		return config{}, errors.New(
			"TRENOVA_OPERATOR_REPORT_INTERVAL must be at most 1m: the control plane expects a report every minute",
		)
	}
	if cfg.usage, err = duration("TRENOVA_OPERATOR_USAGE_INTERVAL", 15*time.Minute); err != nil {
		return config{}, err
	}
	if cfg.usage > 15*time.Minute {
		return config{}, errors.New(
			"TRENOVA_OPERATOR_USAGE_INTERVAL must be at most 15m: windows are billed 30m after they end",
		)
	}
	cfg.storagePrivate = env("TRENOVA_OPERATOR_STORAGE_PRIVATE_ENDPOINT", "")
	if !storage.ValidPrivateEndpoint(cfg.storagePrivate) {
		return config{}, fmt.Errorf(
			"TRENOVA_OPERATOR_STORAGE_PRIVATE_ENDPOINT %q must be http(s)://host[:port]",
			cfg.storagePrivate,
		)
	}
	for key, into := range map[string]*int64{
		"TRENOVA_OPERATOR_NODE_VCPU":       &cfg.capacity.VCPU,
		"TRENOVA_OPERATOR_NODE_MEMORY_GIB": &cfg.capacity.MemoryGiB,
		"TRENOVA_OPERATOR_NODE_DISK_GIB":   &cfg.capacity.DiskGiB,
	} {
		if *into, err = count(key); err != nil {
			return config{}, err
		}
	}
	cfg.tmsURL = env("TRENOVA_OPERATOR_TMS_URL", fmt.Sprintf(
		"http://%s.%s.svc:%d",
		values.APIService(cfg.release),
		cfg.targetNamespace,
		tms.Port,
	))
	if err = cfg.logLevel.UnmarshalText(
		[]byte(env("TRENOVA_OPERATOR_LOG_LEVEL", "info")),
	); err != nil {
		return config{}, fmt.Errorf("TRENOVA_OPERATOR_LOG_LEVEL: %w", err)
	}
	return cfg, nil
}

func baseValues(path string) (map[string]any, error) {
	if path == "" {
		return map[string]any{}, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read the base values: %w", err)
	}
	out := map[string]any{}
	if err = yaml.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse the base values %s: %w", path, err)
	}
	return out, nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) (time.Duration, error) {
	raw := env(key, "")
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d < time.Second {
		return 0, fmt.Errorf("%s: %q is not a duration of at least 1s", key, raw)
	}
	return d, nil
}

func count(key string) (int64, error) {
	raw := env(key, "0")
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 || n > 1024 {
		return 0, fmt.Errorf("%s: %q is not a whole number from 0 to 1024", key, raw)
	}
	return n, nil
}

func booleanOr(key string, def bool) (bool, error) {
	switch strings.ToLower(env(key, strconv.FormatBool(def))) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", key)
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
