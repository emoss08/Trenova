package credential

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotEnrolled = errors.New(
		"no identity and no enrolment bundle: install one as the enrolment Secret",
	)
	ErrBundleExpired = errors.New(
		"the enrolment bundle has expired: issue a new one in the admin",
	)
	ErrReenrolmentNeeded = errors.New(
		"the control plane refuses this operator's certificate and the enrolment bundle has been used: issue a new bundle in the admin",
	)
)

type Issued struct {
	CertPEM []byte
	CAPEM   []byte
}

type Renewer interface {
	Renew(ctx context.Context, csrPEM []byte) (Issued, error)
}

type Manager struct {
	store     Store
	transport *Transport
	renewer   Renewer
	logger    *slog.Logger
	now       func() time.Time
	current   atomic.Pointer[Identity]
}

type ManagerParams struct {
	Store     Store
	Transport *Transport
	Renewer   Renewer
	Logger    *slog.Logger
	Now       func() time.Time
}

func NewManager(p *ManagerParams) *Manager {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Manager{
		store:     p.Store,
		transport: p.Transport,
		renewer:   p.Renewer,
		logger:    p.Logger,
		now:       now,
	}
}

func (m *Manager) Current() Identity {
	if c := m.current.Load(); c != nil {
		return *c
	}
	return Identity{}
}

func (m *Manager) InstanceID() uuid.UUID {
	cur := m.Current()
	return cur.InstanceID
}

func (m *Manager) Start(ctx context.Context) error {
	ident, ok, err := m.store.Load(ctx)
	if err != nil && !errors.Is(err, ErrIdentityInvalid) {
		return err
	}
	if err != nil {
		m.logger.Warn("the stored identity is unusable; enrolling from the bundle", "error", err)
	}
	bundle, hasBundle, err := m.store.Bundle(ctx)
	if err != nil {
		return err
	}
	if ok {
		if hasBundle {
			fresh, ferr := bundle.Identity()
			if ferr == nil && fresh.Fingerprint() != ident.EnrolledFrom &&
				m.now().Before(fresh.NotAfter()) {
				m.logger.Info("a new enrolment bundle was installed; enrolling from it")
				return m.enrol(ctx, &fresh)
			}
		}
		return m.install(&ident)
	}
	if !hasBundle {
		return ErrNotEnrolled
	}
	fresh, err := bundle.Identity()
	if err != nil {
		return err
	}
	return m.enrol(ctx, &fresh)
}

func (m *Manager) Recover(ctx context.Context) error {
	bundle, hasBundle, err := m.store.Bundle(ctx)
	if err != nil {
		return err
	}
	if !hasBundle {
		return ErrReenrolmentNeeded
	}
	fresh, err := bundle.Identity()
	if err != nil {
		return err
	}
	if fresh.Fingerprint() == m.Current().EnrolledFrom {
		return ErrReenrolmentNeeded
	}
	return m.enrol(ctx, &fresh)
}

func (m *Manager) RenewIfDue(ctx context.Context) error {
	cur := m.Current()
	if cur.pair.Leaf == nil {
		return ErrNoIdentity
	}
	if m.now().Before(cur.RenewAt()) {
		return nil
	}
	return m.renew(ctx, &cur)
}

func (m *Manager) enrol(ctx context.Context, bundle *Identity) error {
	if !m.now().Before(bundle.NotAfter()) {
		return ErrBundleExpired
	}
	if err := m.install(bundle); err != nil {
		return err
	}
	if err := m.renew(ctx, bundle); err != nil {
		return fmt.Errorf("first renewal onto the operator's own key: %w", err)
	}
	cur := m.Current()
	m.logger.Info("enrolled", "instance", bundle.InstanceID, "notAfter", cur.NotAfter())
	return nil
}

func (m *Manager) renew(ctx context.Context, cur *Identity) error {
	key, err := GenerateKey(cur.InstanceID)
	if err != nil {
		return err
	}
	issued, err := m.renewer.Renew(ctx, key.CSRPEM)
	if err != nil {
		return err
	}
	next, err := NewIdentity(&IdentityParams{
		InstanceID:   cur.InstanceID,
		APIURL:       cur.APIURL,
		CertPEM:      issued.CertPEM,
		KeyPEM:       key.KeyPEM,
		CAPEM:        issued.CAPEM,
		EnrolledFrom: cur.EnrolledFrom,
	})
	if err != nil {
		return fmt.Errorf("the renewed certificate: %w", err)
	}
	if err = m.store.Save(ctx, &next); err != nil {
		return err
	}
	if err = m.install(&next); err != nil {
		return err
	}
	m.logger.Info("renewed the certificate", "notAfter", next.NotAfter(), "renewAt", next.RenewAt())
	return nil
}

func (m *Manager) install(ident *Identity) error {
	if err := m.transport.Use(ident); err != nil {
		return err
	}
	c := *ident
	m.current.Store(&c)
	return nil
}
