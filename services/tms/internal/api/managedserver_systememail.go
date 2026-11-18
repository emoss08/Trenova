package api

import (
	"context"
	"io"
	"net/http"

	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	managedSystemEmailPath       = "/managed/v1/system-email"
	managedSystemEmailEventsPath = "/managed/v1/system-email/events"
	managedMaxSystemEmail        = 8 << 10
	managedMaxEmailEvents        = 50
	managedMaxEmailEventPayload  = 65536
	managedMaxEmailEventsBody    = 8 << 20
)

type ManagedSystemEmail interface {
	Status(ctx context.Context) (systememail.Status, error)
	Set(ctx context.Context, credential *systememail.Credential) error
	Remove(ctx context.Context) error
}

type ManagedSystemEmailEvents interface {
	HandleRelayedSystemEvent(ctx context.Context, relayed *emailservice.RelayedSystemEvent) error
}

type managedSystemEmail struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"apiKey"`
	FromAddress string `json:"fromAddress"`
	FromName    string `json:"fromName"`
}

type managedEmailEvent struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Payload  string `json:"payload"`
}

type managedEmailEvents struct {
	Events []managedEmailEvent `json:"events"`
}

type managedHandledEmailEvents struct {
	Handled []string `json:"handled"`
}

type managedHeldSystemEmail struct {
	Configured  bool   `json:"configured"`
	Provider    string `json:"provider,omitempty"`
	FromAddress string `json:"fromAddress,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

func (h *managedHandler) routeSystemEmail(mux *http.ServeMux) {
	mux.HandleFunc("GET "+managedSystemEmailPath, h.heldSystemEmail)
	mux.HandleFunc("PUT "+managedSystemEmailPath, h.setSystemEmail)
	mux.HandleFunc("DELETE "+managedSystemEmailPath, h.removeSystemEmail)
	mux.HandleFunc("POST "+managedSystemEmailEventsPath, h.relaySystemEmailEvents)
}

func (h *managedHandler) heldSystemEmail(w http.ResponseWriter, r *http.Request) {
	status, err := h.systemEmail.Status(r.Context())
	if err != nil {
		h.l.Error("cannot read the system email sender for the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	out := managedHeldSystemEmail{Configured: status.Configured}
	if status.Configured {
		out.Provider = string(status.Provider)
		out.FromAddress = status.FromAddress
		out.Fingerprint = status.Fingerprint
	}
	h.json(w, http.StatusOK, out)
}

func (h *managedHandler) setSystemEmail(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxSystemEmail))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 8 KiB")
		return
	}
	var in managedSystemEmail
	if err = managedStrictJSON.Unmarshal(body, &in); err != nil {
		h.problem(
			w,
			http.StatusBadRequest,
			"the body must be {provider, apiKey, fromAddress, fromName}",
		)
		return
	}
	err = h.systemEmail.Set(r.Context(), &systememail.Credential{
		Provider:    email.Provider(in.Provider),
		APIKey:      in.APIKey,
		FromAddress: in.FromAddress,
		FromName:    in.FromName,
	})
	invalid, isInvalid := errortypes.RefusalOf(err)
	switch {
	case isInvalid:
		h.problem(w, http.StatusBadRequest, invalid.Error())
		return
	case err != nil:
		h.l.Error("cannot record the system email sender from the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *managedHandler) removeSystemEmail(w http.ResponseWriter, r *http.Request) {
	if err := h.systemEmail.Remove(r.Context()); err != nil {
		h.l.Error("cannot remove the system email sender for the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *managedHandler) relaySystemEmailEvents(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxEmailEventsBody))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 8 MiB")
		return
	}
	var in managedEmailEvents
	if err = managedStrictJSON.Unmarshal(body, &in); err != nil || !validEmailEventBatch(in.Events) {
		h.problem(
			w,
			http.StatusBadRequest,
			`the body must be {events: [{id, provider: "Resend"|"Postmark", payload}]} `+
				"with 1 to 50 events, each id a distinct lowercase UUID",
		)
		return
	}
	out := managedHandledEmailEvents{Handled: make([]string, 0, len(in.Events))}
	for _, event := range in.Events {
		if h.relaySystemEmailEvent(r.Context(), &event) {
			out.Handled = append(out.Handled, event.ID)
		}
	}
	h.l.Info(
		"took the system email events the operator relayed",
		zap.Int("relayed", len(in.Events)),
		zap.Int("handled", len(out.Handled)),
	)
	h.json(w, http.StatusOK, out)
}

func (h *managedHandler) relaySystemEmailEvent(ctx context.Context, event *managedEmailEvent) bool {
	fields := []zap.Field{zap.String("eventId", event.ID), zap.String("provider", event.Provider)}
	if event.Payload == "" || len(event.Payload) > managedMaxEmailEventPayload {
		h.l.Warn("dropped a relayed system email event whose payload is empty or too large", fields...)
		return true
	}
	err := h.emailEvents.HandleRelayedSystemEvent(ctx, &emailservice.RelayedSystemEvent{
		ID:       event.ID,
		Provider: email.Provider(event.Provider),
		Payload:  []byte(event.Payload),
	})
	if err != nil {
		h.l.Error(
			"cannot apply a relayed system email event; the operator relays it again",
			append(fields, zap.Error(err))...,
		)
		return false
	}
	return true
}

func validEmailEventBatch(events []managedEmailEvent) bool {
	if len(events) == 0 || len(events) > managedMaxEmailEvents {
		return false
	}
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		parsed, err := uuid.Parse(event.ID)
		if err != nil || parsed.String() != event.ID {
			return false
		}
		if _, dup := seen[event.ID]; dup {
			return false
		}
		seen[event.ID] = struct{}{}
	}
	return true
}
