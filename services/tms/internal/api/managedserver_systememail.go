package api

import (
	"context"
	"io"
	"net/http"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

const (
	managedSystemEmailPath = "/managed/v1/system-email"
	managedMaxSystemEmail  = 8 << 10
)

type ManagedSystemEmail interface {
	Status(ctx context.Context) (systememail.Status, error)
	Set(ctx context.Context, credential *systememail.Credential) error
	Remove(ctx context.Context) error
}

type managedSystemEmail struct {
	Provider      string `json:"provider"`
	APIKey        string `json:"apiKey"`
	FromAddress   string `json:"fromAddress"`
	FromName      string `json:"fromName"`
	WebhookSecret string `json:"webhookSecret"`
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
			"the body must be {provider, apiKey, fromAddress, fromName, webhookSecret}",
		)
		return
	}
	err = h.systemEmail.Set(r.Context(), &systememail.Credential{
		Provider:      email.Provider(in.Provider),
		APIKey:        in.APIKey,
		FromAddress:   in.FromAddress,
		FromName:      in.FromName,
		WebhookSecret: in.WebhookSecret,
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
