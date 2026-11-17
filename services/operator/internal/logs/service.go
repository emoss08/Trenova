package logs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/health"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/redact"
	"github.com/emoss08/trenova/operator/internal/storage"
)

const (
	KeyPrefix = "support/logs/"

	queueSize      = 8
	bundleTimeout  = 45 * time.Minute
	jobDeadline    = 90 * time.Minute
	maxQueueWait   = 25 * time.Minute
	reportTimeout  = 2 * time.Minute
	reportTries    = 5
	uploadTries    = 3
	remembered     = time.Hour
	maxMessage     = 500
	lastLookback   = 7 * 24 * time.Hour
	maxViewBytes   = 262144
	metaComponents = "components"
	maxMetaSources = 50
)

const (
	NotKeptMessage   = "Log keeping is off on this instance, so there are no kept logs to bundle."
	noStorageMessage = "This instance has no object storage of its own, so a log bundle has nowhere to be kept."
	waitedMessage    = "The bundle waited too long behind other bundles to be collected. Ask for it again."
	slowMessage      = "Collecting the bundle took longer than the instance allows. Ask for a shorter period."
)

var ErrNotKept = errors.New("log keeping is off on this instance")

type Objects interface {
	Put(ctx context.Context, obj storage.Object) error
	Head(ctx context.Context, key string) (storage.ObjectInfo, bool, error)
}

type ControlPlane interface {
	RegisterLogBundle(ctx context.Context, reg instanceapi.LogBundleRegistration) (uuid.UUID, error)
	UpdateLogBundle(ctx context.Context, id uuid.UUID, update instanceapi.LogBundleUpdate) error
}

type Redactors interface {
	Load(ctx context.Context) (*redact.Redactor, error)
}

type Outcome struct {
	Status     instanceapi.LogBundleUpdateStatus
	Message    string
	Key        string
	Size       int64
	Components []instanceapi.LogBundleComponent
}

func Failed(message string) Outcome {
	return Outcome{Status: instanceapi.LogBundleFailed, Message: health.Clean(message, maxMessage)}
}

func Refused(message string) Outcome {
	return Outcome{Status: instanceapi.LogBundleRefused, Message: health.Clean(message, maxMessage)}
}

func (o Outcome) Update() instanceapi.LogBundleUpdate {
	u := instanceapi.LogBundleUpdate{Status: o.Status}
	if o.Status == instanceapi.LogBundleStored {
		key, size := o.Key, o.Size
		components := append([]instanceapi.LogBundleComponent{}, o.Components...)
		u.ObjectKey, u.SizeBytes, u.Components = &key, &size, &components
	}
	if o.Message != "" {
		msg := o.Message
		u.Message = &msg
	}
	return u
}

type Job struct {
	BundleID uuid.UUID
	Since    time.Time
	Until    time.Time
	Done     func(Outcome)
	queuedAt time.Time
}

type Submitted int

const (
	Queued Submitted = iota
	AlreadyRunning
	QueueFull
)

type finished struct {
	outcome Outcome
	at      time.Time
}

type Service struct {
	src        Source
	objects    Objects
	cp         ControlPlane
	tms        TMS
	redactors  Redactors
	instanceID func() uuid.UUID
	dir        string
	caps       Caps
	logger     *slog.Logger
	now        func() time.Time

	jobs chan *Job

	mu       sync.Mutex
	active   map[uuid.UUID]struct{}
	outcomes map[uuid.UUID]finished
	requests map[string]*request
}

type Params struct {
	Source       Source
	Objects      Objects
	ControlPlane ControlPlane
	TMS          TMS
	Redactors    Redactors
	InstanceID   func() uuid.UUID
	Dir          string
	Caps         Caps
	Logger       *slog.Logger
	Now          func() time.Time
}

func New(p *Params) (*Service, error) {
	var missing []string
	if p.Objects == nil {
		missing = append(missing, "object storage")
	}
	if p.ControlPlane == nil {
		missing = append(missing, "the control plane")
	}
	if p.TMS == nil {
		missing = append(missing, "the TMS")
	}
	if p.Redactors == nil {
		missing = append(missing, "the secrets to remove")
	}
	if p.InstanceID == nil {
		missing = append(missing, "the instance's ID")
	}
	if p.Dir == "" {
		missing = append(missing, "scratch space")
	}
	if p.Logger == nil {
		missing = append(missing, "a logger")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("logs: the service needs %s", strings.Join(missing, ", "))
	}
	caps := p.Caps
	if caps == (Caps{}) {
		caps = DefaultCaps
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		src: p.Source, objects: p.Objects, cp: p.ControlPlane, tms: p.TMS, redactors: p.Redactors,
		instanceID: p.InstanceID, dir: p.Dir, caps: caps, logger: p.Logger, now: now,
		jobs:     make(chan *Job, queueSize),
		active:   map[uuid.UUID]struct{}{},
		outcomes: map[uuid.UUID]finished{},
		requests: map[string]*request{},
	}, nil
}

