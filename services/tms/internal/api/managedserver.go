//revive:disable-next-line:var-naming
package api

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportlogservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/datausageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/systememailservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	managedReadingsPath = "/managed/v1/readings"
	managedStatePath    = "/managed/v1/state"
	managedFeaturesPath = "/managed/v1/features"
	managedInvitePath   = "/managed/v1/first-administrator/invite"
	managedMaxBody      = 1 << 20
	managedMaxInvite    = 4 << 10
	managedTimeout      = 30 * time.Second
	problemJSON         = "application/problem+json"
)

type ManagedUsage interface {
	Readings(ctx context.Context, since int64) ([]managedusageservice.Reading, error)
	SetState(ctx context.Context, state *managedusage.State) error
}

type ManagedFeatures interface {
	State(ctx context.Context) (*managedfeatures.State, error)
	SetFeatures(ctx context.Context, received *managedfeatures.Received) error
}

type ManagedServerParams struct {
	fx.In

	Config   *managedconfig.Config
	Usage    *managedusageservice.Receiver
	Features *managedfeaturesservice.Receiver
	Invites  instancebootstrapports.FirstAdministratorInviteService
	Logger   *zap.Logger
	LC       fx.Lifecycle

	SystemEmail *systememailservice.Service
	Email       *emailservice.Service

	SupportAccess *supportaccessservice.Service
	SupportLogs   *supportlogservice.Service

	DataUsage *datausageservice.Service
}

type ManagedHandlerParams struct {
	Usage    ManagedUsage
	Features ManagedFeatures
	Invites  instancebootstrapports.FirstAdministratorInviteService
	Token    string
	Logger   *zap.Logger

	SystemEmail       ManagedSystemEmail
	SystemEmailEvents ManagedSystemEmailEvents

	SupportAccess ManagedSupportAccess
	SupportLogs   ManagedSupportLogs
	LogPulls      ManagedLogPulls

	DataUsage ManagedDataUsage
}

type managedHandler struct {
	usage    ManagedUsage
	features ManagedFeatures
	invites  instancebootstrapports.FirstAdministratorInviteService
	token    []byte
	l        *zap.Logger

	systemEmail ManagedSystemEmail
	emailEvents ManagedSystemEmailEvents

	support     ManagedSupportAccess
	supportLogs ManagedSupportLogs
	logPulls    ManagedLogPulls

	dataUsage ManagedDataUsage
}

