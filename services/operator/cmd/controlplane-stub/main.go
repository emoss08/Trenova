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

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/fakecp"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("stub stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	spec := fakecp.Spec(uuid.New())
	spec.Hostname = env("STUB_HOSTNAME", spec.Hostname)
	spec.Release.Version = env("STUB_VERSION", spec.Release.Version)
	spec.Allowances = allowances()
	var err error
	if spec.Plan.Vcpu, err = number("STUB_VCPU", spec.Plan.Vcpu); err != nil {
		return err
	}
	if spec.Plan.MemoryGiB, err = number("STUB_MEMORY_GIB", spec.Plan.MemoryGiB); err != nil {
		return err
	}
	if spec.Plan.DiskGiB, err = number("STUB_DISK_GIB", spec.Plan.DiskGiB); err != nil {
		return err
	}
	fake, err := fakecp.New(&fakecp.Params{
		Spec:  spec,
		Hosts: strings.Split(env("STUB_HOSTS", "localhost"), ","),
		URL:   env("STUB_URL", "https://localhost:8443"),
	})
	if err != nil {
		return err
	}
	api := &http.Server{
		Addr:              ":8443",
		Handler:           fake.Handler(),
		TLSConfig:         fake.TLSConfig(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	admin := &http.Server{
		Addr:              ":8080",
		Handler:           adminHandler(fake),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errs := make(chan error, 2)
	go func() { errs <- api.ListenAndServeTLS("", "") }()
	go func() { errs <- admin.ListenAndServe() }()
	logger.Info(
		"control plane stub serving",
		"instance",
		fake.InstanceID,
		"version",
		spec.Release.Version,
	)
	select {
	case err = <-errs:
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = api.Shutdown(shutdown)
	_ = admin.Shutdown(shutdown)
	return err
}

func adminHandler(fake *fakecp.Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /bundle", func(w http.ResponseWriter, _ *http.Request) {
		b, err := fake.Bundle(7 * 24 * time.Hour)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		write(w, b)
	})
	mux.HandleFunc("GET /report", func(w http.ResponseWriter, _ *http.Request) {
		reports := fake.Reports()
		if len(reports) == 0 {
			http.Error(w, "no report yet", http.StatusNotFound)
			return
		}
		write(w, map[string]any{"revision": fake.Revision(), "report": reports[len(reports)-1]})
	})
	mux.HandleFunc("GET /usage", func(w http.ResponseWriter, _ *http.Request) {
		reports := fake.UsageReports()
		if len(reports) == 0 {
			http.Error(w, "no usage report yet", http.StatusNotFound)
			return
		}
		write(w, map[string]any{"reports": len(reports), "last": reports[len(reports)-1]})
	})
	mux.HandleFunc("GET /secrets", func(w http.ResponseWriter, _ *http.Request) {
		write(w, map[string]any{"calls": fake.SecretCalls()})
	})
	mux.HandleFunc("PUT /lifecycle/{lifecycle}", func(w http.ResponseWriter, r *http.Request) {
		lc := instanceapi.InstanceSpecLifecycle(r.PathValue("lifecycle"))
		switch lc {
		case instanceapi.Active, instanceapi.Suspended, instanceapi.Decommissioned:
		default:
			http.Error(w, "unknown lifecycle", http.StatusBadRequest)
			return
		}
		spec := fake.Spec()
		spec.Lifecycle = lc
		fake.SetSpec(&spec)
		write(w, map[string]any{"revision": fake.Revision()})
	})
	return mux
}

func allowances() []instanceapi.SpecAllowance {
	allowance := func(
		meter, app string,
		unit instanceapi.SpecAllowanceUnit,
		quantity int64,
		hardCap bool,
	) instanceapi.SpecAllowance {
		return instanceapi.SpecAllowance{
			Meter:    meter,
			AppMeter: &app,
			Unit:     unit,
			Quantity: quantity,
			HardCap:  hardCap,
		}
	}
	return []instanceapi.SpecAllowance{
		allowance("shipments", "shipments.total", instanceapi.SpecAllowanceUnitCount, 5000, false),
		allowance("ai_credits", "ai.spend_cents", instanceapi.SpecAllowanceUnitCents, 2000, true),
		allowance(
			"document_extractions",
			"document_intelligence.extractions",
			instanceapi.SpecAllowanceUnitCount,
			500,
			true,
		),
		allowance(
			"document_storage",
			"documents.storage_bytes",
			instanceapi.SpecAllowanceUnitBytes,
			50<<30,
			false,
		),
		{Meter: "users", Unit: instanceapi.SpecAllowanceUnitCount, Quantity: 25},
	}
}

func write(w http.ResponseWriter, body any) {
	out, err := sonic.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func number(key string, def int64) (int64, error) {
	raw := env(key, "")
	if raw == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}