func (s *Service) Kept() bool {
	return s.src != nil
}

func (s *Service) Longest() time.Duration {
	return s.caps.Period
}

func (s *Service) Run(ctx context.Context) error {
	s.clearScratch()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case job := <-s.jobs:
			s.process(ctx, job)
		}
	}
}

func (s *Service) Submit(job *Job) Submitted {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.forget()
	if _, ok := s.active[job.BundleID]; ok {
		return AlreadyRunning
	}
	if f, ok := s.outcomes[job.BundleID]; ok {
		if job.Done != nil {
			go job.Done(f.outcome)
		}
		return AlreadyRunning
	}
	job.queuedAt = time.Now()
	select {
	case s.jobs <- job:
		s.active[job.BundleID] = struct{}{}
		return Queued
	default:
		return QueueFull
	}
}

func (s *Service) forget() {
	cutoff := s.now().Add(-remembered)
	for id, f := range s.outcomes {
		if f.at.Before(cutoff) {
			delete(s.outcomes, id)
		}
	}
}

func (s *Service) Finish(ctx context.Context, id uuid.UUID, outcome Outcome) {
	run, cancel := context.WithTimeout(context.WithoutCancel(ctx), reportTimeout)
	defer cancel()
	if err := s.report(run, id, outcome); err != nil {
		s.logger.WarnContext(ctx, "cannot report a log bundle to the control plane",
			"bundle", id, "status", string(outcome.Status), "error", err)
	}
}

func (s *Service) process(ctx context.Context, job *Job) {
	deadline := job.queuedAt.Add(jobDeadline)
	if limit := time.Now().Add(bundleTimeout); limit.Before(deadline) {
		deadline = limit
	}
	run, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	s.logger.InfoContext(run, "collecting a log bundle",
		"bundle", job.BundleID, "since", job.Since, "until", job.Until)
	var outcome Outcome
	if time.Now().After(job.queuedAt.Add(maxQueueWait)) {
		outcome = Failed(waitedMessage)
	} else {
		outcome = s.collect(run, job)
		if outcome.Status != instanceapi.LogBundleStored && errors.Is(run.Err(), context.DeadlineExceeded) {
			outcome = Failed(slowMessage)
		}
	}
	s.Finish(ctx, job.BundleID, outcome)
	s.logger.InfoContext(ctx, "log bundle done", "bundle", job.BundleID,
		"status", string(outcome.Status), "size", outcome.Size, "message", outcome.Message)
	s.mu.Lock()
	delete(s.active, job.BundleID)
	s.outcomes[job.BundleID] = finished{outcome: outcome, at: s.now()}
	s.mu.Unlock()
	if job.Done != nil {
		job.Done(outcome)
	}
}

func (s *Service) collect(ctx context.Context, job *Job) Outcome {
	if !s.Kept() {
		return Failed(NotKeptMessage)
	}
	if err := s.report(ctx, job.BundleID, Outcome{Status: instanceapi.LogBundleCollecting}); err != nil {
		s.logger.WarnContext(ctx, "cannot report a log bundle as collecting; collecting anyway",
			"bundle", job.BundleID, "error", err)
	}
	key := KeyPrefix + job.BundleID.String() + ".zip"
	info, exists, err := s.objects.Head(ctx, key)
	switch {
	case errors.Is(err, storage.ErrNoDocumentsBucket):
		return Failed(noStorageMessage)
	case err == nil && exists:
		if components, ok := parseComponents(info.Metadata[metaComponents]); ok {
			return Outcome{
				Status: instanceapi.LogBundleStored, Key: key, Size: info.Size, Components: components,
			}
		}
	case err != nil:
		s.logger.WarnContext(ctx, "cannot look for an earlier copy of a log bundle; collecting it",
			"bundle", job.BundleID, "error", err)
	}
	red, err := s.redactors.Load(ctx)
	if err != nil {
		return Failed("Trenova could not read the instance's secrets to remove them from the logs, " +
			"so nothing was collected: " + err.Error())
	}
	bundle, err := writeBundle(ctx, s.src, &bundleSpec{
		bundleID: job.BundleID, instanceID: s.instanceID(), since: job.Since, until: job.Until,
		dir: s.dir, redact: red.Line, created: s.now(),
	}, s.caps)
	if err != nil {
		return Failed("Trenova could not collect the kept logs: " + err.Error())
	}
	defer func() { _ = os.Remove(bundle.Path) }()
	components := summary(bundle.Manifest.Components)
	if err = s.upload(ctx, key, bundle, components); err != nil {
		if errors.Is(err, storage.ErrNoDocumentsBucket) {
			return Failed(noStorageMessage)
		}
		return Failed("Trenova collected the logs but could not store them in the instance's object storage: " +
			err.Error())
	}
	return Outcome{Status: instanceapi.LogBundleStored, Key: key, Size: bundle.Size, Components: components}
}

