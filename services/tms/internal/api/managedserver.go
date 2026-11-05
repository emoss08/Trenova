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
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	managedReadingsPath = "/managed/v1/readings"
	managedStatePath    = "/managed/v1/state"
	managedMaxBody      = 1 << 20
	managedTimeout      = 30 * time.Second
	problemJSON         = "application/problem+json"
)

type ManagedUsage interface {
	Readings(ctx context.Context, since int64) ([]managedusageservice.Reading, error)
	SetState(ctx context.Context, state *managedusage.State) error
}

type ManagedServerParams struct {
	fx.In

	Config *managedconfig.Config
	Usage  *managedusageservice.Service
	Logger *zap.Logger
	LC     fx.Lifecycle
}

type managedHandler struct {
	usage ManagedUsage
	token []byte
	l     *zap.Logger
}

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

type managedProblem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func NewManagedHandler(usage ManagedUsage, token string, logger *zap.Logger) http.Handler {
	h := &managedHandler{usage: usage, token: []byte(token), l: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+managedReadingsPath, h.readings)
	mux.HandleFunc("PUT "+managedStatePath, h.setState)
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
			NewManagedHandler(p.Usage, p.Config.Token, logger),
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
	var invalid *errortypes.MultiError
	switch {
	case errors.As(err, &invalid):
		h.problem(w, http.StatusBadRequest, invalid.Error())
		return
	case err != nil:
		h.l.Error("cannot record the usage state from the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
