package realtimeservice

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxScopeLength = 200

var errForeignConnection = errortypes.NewAuthorizationError(
	"This live connection belongs to another session",
)

type GatewayParams struct {
	fx.In

	Logger   *zap.Logger
	Config   *config.Config
	Broker   services.RealtimeBroker
	UserRepo repositories.UserRepository
}

// Gateway is what a browser talks to: it opens a reader's stream and lets that
// reader join a presence scope or say it is typing in one. Every action after
// the stream opens names the stream's connection, and the connection is
// checked against the caller, so one person cannot act through another's.
type Gateway struct {
	l      *zap.Logger
	broker services.RealtimeBroker
	users  repositories.UserRepository
	cfg    config.RealtimeConfig
}

func NewGateway(p GatewayParams) services.RealtimeGateway {
	return &Gateway{
		l:      p.Logger.Named("service.realtime-gateway"),
		broker: p.Broker,
		users:  p.UserRepo,
		cfg:    *p.Config.GetRealtimeConfig(),
	}
}

func (g *Gateway) Open(
	ctx context.Context,
	req *services.OpenRealtimeStreamRequest,
) (services.RealtimeStream, error) {
	if req == nil || req.UserID.IsNil() || req.OrganizationID.IsNil() ||
		req.BusinessUnitID.IsNil() {
		return nil, errortypes.NewAuthenticationError(
			"A signed-in user is required for live updates",
		)
	}

	user, err := g.users.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  req.OrganizationID,
			BuID:   req.BusinessUnitID,
			UserID: req.UserID,
		},
		LookupUserID: req.UserID,
	})
	if err != nil {
		return nil, err
	}

	return g.broker.Subscribe(ctx, &services.RealtimeSubscribeRequest{
		Connection: &services.RealtimeConnection{
			ConnectionID:   pulid.MustNew("rtc_").String(),
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
			UserID:         req.UserID,
			Name:           user.Name,
			Portal:         req.Portal,
		},
		JoinUsers:   req.JoinUsers && !req.Portal,
		LastEventID: req.LastEventID,
	})
}

func (g *Gateway) JoinPresence(
	ctx context.Context,
	req *services.RealtimeScopeRequest,
) (*services.RealtimePresenceSnapshot, error) {
	conn, err := g.connectionFor(ctx, req)
	if err != nil {
		return nil, err
	}

	return g.broker.JoinPresence(ctx, conn, req.Scope)
}

func (g *Gateway) LeavePresence(ctx context.Context, req *services.RealtimeScopeRequest) error {
	conn, err := g.connectionFor(ctx, req)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}

	return g.broker.LeavePresence(ctx, conn, req.Scope)
}

// SignalTyping tells the scope's other members that the caller is typing, at
// most once per throttle window per connection. A stop always goes through and
// reopens the window, so the next keystroke is announced at once.
func (g *Gateway) SignalTyping(ctx context.Context, req *services.RealtimeTypingRequest) error {
	if req == nil {
		return errortypes.NewValidationError("scope", errortypes.ErrRequired, "Scope is required")
	}

	conn, err := g.connectionFor(ctx, &req.RealtimeScopeRequest)
	if err != nil {
		return err
	}

	throttleKey := "typing:" + conn.ConnectionID + ":" + req.Scope
	if req.Stop {
		if err = g.broker.Release(ctx, throttleKey); err != nil {
			g.l.Warn("could not reset typing throttle", zap.Error(err))
		}
	} else {
		allowed, throttleErr := g.broker.Throttle(
			ctx,
			throttleKey,
			g.cfg.GetTypingThrottle(),
		)
		if throttleErr != nil {
			return throttleErr
		}
		if !allowed {
			return nil
		}
	}

	payload, err := sonic.Marshal(services.RealtimeTypingEvent{
		Scope:  req.Scope,
		UserID: conn.UserID.String(),
		Name:   conn.Name,
		Stop:   req.Stop,
	})
	if err != nil {
		return err
	}

	g.broker.Enqueue(&services.RealtimeEnvelope{
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		Event:          services.RealtimeEventTyping,
		Scope:          req.Scope,
		Payload:        payload,
		Ephemeral:      true,
	})

	return nil
}

func (g *Gateway) Drain() {
	g.broker.Drain()
}

// connectionFor resolves the stream a request names and proves it is the
// caller's own, in the caller's current tenant.
func (g *Gateway) connectionFor(
	ctx context.Context,
	req *services.RealtimeScopeRequest,
) (*services.RealtimeConnection, error) {
	if err := validateScopeRequest(req); err != nil {
		return nil, err
	}

	conn, err := g.broker.Connection(ctx, req.ConnectionID)
	if err != nil {
		return nil, err
	}

	if conn.UserID != req.UserID ||
		conn.OrganizationID != req.OrganizationID ||
		conn.BusinessUnitID != req.BusinessUnitID {
		return nil, errForeignConnection
	}
	if conn.Portal {
		return nil, errortypes.NewAuthorizationError("Presence is not available in the portal")
	}

	return conn, nil
}

func validateScopeRequest(req *services.RealtimeScopeRequest) error {
	if req == nil {
		return errortypes.NewValidationError("scope", errortypes.ErrRequired, "Scope is required")
	}

	multiErr := errortypes.NewMultiError()
	if req.ConnectionID == "" {
		multiErr.Add("connectionId", errortypes.ErrRequired, "Connection ID is required")
	} else if !pulid.LooksLike(req.ConnectionID) {
		multiErr.Add("connectionId", errortypes.ErrInvalid, "Connection ID is invalid")
	}

	switch {
	case req.Scope == "":
		multiErr.Add("scope", errortypes.ErrRequired, "Scope is required")
	case len(req.Scope) > maxScopeLength:
		multiErr.Add("scope", errortypes.ErrInvalid, "Scope is too long")
	case req.Scope == services.RealtimeScopeUsers:
		multiErr.Add("scope", errortypes.ErrInvalid, "Scope is reserved")
	case !validScope(req.Scope):
		multiErr.Add("scope", errortypes.ErrInvalid, "Scope is invalid")
	}

	if req.UserID.IsNil() || req.OrganizationID.IsNil() || req.BusinessUnitID.IsNil() {
		return errortypes.NewAuthenticationError("A signed-in user is required for live updates")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// validScope keeps a scope to the characters its Redis keys and presence index
// are built from, so no scope can reach into another's key.
func validScope(scope string) bool {
	for i := range len(scope) {
		c := scope[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '_', c == '-', c == ':':
		default:
			return false
		}
	}
	return true
}