func (s *Service) upload(
	ctx context.Context,
	key string,
	bundle *Bundle,
	components []instanceapi.LogBundleComponent,
) error {
	f, err := os.Open(bundle.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = backoff.Retry(ctx, func() (struct{}, error) {
		if _, serr := f.Seek(0, io.SeekStart); serr != nil {
			return struct{}{}, backoff.Permanent(serr)
		}
		perr := s.objects.Put(ctx, storage.Object{
			Key:         key,
			Body:        f,
			Size:        bundle.Size,
			SHA256:      bundle.SHA256,
			ContentType: "application/zip",
			Metadata:    map[string]string{metaComponents: formatComponents(components)},
		})
		var oe *storage.ObjectError
		if errors.Is(perr, storage.ErrNoDocumentsBucket) || (errors.As(perr, &oe) && !oe.Retryable()) {
			return struct{}{}, backoff.Permanent(perr)
		}
		return struct{}{}, perr
	}, backoff.WithBackOff(backoff.NewExponentialBackOff()), backoff.WithMaxTries(uploadTries))
	return err
}

func (s *Service) report(ctx context.Context, id uuid.UUID, outcome Outcome) error {
	update := outcome.Update()
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		err := s.cp.UpdateLogBundle(ctx, id, update)
		var problem *controlplane.ProblemError
		if errors.Is(err, controlplane.ErrUnauthenticated) ||
			(errors.As(err, &problem) && problem.Status < 500) {
			return struct{}{}, backoff.Permanent(err)
		}
		return struct{}{}, err
	}, backoff.WithBackOff(backoff.NewExponentialBackOff()), backoff.WithMaxTries(reportTries))
	return err
}

func (s *Service) clearScratch() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		s.logger.Warn("cannot read the scratch space for log bundles", "dir", s.dir, "error", err)
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".zip.part") || strings.HasSuffix(name, ".spill") {
			_ = os.Remove(filepath.Join(s.dir, name))
		}
	}
}

func summary(results []ComponentResult) []instanceapi.LogBundleComponent {
	out := make([]instanceapi.LogBundleComponent, 0, len(results))
	for _, r := range results {
		out = append(out, instanceapi.LogBundleComponent{
			Component: r.Component, Lines: r.Lines, Truncated: r.Truncated,
		})
	}
	return out
}

func formatComponents(components []instanceapi.LogBundleComponent) string {
	parts := make([]string, 0, len(components))
	for _, c := range components {
		cut := "0"
		if c.Truncated {
			cut = "1"
		}
		parts = append(parts, c.Component+":"+strconv.FormatInt(c.Lines, 10)+":"+cut)
	}
	return strings.Join(parts, ",")
}

func parseComponents(raw string) ([]instanceapi.LogBundleComponent, bool) {
	out := []instanceapi.LogBundleComponent{}
	if raw == "" {
		return out, true
	}
	for part := range strings.SplitSeq(raw, ",") {
		fields := strings.Split(part, ":")
		if len(fields) != 3 || !ValidComponent(fields[0]) || len(out) >= maxMetaSources {
			return nil, false
		}
		lines, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || lines < 0 || (fields[2] != "0" && fields[2] != "1") {
			return nil, false
		}
		out = append(out, instanceapi.LogBundleComponent{
			Component: fields[0], Lines: lines, Truncated: fields[2] == "1",
		})
	}
	return out, true
}

func (s *Service) Last(
	ctx context.Context,
	component string,
	lines int,
	redact func(string) string,
) (string, error) {
	if !s.Kept() {
		return "", ErrNotKept
	}
	if !ValidComponent(component) {
		return "", fmt.Errorf("%q is not a component name", component)
	}
	now := s.now()
	p := newPager(s.src, component, now.Add(-lastLookback), now, min(lines, pageLimit))
	var picked []string
	size := 0
	for len(picked) < lines {
		e, ok, err := p.next(ctx)
		if err != nil {
			return "", err
		}
		if !ok {
			break
		}
		line := viewLine(&e, redact)
		if size+len(line) > maxViewBytes {
			break
		}
		size += len(line)
		picked = append(picked, line)
	}
	if len(picked) == 0 {
		return "", fmt.Errorf("the component %q has no kept log lines from the last 7 days", component)
	}
	slices.Reverse(picked)
	return strings.Join(picked, ""), nil
}
