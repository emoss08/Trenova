package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/registry"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	maxHistory     = 10
	actionTimeout  = 10 * time.Minute
	chartCacheSize = 2
)

type Source struct {
	ref        string
	path       string
	registry   *registry.Client
	provenance Provenance
	mu         sync.Mutex
	cache      map[string]*chart.Chart
	order      []string
}

type SourceParams struct {
	Ref        string
	Path       string
	Registry   *registry.Client
	Provenance Provenance
}

func NewSource(p SourceParams) *Source {
	return &Source{
		ref:        p.Ref,
		path:       p.Path,
		registry:   p.Registry,
		provenance: p.Provenance,
		cache:      map[string]*chart.Chart{},
	}
}

func (s *Source) Load(version string) (*chart.Chart, error) {
	if s.path != "" {
		ch, err := loader.Load(s.path)
		if err != nil {
			return nil, fmt.Errorf("load the chart from %s: %w", s.path, err)
		}
		return ch, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.cache[version]; ok {
		return ch, nil
	}
	ref := strings.TrimPrefix(s.ref, "oci://") + ":" + version
	required := s.provenance.Required()
	res, err := s.registry.Pull(
		ref,
		registry.PullOptWithChart(true),
		registry.PullOptWithProv(required),
	)
	if err != nil {
		return nil, fmt.Errorf("pull %s: %w", ref, err)
	}
	var signature []byte
	if required {
		signature = res.Prov.Data
	}
	if err = s.provenance.Verify(res.Chart.Data, signature); err != nil {
		return nil, fmt.Errorf("refuse %s: %w", ref, err)
	}
	ch, err := loader.LoadArchive(bytes.NewReader(res.Chart.Data))
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", ref, err)
	}
	if err = namesRelease(ch, path.Base(s.ref), version); err != nil {
		return nil, fmt.Errorf("refuse %s: %w", ref, err)
	}
	s.cache[version] = ch
	s.order = append(s.order, version)
	if len(s.order) > chartCacheSize {
		delete(s.cache, s.order[0])
		s.order = s.order[1:]
	}
	return ch, nil
}

func namesRelease(ch *chart.Chart, name, version string) error {
	if ch.Metadata == nil || ch.Metadata.Name != name || ch.Metadata.Version != version {
		return ErrChartIdentity
	}
	return nil
}

type Deployed struct {
	Exists       bool
	Status       helmrelease.Status
	ChartVersion string
	AppVersion   string
	Config       map[string]any
	Revision     int
}

type Releaser struct {
	cfg       *action.Configuration
	name      string
	namespace string
	logger    *slog.Logger
}

type ReleaserParams struct {
	Config    *action.Configuration
	Name      string
	Namespace string
	Logger    *slog.Logger
}

func NewReleaser(p ReleaserParams) *Releaser {
	return &Releaser{cfg: p.Config, name: p.Name, namespace: p.Namespace, logger: p.Logger}
}

type ConfigurationParams struct {
	Namespace string
	PlainHTTP bool
	Logger    *slog.Logger
}

func NewConfiguration(p ConfigurationParams) (*action.Configuration, error) {
	namespace, logger := p.Namespace, p.Logger
	flags := genericclioptions.NewConfigFlags(false)
	flags.Namespace = &namespace
	cfg := new(action.Configuration)
	err := cfg.Init(flags, namespace, "secret", func(format string, v ...any) {
		logger.Debug(fmt.Sprintf(format, v...), "component", "helm")
	})
	if err != nil {
		return nil, fmt.Errorf("helm: %w", err)
	}
	opts := []registry.ClientOption{registry.ClientOptWriter(io.Discard)}
	if p.PlainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
	}
	rc, err := registry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("helm registry client: %w", err)
	}
	cfg.RegistryClient = rc
	return cfg, nil
}

