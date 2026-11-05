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

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/release"
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
		Usage:      usage.New(usage.Params{ControlPlane: cp, TMS: tmsClient}),
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
