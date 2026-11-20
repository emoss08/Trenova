//revive:disable-next-line:var-naming
package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	managedLogRequestsPath = "/managed/v1/log-requests"
	managedLogRequestPath  = "/managed/v1/log-requests/{id}"
	managedLogPullsPath    = "/managed/v1/support/log-pulls"
	managedMaxLogReport    = 16 << 10
	managedMaxLogPull      = 4 << 10
)

type ManagedSupportLogs interface {
	Waiting(ctx context.Context) ([]*supportlog.LogRequest, error)
	Report(ctx context.Context, id pulid.ID, report *supportlog.Report) error
}

type ManagedLogPulls interface {
	RecordLogPull(ctx context.Context, req *supportaccessservice.LogPullRequest) error
}

type managedLogRequester struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type managedLogRequest struct {
	ID          string              `json:"id"`
	Status      string              `json:"status"`
	BundleID    string              `json:"bundleId,omitempty"`
	Since       time.Time           `json:"since"`
	Until       time.Time           `json:"until"`
	RequestedBy managedLogRequester `json:"requestedBy"`
	RequestedAt time.Time           `json:"requestedAt"`
}

type managedLogRequests struct {
	Requests []managedLogRequest `json:"requests"`
}

type managedLogReport struct {
	Status   string `json:"status"`
	BundleID string `json:"bundleId,omitempty"`
	Message  string `json:"message,omitempty"`
}

type managedLogPull struct {
	Email string `json:"email"`
	Kind  string `json:"kind"`
}

func (h *managedHandler) routeSupportLogs(mux *http.ServeMux) {
	mux.HandleFunc("GET "+managedLogRequestsPath, h.dedicatedOnly(h.waitingLogRequests))
	mux.HandleFunc("PUT "+managedLogRequestPath, h.dedicatedOnly(h.reportLogRequest))
	mux.HandleFunc("POST "+managedLogPullsPath, h.dedicatedOnly(h.checkLogPull))
}

func (h *managedHandler) waitingLogRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := h.supportLogs.Waiting(r.Context())
	if err != nil {
		h.l.Error("cannot read the waiting log requests for the operator", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
		return
	}
	out := managedLogRequests{Requests: make([]managedLogRequest, 0, len(requests))}
	for _, req := range requests {
		out.Requests = append(out.Requests, managedLogRequest{
			ID:       req.ID.String(),
			Status:   req.Status.String(),
			BundleID: req.BundleID,
			Since:    time.Unix(req.Since, 0).UTC(),
			Until:    time.Unix(req.Until, 0).UTC(),
			RequestedBy: managedLogRequester{
				Email: req.RequesterEmail,
				Name:  req.RequesterName,
			},
			RequestedAt: time.Unix(req.CreatedAt, 0).UTC(),
		})
	}
	h.json(w, http.StatusOK, out)
}

func (h *managedHandler) reportLogRequest(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("id")
	if !pulid.LooksLike(raw) || pulid.ID(raw).Prefix() != supportlog.RequestIDPrefix {
		h.problem(w, http.StatusNotFound, "no log request has that id")
		return
	}
	id := pulid.ID(raw)
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxLogReport))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 16 KiB")
		return
	}
	var in managedLogReport
	if err = managedStrictJSON.Unmarshal(body, &in); err != nil {
		h.problem(
			w,
			http.StatusBadRequest,
			`the body must be {status: "collecting"|"sent"|"failed", bundleId?, message?}`,
		)
		return
	}

	err = h.supportLogs.Report(r.Context(), id, &supportlog.Report{
		Status:   supportlog.Status(in.Status),
		BundleID: in.BundleID,
		Message:  in.Message,
	})
	invalid, isInvalid := errortypes.RefusalOf(err)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, supportlog.ErrRequestNotFound):
		h.problem(w, http.StatusNotFound, "no log request has that id")
	case errors.Is(err, supportlog.ErrFinished),
		errors.Is(err, supportlog.ErrBundleMismatch),
		errors.Is(err, supportlog.ErrChangedMeanwhile):
		h.problem(w, http.StatusConflict, err.Error())
	case isInvalid:
		h.problem(w, http.StatusBadRequest, invalid.Error())
	default:
		h.l.Error("cannot record the operator's report on a log request", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
	}
}

func (h *managedHandler) checkLogPull(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, managedMaxLogPull))
	if err != nil {
		h.problem(w, http.StatusRequestEntityTooLarge, "the body is larger than 4 KiB")
		return
	}
	var in managedLogPull
	if err = managedStrictJSON.Unmarshal(body, &in); err != nil || in.Email == "" ||
		!supportlog.PullKind(in.Kind).IsValid() {
		h.problem(w, http.StatusBadRequest, `the body must be {email, kind: "lines"|"bundle"}`)
		return
	}

	err = h.logPulls.RecordLogPull(r.Context(), &supportaccessservice.LogPullRequest{
		StaffEmail: in.Email,
		Kind:       supportlog.PullKind(in.Kind),
	})
	var refusal *supportaccess.RefusalError
	switch {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.As(err, &refusal):
		h.problem(w, http.StatusForbidden, "")
	default:
		h.l.Error("cannot check a Trenova support log pull", zap.Error(err))
		h.problem(w, http.StatusInternalServerError, "")
	}
}