func (r *Releaser) Deployed() (Deployed, error) {
	rel, err := action.NewGet(r.cfg).Run(r.name)
	if errors.Is(err, driver.ErrReleaseNotFound) {
		return Deployed{}, nil
	}
	if err != nil {
		return Deployed{}, fmt.Errorf("read release %s: %w", r.name, err)
	}
	return Deployed{
		Exists:       true,
		Status:       rel.Info.Status,
		ChartVersion: rel.Chart.Metadata.Version,
		AppVersion:   rel.Chart.Metadata.AppVersion,
		Config:       rel.Config,
		Revision:     rel.Version,
	}, nil
}

func NeedsApply(d Deployed, ch *chart.Chart, vals map[string]any) (bool, error) {
	if !d.Exists || d.Status != helmrelease.StatusDeployed ||
		d.ChartVersion != ch.Metadata.Version {
		return true, nil
	}
	same, err := equal(d.Config, vals)
	if err != nil {
		return false, err
	}
	return !same, nil
}

func (r *Releaser) Apply(ctx context.Context, ch *chart.Chart, vals map[string]any) error {
	d, err := r.Deployed()
	if err != nil {
		return err
	}
	if d.Exists && (d.Status.IsPending() || d.Status == helmrelease.StatusFailed) {
		if d, err = r.recover(d); err != nil {
			return err
		}
	}
	if !d.Exists {
		install := action.NewInstall(r.cfg)
		install.ReleaseName = r.name
		install.Namespace = r.namespace
		install.Timeout = actionTimeout
		if _, err = install.RunWithContext(ctx, ch, vals); err != nil {
			return fmt.Errorf("install %s %s: %w", r.name, ch.Metadata.Version, err)
		}
		r.logger.Info("installed", "release", r.name, "version", ch.Metadata.Version)
		return nil
	}
	up := action.NewUpgrade(r.cfg)
	up.Namespace = r.namespace
	up.MaxHistory = maxHistory
	up.Timeout = actionTimeout
	if _, err = up.RunWithContext(ctx, r.name, ch, vals); err != nil {
		return fmt.Errorf("upgrade %s to %s: %w", r.name, ch.Metadata.Version, err)
	}
	r.logger.Info("upgraded", "release", r.name, "version", ch.Metadata.Version)
	return nil
}

func (r *Releaser) recover(d Deployed) (Deployed, error) {
	history, err := action.NewHistory(r.cfg).Run(r.name)
	if err != nil {
		return d, fmt.Errorf("read the history of %s: %w", r.name, err)
	}
	deployedBefore := false
	for _, rel := range history {
		if rel.Version != d.Revision && (rel.Info.Status == helmrelease.StatusDeployed ||
			rel.Info.Status == helmrelease.StatusSuperseded) {
			deployedBefore = true
		}
	}
	if !deployedBefore {
		r.logger.Warn(
			"removing a release that never deployed",
			"release",
			r.name,
			"status",
			d.Status,
		)
		un := action.NewUninstall(r.cfg)
		un.Timeout = actionTimeout
		if _, err = un.Run(r.name); err != nil {
			return d, fmt.Errorf("remove the undeployed release %s: %w", r.name, err)
		}
		return Deployed{}, nil
	}
	if d.Status == helmrelease.StatusFailed {
		return d, nil
	}
	r.logger.Warn("rolling back an interrupted change", "release", r.name, "status", d.Status)
	rb := action.NewRollback(r.cfg)
	rb.Timeout = actionTimeout
	rb.MaxHistory = maxHistory
	if err = rb.Run(r.name); err != nil {
		return d, fmt.Errorf("roll back the interrupted change to %s: %w", r.name, err)
	}
	return r.Deployed()
}

func equal(a, b map[string]any) (bool, error) {
	na, err := normalize(a)
	if err != nil {
		return false, err
	}
	nb, err := normalize(b)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(na, nb), nil
}

func normalize(m map[string]any) (any, error) {
	if m == nil {
		m = map[string]any{}
	}
	body, err := sonic.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("compare values: %w", err)
	}
	var out any
	if err = sonic.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("compare values: %w", err)
	}
	return out, nil
}