type managedInvite struct {
	TokenSHA256 string    `json:"tokenSha256"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type managedInviteAnswer struct {
	SetUp             bool   `json:"setUp"`
	InviteTokenSHA256 string `json:"inviteTokenSha256,omitempty"`
}

var managedStrictJSON = sonic.Config{DisallowUnknownFields: true}.Froze()

type managedReading struct {
	Meter    string `json:"meter"`
	Hour     string `json:"hour"`
	Quantity int64  `json:"quantity"`
}

type managedReadings struct {
	Readings []managedReading `json:"readings"`
}

type managedWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type managedState struct {
	Window  *managedWindow       `json:"window"`
	Overage bool                 `json:"overage"`
	Meters  []managedusage.Meter `json:"meters"`
}

type managedAddOnEnding struct {
	AddOn  platformcatalog.AddOnKey `json:"addon"`
	EndsAt time.Time                `json:"endsAt"`
}

type managedFeatureList struct {
	Features        *[]platformcatalog.FeatureKey `json:"features"`
	PlanURL         *string                       `json:"planUrl"`
	AddOnEndings    *[]managedAddOnEnding         `json:"addonEndings"`
	PlanEndsAt      *time.Time                    `json:"planEndsAt"`
	FinalBackupDays int                           `json:"finalBackupDays"`
}

func (l *managedFeatureList) received() *managedfeatures.Received {
	out := &managedfeatures.Received{
		Features:        *l.Features,
		PlanURL:         l.PlanURL,
		FinalBackupDays: l.FinalBackupDays,
	}
	if l.PlanEndsAt != nil && !l.PlanEndsAt.IsZero() {
		out.PlanEndsAt = l.PlanEndsAt.Unix()
	}
	if l.AddOnEndings != nil {
		endings := make([]managedfeatures.AddOnEnding, 0, len(*l.AddOnEndings))
		for _, e := range *l.AddOnEndings {
			ending := managedfeatures.AddOnEnding{AddOn: e.AddOn}
			if !e.EndsAt.IsZero() {
				ending.EndsAt = e.EndsAt.Unix()
			}
			endings = append(endings, ending)
		}
		out.AddOnEndings = &endings
	}
	return out
}

type managedHeldFeatures struct {
	Features   []platformcatalog.FeatureKey `json:"features"`
	ReceivedAt string                       `json:"receivedAt,omitempty"`
}

type managedProblem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func NewManagedHandler(p *ManagedHandlerParams) http.Handler {
	h := &managedHandler{
		usage:    p.Usage,
		features: p.Features,
		invites:  p.Invites,
		token:    []byte(p.Token),
		l:        p.Logger,

		systemEmail: p.SystemEmail,
		emailEvents: p.SystemEmailEvents,

		support:     p.SupportAccess,
		supportLogs: p.SupportLogs,
		logPulls:    p.LogPulls,

		dataUsage: p.DataUsage,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+managedReadingsPath, h.readings)
	mux.HandleFunc("PUT "+managedStatePath, h.setState)
	mux.HandleFunc("GET "+managedFeaturesPath, h.heldFeatures)
	mux.HandleFunc("PUT "+managedFeaturesPath, h.setFeatures)
	mux.HandleFunc("PUT "+managedInvitePath, h.installInvite)
	h.routeSystemEmail(mux)
	h.routeSupportAccess(mux)
	h.routeSupportLogs(mux)
	h.routeDataUsage(mux)
	return h.authenticate(mux)
}

func StartManagedServer(p ManagedServerParams) {
	if !p.Config.Enabled {
		return
	}
	logger := p.Logger.Named("managed-server")
	server := &http.Server{
		Addr: net.JoinHostPort(p.Config.GetHost(), strconv.Itoa(p.Config.GetPort())),
		Handler: http.TimeoutHandler(
			NewManagedHandler(&ManagedHandlerParams{
				Usage:    p.Usage,
				Features: p.Features,
				Invites:  p.Invites,
				Token:    p.Config.Token,
				Logger:   logger,

				SystemEmail:       p.SystemEmail,
				SystemEmailEvents: p.Email,

				SupportAccess: p.SupportAccess,
				SupportLogs:   p.SupportLogs,
				LogPulls:      p.SupportAccess,

				DataUsage: p.DataUsage,
			}),
			managedTimeout,
			"",
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       managedTimeout,
		WriteTimeout:      managedTimeout + 5*time.Second,
	}
	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", server.Addr)
			if err != nil {
				return fmt.Errorf("listen for the operator on %s: %w", server.Addr, err)
			}
			logger.Info("serving the operator", zap.String("address", server.Addr))
			go func() {
				if serr := server.Serve(ln); serr != nil && !errors.Is(serr, http.ErrServerClosed) {
					logger.Error("the managed server stopped", zap.Error(serr))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
}

func (h *managedHandler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		given, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(given), h.token) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="trenova-managed"`)
			h.problem(w, http.StatusUnauthorized, "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *managedHandler) readings(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("since")
	since, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		h.problem(w, http.StatusBadRequest, "since must be an RFC 3339 time on the hour")
		return
	}
	readings, err := h.usage.Readings(r.Context(), since.Unix())
	switch {
	case errors.Is(err, managedusageservice.ErrSinceNotOnTheHour),
		errors.Is(err, managedusageservice.ErrSinceInFuture),
		errors.Is(err, managedusageservice.ErrSinceTooOld):
		h.problem(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		h.l.Error("cannot read usage for the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	out := managedReadings{Readings: make([]managedReading, 0, len(readings))}
	for _, rd := range readings {
		out.Readings = append(out.Readings, managedReading{
			Meter:    string(rd.Meter),
			Hour:     time.Unix(rd.Hour, 0).UTC().Format(time.RFC3339),
			Quantity: rd.Quantity,
		})
	}
	h.json(w, http.StatusOK, out)
}

func (h *managedHandler) setState(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxBody))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 1 MiB")
		return
	}
	var in managedState
	if err = sonic.Unmarshal(body, &in); err != nil {
		h.problem(w, http.StatusBadRequest, "the body is not a usage state")
		return
	}
	state := &managedusage.State{Overage: in.Overage, Meters: in.Meters}
	if state.Meters == nil {
		state.Meters = []managedusage.Meter{}
	}
	if in.Window != nil {
		state.WindowStart, state.WindowEnd = in.Window.Start.Unix(), in.Window.End.Unix()
	}
	err = h.usage.SetState(r.Context(), state)
	invalid, isInvalid := errortypes.RefusalOf(err)
	switch {
	case isInvalid:
		h.problem(w, http.StatusBadRequest, invalid.Error())
		return
	case err != nil:
		h.l.Error("cannot record the usage state from the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *managedHandler) setFeatures(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxBody))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 1 MiB")
		return
	}
	var in managedFeatureList
	if err = sonic.Unmarshal(body, &in); err != nil || in.Features == nil {
		h.problem(
			w,
			http.StatusBadRequest,
			"the body must be {features, planUrl?, addonEndings?} with features an array of feature keys",
		)
		return
	}
	err = h.features.SetFeatures(r.Context(), in.received())
	invalid, isInvalid := errortypes.RefusalOf(err)
	switch {
	case isInvalid:
		h.problem(w, http.StatusBadRequest, invalid.Error())
		return
	case err != nil:
		h.l.Error("cannot record the feature list from the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *managedHandler) heldFeatures(w http.ResponseWriter, r *http.Request) {
	state, err := h.features.State(r.Context())
	if err != nil {
		h.l.Error("cannot read the feature list for the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	out := managedHeldFeatures{Features: []platformcatalog.FeatureKey{}}
	if state != nil {
		if state.Features != nil {
			out.Features = state.Features
		}
		out.ReceivedAt = time.Unix(state.ReceivedAt, 0).UTC().Format(time.RFC3339)
	}
	h.json(w, http.StatusOK, out)
}

func (h *managedHandler) installInvite(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxInvite))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 4 KiB")
		return
	}
	var in managedInvite
	if err = managedStrictJSON.Unmarshal(body, &in); err != nil || in.ExpiresAt.IsZero() {
		h.problem(
			w,
			http.StatusBadRequest,
			"the body must be {tokenSha256, expiresAt} with expiresAt an RFC 3339 time",
		)
		return
	}
	result, err := h.invites.Install(r.Context(), &instancebootstrapports.FirstAdministratorInvite{
		TokenSHA256: in.TokenSHA256,
		ExpiresAt:   in.ExpiresAt.Unix(),
	})
	invalid, isInvalid := errortypes.RefusalOf(err)
	switch {
	case isInvalid:
		h.problem(w, http.StatusBadRequest, invalid.Error())
		return
	case errors.Is(err, instancebootstrapports.ErrNotBootstrapped):
		h.problem(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		h.l.Error("cannot install the first administrator's invite", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	h.json(w, http.StatusOK, managedInviteAnswer{
		SetUp:             result.SetUp,
		InviteTokenSHA256: result.InviteTokenSHA256,
	})
}

func (h *managedHandler) problem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", problemJSON)
	w.WriteHeader(status)
	body, err := sonic.Marshal(managedProblem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
	if err == nil {
		_, err = w.Write(body)
	}
	if err != nil {
		h.l.Warn("cannot write a problem to the operator", zap.Error(err))
	}
}

func (h *managedHandler) json(w http.ResponseWriter, status int, v any) {
	body, err := sonic.Marshal(v)
	if err != nil {
		h.l.Error("cannot encode an answer to the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err = w.Write(body); err != nil {
		h.l.Warn("cannot write an answer to the operator", zap.Error(err))
	}
}
