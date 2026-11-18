package systememail

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
)

const (
	fingerprintVersion = "trenova.system-email.v1"
	fetchRetry         = time.Minute
	fingerprintShown   = 12
)

var (
	ErrNoCredential = errors.New(
		"the spec names a system email sender but the control plane's secrets carry none",
	)
	ErrMismatch = errors.New(
		"the control plane's system email credential does not match the spec's fingerprint",
	)
	ErrWaiting = errors.New(
		"waiting before asking the control plane for the system email credential again",
	)
)

type TMS interface {
	SystemEmail(ctx context.Context) (tms.HeldSystemEmail, error)
	PutSystemEmail(ctx context.Context, sender tms.SystemEmail) error
	DeleteSystemEmail(ctx context.Context) error
}

type ControlPlane interface {
	Secrets(ctx context.Context) (instanceapi.InstanceSecrets, error)
}

type Relay struct {
	tms       TMS
	cp        ControlPlane
	logger    *slog.Logger
	now       func() time.Time
	cached    *tms.SystemEmail
	nextFetch time.Time
}

type Params struct {
	TMS          TMS
	ControlPlane ControlPlane
	Logger       *slog.Logger
	Now          func() time.Time
}

func New(p *Params) *Relay {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Relay{tms: p.TMS, cp: p.ControlPlane, logger: p.Logger, now: now}
}

func Fingerprint(sender *tms.SystemEmail) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		fingerprintVersion,
		sender.Provider,
		sender.FromAddress,
		sender.FromName,
		sender.APIKey,
	}, "\n")))
	return hex.EncodeToString(sum[:])
}

func (r *Relay) Sync(ctx context.Context, desired *instanceapi.SpecSystemEmail) error {
	held, err := r.tms.SystemEmail(ctx)
	if err != nil {
		return err
	}
	if desired == nil {
		r.cached = nil
		if !held.Configured {
			return nil
		}
		if err = r.tms.DeleteSystemEmail(ctx); err != nil {
			return err
		}
		r.logger.Info("system email sender removed from the TMS")
		return nil
	}
	if held.Configured && held.Fingerprint == desired.Fingerprint {
		return nil
	}
	sender, err := r.credential(ctx, desired.Fingerprint)
	if err != nil {
		return err
	}
	if err = r.tms.PutSystemEmail(ctx, sender); err != nil {
		return err
	}
	r.logger.Info(
		"system email sender sent to the TMS",
		"provider", sender.Provider,
		"fromAddress", sender.FromAddress,
		"fingerprint", desired.Fingerprint[:min(fingerprintShown, len(desired.Fingerprint))],
		"replaced", held.Configured,
	)
	return nil
}

func (r *Relay) Held(ctx context.Context) (string, error) {
	held, err := r.tms.SystemEmail(ctx)
	if err != nil {
		return "", err
	}
	if !held.Configured {
		return "", nil
	}
	return held.Fingerprint, nil
}

func (r *Relay) credential(ctx context.Context, want string) (tms.SystemEmail, error) {
	if r.cached != nil && Fingerprint(r.cached) == want {
		return *r.cached, nil
	}
	if r.now().Before(r.nextFetch) {
		return tms.SystemEmail{}, ErrWaiting
	}
	r.nextFetch = r.now().Add(fetchRetry)
	secrets, err := r.cp.Secrets(ctx)
	if err != nil {
		var later *controlplane.RetryLaterError
		if errors.As(err, &later) {
			r.nextFetch = r.now().Add(max(later.RetryAfter, fetchRetry))
		}
		return tms.SystemEmail{}, fmt.Errorf("fetch the system email credential: %w", err)
	}
	if secrets.SystemEmail == nil {
		return tms.SystemEmail{}, ErrNoCredential
	}
	sender := fromSecrets(secrets.SystemEmail)
	if Fingerprint(&sender) != want {
		return tms.SystemEmail{}, ErrMismatch
	}
	r.cached = &sender
	r.nextFetch = time.Time{}
	return sender, nil
}

func fromSecrets(c *instanceapi.SystemEmailCredential) tms.SystemEmail {
	return tms.SystemEmail{
		Provider:    string(c.Provider),
		APIKey:      c.ApiKey,
		FromAddress: string(c.FromAddress),
		FromName:    c.FromName,
	}
}
