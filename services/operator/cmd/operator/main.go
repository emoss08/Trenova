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
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/yaml"

	"github.com/emoss08/trenova/operator/internal/backup"
	"github.com/emoss08/trenova/operator/internal/commands"
	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/credential/secretstore"
	"github.com/emoss08/trenova/operator/internal/databases"
	"github.com/emoss08/trenova/operator/internal/emailevents"
	"github.com/emoss08/trenova/operator/internal/escrow/escrowsecret"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/importer"
	"github.com/emoss08/trenova/operator/internal/logs"
	"github.com/emoss08/trenova/operator/internal/reconcile"
	"github.com/emoss08/trenova/operator/internal/redact"
	"github.com/emoss08/trenova/operator/internal/release"
	"github.com/emoss08/trenova/operator/internal/selfupgrade"
	"github.com/emoss08/trenova/operator/internal/storage"
	"github.com/emoss08/trenova/operator/internal/supportkeys"
	"github.com/emoss08/trenova/operator/internal/systememail"
	"github.com/emoss08/trenova/operator/internal/telemetry"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmssecret"
	"github.com/emoss08/trenova/operator/internal/usage"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	chartName      = "trenova"
	leaseDuration  = 30 * time.Second
	renewDeadline  = 20 * time.Second
	retryPeriod    = 5 * time.Second
	standbyRecheck = 15 * time.Second
)

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
	bucketCORS      bool
	capacity        values.Capacity
	helmChart       string
	helmChartSpace  string
	deployment      string
	lease           string
	lokiURL         string
	scratchDir      string
	nodeName        string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("the operator stopped", "error", err)
		os.Exit(1)
	}
}

type clients struct {
	kube    kubernetes.Interface
	dyn     dynamic.Interface
	metrics metrics.Interface
	kubelet telemetry.Summaries
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
	used, err := metrics.NewForConfig(rc)
	if err != nil {
		return fmt.Errorf("kubernetes metrics: %w", err)
	}
	ca, err := os.ReadFile(rc.CAFile)
	if err != nil {
		return fmt.Errorf("the cluster's CA bundle: %w", err)
	}
	kubelet, err := telemetry.NewKubeletSummaries(telemetry.KubeletParams{
		CA:        ca,
		TokenFile: rc.BearerTokenFile,
	})
	if err != nil {
		return err
	}
	store := secretstore.New(secretstore.Params{
		Client:    kube,
		Namespace: cfg.namespace,
		Enrolment: cfg.enrolmentSecret,
		Identity:  cfg.identitySecret,
	})
	loop, err := newLoop(
		&cfg,
		clients{kube: kube, dyn: dyn, metrics: used, kubelet: kubelet},
		store,
		base,
		logger,
	)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	var leading, able atomic.Bool
	go becomeAble(ctx, store, &able, logger)
	srv := healthServer(cfg.healthAddr, func() error {
		if leading.Load() {
			return loop.Healthy()
		}
		if !able.Load() {
			return errors.New("standing by: not yet able to reach the control plane")
		}
		return nil
	})
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
		"storageBucketCORS", cfg.bucketCORS,
		"capacityCapped", cfg.capacity.Set(),
		"helmChart", cfg.helmChart,
		"logKeeping", cfg.lokiURL != "",
	)
	return lead(ctx, &leadParams{
		cfg:     &cfg,
		kube:    kube,
		loop:    loop,
		leading: &leading,
		logger:  logger,
	})
}

