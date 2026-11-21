package tenantagent

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

type SharedControlPlane interface {
	ControlPlane
	ReportSharedUsage(
		ctx context.Context,
		report instanceapi.SharedUsageReport,
	) (instanceapi.SharedUsageAck, error)
}

const (
	opTenants = "tenants"
	opStatus  = "status"
	opStorage = "storage"
	opUsage   = "usage"
)

type PacedControlPlane struct {
	cp    SharedControlPlane
	floor time.Duration
	now   func() time.Time

	mu    sync.Mutex
	until map[string]time.Time
}

func NewPacedControlPlane(
	cp SharedControlPlane,
	floor time.Duration,
	now func() time.Time,
) *PacedControlPlane {
	if floor <= 0 {
		floor = DefaultPoll
	}
	if now == nil {
		now = time.Now
	}
	return &PacedControlPlane{cp: cp, floor: floor, now: now, until: make(map[string]time.Time)}
}

func (p *PacedControlPlane) held(op string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	until, ok := p.until[op]
	if !ok {
		return nil
	}
	left := until.Sub(p.now())
	if left <= 0 {
		delete(p.until, op)
		return nil
	}
	return &controlplane.RetryLaterError{
		Reason:     "the control plane asked this operation to wait",
		RetryAfter: left,
	}
}

func (p *PacedControlPlane) settle(op string, err error) error {
	var later *controlplane.RetryLaterError
	if !errors.As(err, &later) {
		return err
	}
	wait := max(later.RetryAfter, p.floor)
	p.mu.Lock()
	p.until[op] = p.now().Add(wait)
	p.mu.Unlock()
	later.RetryAfter = wait
	return err
}

func (p *PacedControlPlane) SharedTenants(
	ctx context.Context,
	etag string,
) (controlplane.SharedDesired, bool, error) {
	if err := p.held(opTenants); err != nil {
		return controlplane.SharedDesired{}, false, err
	}
	got, changed, err := p.cp.SharedTenants(ctx, etag)
	return got, changed, p.settle(opTenants, err)
}

func (p *PacedControlPlane) ReportSharedStatus(
	ctx context.Context,
	report *instanceapi.SharedStatusReport,
) (instanceapi.SharedReportAck, error) {
	if err := p.held(opStatus); err != nil {
		return instanceapi.SharedReportAck{}, err
	}
	ack, err := p.cp.ReportSharedStatus(ctx, report)
	return ack, p.settle(opStatus, err)
}

func (p *PacedControlPlane) SharedStorage(ctx context.Context) (instanceapi.SharedStorage, error) {
	if err := p.held(opStorage); err != nil {
		return instanceapi.SharedStorage{}, err
	}
	storage, err := p.cp.SharedStorage(ctx)
	return storage, p.settle(opStorage, err)
}

func (p *PacedControlPlane) ReportSharedUsage(
	ctx context.Context,
	report instanceapi.SharedUsageReport,
) (instanceapi.SharedUsageAck, error) {
	if err := p.held(opUsage); err != nil {
		return instanceapi.SharedUsageAck{}, err
	}
	ack, err := p.cp.ReportSharedUsage(ctx, report)
	return ack, p.settle(opUsage, err)
}
