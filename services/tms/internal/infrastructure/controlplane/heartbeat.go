package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type HeartbeatReporterParams struct {
	fx.In

	Config   *config.Config
	Client   Client
	Registry *platformcatalog.Registry
	LC       fx.Lifecycle
	Logger   *zap.Logger
}

type HeartbeatReporter struct {
	cfg      *config.Config
	client   Client
	registry *platformcatalog.Registry
	logger   *zap.Logger
	now      func() time.Time
	cancel   context.CancelFunc
}

func NewHeartbeatReporter(p HeartbeatReporterParams) *HeartbeatReporter {
	reporter := &HeartbeatReporter{
		cfg:      p.Config,
		client:   p.Client,
		registry: p.Registry,
		logger:   p.Logger.Named("control-plane-heartbeat"),
		now:      time.Now,
	}

	p.LC.Append(fx.Hook{
		OnStart: reporter.start,
		OnStop:  reporter.stop,
	})

	return reporter
}

func (r *HeartbeatReporter) start(ctx context.Context) error {
	if !r.cfg.Platform.ControlPlane.Enabled {
		return nil
	}

	if err := r.send(ctx); err != nil {
		if !failOpenAllowed(r.cfg) {
			return err
		}
		r.logger.Warn("control plane startup heartbeat failed", zap.Error(err))
	}

	heartbeatCtx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	go r.run(heartbeatCtx)
	return nil
}

func (r *HeartbeatReporter) stop(context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

func (r *HeartbeatReporter) run(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.Platform.ControlPlane.GetHeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendCtx, cancel := context.WithTimeout(ctx, r.cfg.Platform.ControlPlane.GetTimeout())
			if err := r.send(sendCtx); err != nil {
				r.logger.Warn("control plane heartbeat failed", zap.Error(err))
			}
			cancel()
		}
	}
}

func (r *HeartbeatReporter) send(ctx context.Context) error {
	req, err := r.buildRequest()
	if err != nil {
		return err
	}

	_, err = r.client.Heartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("send control plane heartbeat: %w", err)
	}

	return nil
}

func (r *HeartbeatReporter) buildRequest() (*services.InstanceHeartbeatRequest, error) {
	products := r.registry.ListProducts()
	features := r.registry.ListFeatures()
	meters := r.registry.ListMeters()
	hash, err := catalogHash(products, features, meters)
	if err != nil {
		return nil, err
	}

	return &services.InstanceHeartbeatRequest{
		InstanceID:     r.cfg.Platform.InstanceID,
		AppVersion:     r.cfg.App.Version,
		DeploymentMode: string(r.cfg.Platform.GetMode()),
		Metadata: map[string]string{
			"appName": r.cfg.App.Name,
			"env":     string(r.cfg.App.Env),
		},
		CatalogHash: hash,
		Products:    products,
		Features:    features,
		Meters:      meters,
		SentAt:      r.now().Unix(),
	}, nil
}

func catalogHash(
	products []platformcatalog.Product,
	features []platformcatalog.Feature,
	meters []platformcatalog.Meter,
) (string, error) {
	body, err := sonic.Marshal(struct {
		Products []platformcatalog.Product `json:"products"`
		Features []platformcatalog.Feature `json:"features"`
		Meters   []platformcatalog.Meter   `json:"meters"`
	}{
		Products: products,
		Features: features,
		Meters:   meters,
	})
	if err != nil {
		return "", fmt.Errorf("marshal platform catalog: %w", err)
	}

	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