func newLoop(
	cfg *config,
	c clients,
	store credential.Store,
	base map[string]any,
	logger *slog.Logger,
) (*reconcile.Loop, error) {
	helmCfg, err := release.NewConfiguration(release.ConfigurationParams{
		Namespace: cfg.targetNamespace,
		PlainHTTP: cfg.plainHTTP,
		Logger:    logger,
	})
	if err != nil {
		return nil, err
	}
	transport := &credential.Transport{}
	httpClient := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	cp := controlplane.NewLazy(httpClient)
	creds := credential.NewManager(&credential.ManagerParams{
		Store:     store,
		Transport: transport,
		Renewer:   cp,
		Logger:    logger,
	})
	cp.Locate(func() string { return creds.Current().APIURL })

	tmsClient, err := tms.New(tms.Params{
		URL: cfg.tmsURL,
		Tokens: tmssecret.NewSecretTokens(tmssecret.SecretTokensParams{
			Client:    c.kube,
			Namespace: cfg.targetNamespace,
			Name:      values.AppSecret(cfg.release, base),
		}),
	})
	if err != nil {
		return nil, err
	}
	kept, err := newLogs(cfg, c, cp, tmsClient, creds, logger)
	if err != nil {
		return nil, err
	}
	runner, err := commands.New(&commands.Params{
		ControlPlane: cp,
		Kube:         c.kube,
		Grants:       tmsClient,
		Logs:         kept,
		Namespace:    cfg.targetNamespace,
		Release:      cfg.release,
		Chart:        chartName,
		Logger:       logger,
	})
	if err != nil {
		return nil, err
	}
	events, err := emailevents.New(&emailevents.Params{
		ControlPlane: cp,
		TMS:          tmsClient,
		Logger:       logger,
	})
	if err != nil {
		return nil, err
	}
	checker := health.NewChecker(&health.CheckerParams{
		Kube:      c.kube,
		Dynamic:   c.dyn,
		Namespace: cfg.targetNamespace,
		Release:   cfg.release,
		Chart:     chartName,
	})
	telem, err := telemetry.New(&telemetry.Params{
		ControlPlane: cp,
		Source: telemetry.NewSampler(&telemetry.SamplerParams{
			Kube:      c.kube,
			Metrics:   c.metrics,
			Parts:     checker,
			Summaries: c.kubelet,
			Data:      tmsClient,
			Namespace: cfg.targetNamespace,
			OwnNode:   cfg.nodeName,
			Logger:    logger,
		}),
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}
	upgrader, err := selfupgrade.New(&selfupgrade.Params{
		Dynamic:            c.dyn,
		Kube:               c.kube,
		HelmChartNamespace: cfg.helmChartSpace,
		HelmChart:          cfg.helmChart,
		Namespace:          cfg.namespace,
		Deployment:         cfg.deployment,
		Logger:             logger,
	})
	if err != nil {
		return nil, err
	}
	imports, err := importer.New(&importer.Params{
		Kube:      c.kube,
		Namespace: cfg.targetNamespace,
		Release:   cfg.release,
		Logger:    logger,
	})
	if err != nil {
		return nil, err
	}
	return reconcile.New(&reconcile.Params{
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
			Kube:      c.kube,
			Name:      cfg.release,
			Namespace: cfg.targetNamespace,
			Logger:    logger,
		}),
		Health: checker,
		Escrow: escrowsecret.New(&escrowsecret.Params{
			Client:       c.kube,
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
		SupportKeys:   supportkeys.New(tmsClient, logger),
		SupportGrants: tmsClient,
		Storage: storage.New(&storage.Params{
			PrivateEndpoint: cfg.storagePrivate,
			Client:          c.kube,
			ControlPlane:    cp,
			CORS:            bucketCORS(cfg.bucketCORS),
			Namespace:       cfg.targetNamespace,
			Logger:          logger,
		}),
		Backups: backup.New(&backup.Params{
			Dynamic:   c.dyn,
			Namespace: cfg.targetNamespace,
			Release:   cfg.release,
			Logger:    logger,
		}),
		Databases: databases.New(&databases.Params{
			Dynamic:   c.dyn,
			Namespace: cfg.targetNamespace,
			Release:   cfg.release,
			Logger:    logger,
		}),
		Commands:    runner,
		EmailEvents: events,
		Logs:        kept,
		Telemetry:   telem,
		Operator:    upgrader,
		Imports:     imports,
		Capacity:    cfg.capacity,
		Base:        base,
		Namespace:   cfg.namespace,
		Logger:      logger,
		Poll:        cfg.poll,
		Report:      cfg.report,
		UsageEvery:  cfg.usage,
	})
}

func newLogs(
	cfg *config,
	c clients,
	cp *controlplane.Lazy,
	tmsClient *tms.Client,
	creds *credential.Manager,
	logger *slog.Logger,
) (*logs.Service, error) {
	var src logs.Source
	if cfg.lokiURL != "" {
		loki, err := logs.NewLoki(cfg.lokiURL, nil)
		if err != nil {
			return nil, err
		}
		src = loki
	}
	return logs.New(&logs.Params{
		Source: src,
		Objects: storage.NewDocuments(&storage.DocumentsParams{
			Kube:            c.kube,
			Namespace:       cfg.targetNamespace,
			PrivateEndpoint: cfg.storagePrivate,
		}),
		ControlPlane: cp,
		TMS:          tmsClient,
		Redactors:    redact.Secrets{Kube: c.kube, Namespace: cfg.targetNamespace},
		InstanceID:   creds.InstanceID,
		Dir:          cfg.scratchDir,
		Logger:       logger,
	})
}

type leadParams struct {
	cfg     *config
	kube    kubernetes.Interface
	loop    *reconcile.Loop
	leading *atomic.Bool
	logger  *slog.Logger
}

func lead(ctx context.Context, p *leadParams) error {
	identity, err := os.Hostname()
	if err != nil || identity == "" {
		identity = "trenova-operator-" + uuid.NewString()
	}
	electing, stopElecting := context.WithCancel(context.WithoutCancel(ctx))
	defer stopElecting()
	var mu sync.Mutex
	var running chan struct{}
	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta:  metav1.ObjectMeta{Name: p.cfg.lease, Namespace: p.cfg.namespace},
			Client:     p.kube.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{Identity: identity},
		},
		LeaseDuration:   leaseDuration,
		RenewDeadline:   renewDeadline,
		RetryPeriod:     retryPeriod,
		ReleaseOnCancel: true,
		Name:            p.cfg.lease,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderCtx context.Context) {
				done := make(chan struct{})
				mu.Lock()
				running = done
				mu.Unlock()
				defer close(done)
				runCtx, cancel := context.WithCancel(leaderCtx)
				defer cancel()
				defer context.AfterFunc(ctx, cancel)()
				p.leading.Store(true)
				p.logger.Info("this operator leads: it runs the instance", "identity", identity)
				if err := p.loop.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
					p.logger.Error("the loop stopped", "error", err)
				}
			},
			OnStoppedLeading: func() {
				p.leading.Store(false)
				p.logger.Info("this operator no longer leads", "identity", identity)
			},
		},
	})
	if err != nil {
		return fmt.Errorf("leader election: %w", err)
	}
	go func() {
		<-ctx.Done()
		mu.Lock()
		done := running
		mu.Unlock()
		if done != nil {
			<-done
		}
		stopElecting()
	}()
	elector.Run(electing)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errors.New("this operator lost its lease; it restarts to stand by")
}

