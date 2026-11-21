//revive:disable-next-line:var-naming
package api

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

const (
	managedSupportPath       = "/managed/v1/support-access"
	managedSupportGrantsPath = "/managed/v1/support-access/grants"
	managedMaxSupport        = 16 << 10
)

// ManagedSupportAccess takes the control plane's support assertion keys, this instance's id and
// the staff email domain ("Support access" in managed-mode.md), and lists the live grants the
// operator reports back.
type ManagedSupportAccess interface {
	SetKeys(ctx context.Context, keys *supportaccess.KeySet) error
	ClearKeys(ctx context.Context) error
	KeySet(ctx context.Context) (*supportaccess.KeySet, error)
	LiveGrants(ctx context.Context) ([]*supportaccessports.LiveGrant, error)
}

type managedSupportKeys struct {
	InstanceID  string                     `json:"instanceId,omitempty"`
	StaffDomain string                     `json:"staffDomain"`
	Keys        []supportaccess.SigningKey `json:"keys"`
}

type managedHeldSupportKeys struct {
	InstanceID  string   `json:"instanceId,omitempty"`
	StaffDomain string   `json:"staffDomain,omitempty"`
	KeyIDs      []string `json:"keyIds"`
	ReceivedAt  string   `json:"receivedAt,omitempty"`
}

type managedSupportGrant struct {
	ID               string `json:"id"`
	OrganizationID   string `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	StaffEmail       string `json:"staffEmail"`
	Scope            string `json:"scope"`
	ExpiresAt        string `json:"expiresAt"`
}

type managedSupportGrants struct {
	Grants []managedSupportGrant `json:"grants"`
}

func (h *managedHandler) routeSupportAccess(mux *http.ServeMux) {
	mux.HandleFunc("GET "+managedSupportPath, h.heldSupportKeys)
	mux.HandleFunc("PUT "+managedSupportPath, h.setSupportKeys)
	mux.HandleFunc("DELETE "+managedSupportPath, h.clearSupportKeys)
	mux.HandleFunc("GET "+managedSupportGrantsPath, h.dedicatedOnly(h.liveSupportGrants))
}

func (h *managedHandler) setSupportKeys(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxSupport))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 16 KiB")
		return
	}
	var in managedSupportKeys
	if err = managedStrictJSON.Unmarshal(body, &in); err != nil {
		h.problem(
			w,
			http.StatusBadRequest,
			"the body must be {instanceId, staffDomain, keys} with keys a list of Ed25519 JWKs; a shared server leaves instanceId out",
		)
		return
	}
	err = h.support.SetKeys(r.Context(), &supportaccess.KeySet{
		InstanceID:  in.InstanceID,
		StaffDomain: in.StaffDomain,
		Keys:        in.Keys,
	})
	invalid, isInvalid := errortypes.RefusalOf(err)
	switch {
	case isInvalid:
		h.problem(w, http.StatusBadRequest, invalid.Error())
		return
	case err != nil:
		h.l.Error("cannot record the support access keys from the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *managedHandler) clearSupportKeys(w http.ResponseWriter, r *http.Request) {
	if err := h.support.ClearKeys(r.Context()); err != nil {
		h.l.Error("cannot remove the support access keys", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *managedHandler) heldSupportKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.support.KeySet(r.Context())
	if err != nil {
		h.l.Error("cannot read the support access keys for the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	out := managedHeldSupportKeys{KeyIDs: []string{}}
	if keys != nil {
		out.InstanceID = keys.InstanceID
		out.StaffDomain = keys.StaffDomain
		for _, key := range keys.Keys {
			out.KeyIDs = append(out.KeyIDs, key.KeyID)
		}
		out.ReceivedAt = time.Unix(keys.ReceivedAt, 0).UTC().Format(time.RFC3339)
	}
	h.json(w, http.StatusOK, out)
}

func (h *managedHandler) liveSupportGrants(w http.ResponseWriter, r *http.Request) {
	grants, err := h.support.LiveGrants(r.Context())
	if err != nil {
		h.l.Error("cannot list the live support grants for the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	out := managedSupportGrants{Grants: make([]managedSupportGrant, 0, len(grants))}
	for _, live := range grants {
		grant := live.Grant
		out.Grants = append(out.Grants, managedSupportGrant{
			ID:               grant.ID.String(),
			OrganizationID:   grant.OrganizationID.String(),
			OrganizationName: live.OrganizationName,
			StaffEmail:       grant.StaffEmail,
			Scope:            grant.Scope.ControlPlane(),
			ExpiresAt:        time.Unix(grant.ExpiresAt, 0).UTC().Format(time.RFC3339),
		})
	}
	h.json(w, http.StatusOK, out)
}