func becomeAble(
	ctx context.Context,
	store credential.Store,
	able *atomic.Bool,
	logger *slog.Logger,
) {
	for {
		err := standbyCheck(ctx, store)
		if err == nil {
			able.Store(true)
			logger.Info("able to run the instance")
			return
		}
		logger.Warn("not yet able to run the instance", "error", err)
		if !sleepFor(ctx, standbyRecheck) {
			return
		}
	}
}

func standbyCheck(ctx context.Context, store credential.Store) error {
	ident, ok, err := store.Load(ctx)
	if err != nil {
		return err
	}
	if !ok {
		_, _, err = store.Bundle(ctx)
		return err
	}
	transport := &credential.Transport{}
	if err = transport.Use(&ident); err != nil {
		return err
	}
	api, err := controlplane.New(
		ident.APIURL,
		&http.Client{Transport: transport, Timeout: 30 * time.Second},
	)
	if err != nil {
		return err
	}
	_, _, err = api.Spec(ctx, "")
	return err
}

func sleepFor(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func chartProvenance(verify bool, logger *slog.Logger) release.Provenance {
	if verify {
		return release.NewSigstore()
	}
	logger.Warn("chart signatures are not checked: this operator runs whatever the registry serves")
	return release.Unverified{}
}

func healthServer(addr string, ready func() error) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		if err := ready(); err != nil {
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

var (
	_ commands.ControlPlane    = (*controlplane.Lazy)(nil)
	_ logs.ControlPlane        = (*controlplane.Lazy)(nil)
	_ emailevents.ControlPlane = (*controlplane.Lazy)(nil)
	_ telemetry.ControlPlane   = (*controlplane.Lazy)(nil)
)

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
		helmChart:       env("TRENOVA_OPERATOR_HELMCHART", ""),
		helmChartSpace:  env("TRENOVA_OPERATOR_HELMCHART_NAMESPACE", "kube-system"),
		deployment:      env("TRENOVA_OPERATOR_DEPLOYMENT", "trenova-operator"),
		lease:           env("TRENOVA_OPERATOR_LEASE", "trenova-operator"),
		lokiURL:         env("TRENOVA_OPERATOR_LOKI_URL", ""),
		scratchDir:      env("TRENOVA_OPERATOR_SCRATCH_DIR", "/scratch"),
		nodeName:        env("TRENOVA_OPERATOR_NODE_NAME", ""),
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
	if cfg.bucketCORS, err = booleanOr("TRENOVA_OPERATOR_STORAGE_BUCKET_CORS", true); err != nil {
		return config{}, err
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

func bucketCORS(perBucket bool) storage.CORS {
	if !perBucket {
		return nil
	}
	return storage.S3CORS{}
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
