package tenanthandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/managedroutes"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	TenantsPath  = "/managed/v1/tenants"
	TenantPath   = "/managed/v1/tenants/{instanceId}"
	ReadingsPath = "/managed/v1/tenants/readings"
	UsagePath    = "/managed/v1/tenants/usage"

	maxEntryBody = 256 << 10
	maxUsageBody = 4 << 20
)

type Lifecycle interface {
	Shared() bool
	Apply(
		ctx context.Context,
		instanceID string,
		desired *cloudtenant.Desired,
		source *tenantarchive.Access,
	) (*cloudtenant.Status, error)
	List(ctx context.Context, after string, limit int) (*tenantlifecycle.StatusPage, error)
	Readings(
		ctx context.Context,
		req *tenantlifecycle.ReadingsRequest,
	) (*tenantlifecycle.ReadingsPage, error)
	SetUsage(
		ctx context.Context,
		entries []tenantlifecycle.TenantUsage,
	) (*tenantlifecycle.UsageAnswer, error)
	StartArchive(
		ctx context.Context,
		instanceID string,
		req *tenantlifecycle.ArchiveRequest,
	) (*tenantcommand.View, error)
	StartRestore(
		ctx context.Context,
		instanceID string,
		req *tenantlifecycle.RestoreRequest,
	) (*tenantcommand.View, error)
	Command(
		ctx context.Context,
		instanceID, commandID string,
		kind tenantcommand.Kind,
	) (*tenantcommand.View, error)
	SetNightlyGrant(ctx context.Context, access *tenantarchive.Access, expiresAt time.Time) error
}

type EntryBody struct {
	cloudtenant.Desired
	ImportTarget *Target `json:"importTarget,omitempty"`
}

type Params struct {
	fx.In

	Lifecycle *tenantlifecycle.Service
	Logger    *zap.Logger
}

type Handler struct {
	lifecycle Lifecycle
	l         *zap.Logger
}

type Result struct {
	fx.Out

	Contributor managedroutes.Contributor `group:"managed_route_contributors"`
}

func New(p Params) Result {
	return Result{Contributor: NewHandler(p.Lifecycle, p.Logger)}
}

func NewHandler(lifecycle Lifecycle, logger *zap.Logger) *Handler {
	return &Handler{lifecycle: lifecycle, l: logger.Named("cloud.tenant-routes")}
}

type usageRequest struct {
	Tenants []tenantlifecycle.TenantUsage `json:"tenants"`
}

func (h *Handler) RouteManaged(mux *http.ServeMux) {
	mux.HandleFunc("GET "+TenantsPath, h.sharedOnly(h.list))
	mux.HandleFunc("GET "+ReadingsPath, h.sharedOnly(h.readings))
	mux.HandleFunc("PUT "+UsagePath, h.sharedOnly(h.setUsage))
	mux.HandleFunc("PUT "+TenantPath, h.sharedOnly(h.apply))
	h.routeArchives(mux)
}

func (h *Handler) sharedOnly(next http.HandlerFunc) http.HandlerFunc {
	if h.lifecycle.Shared() {
		return next
	}
	return managedroutes.Refuse(
		h.l,
		"tenant routes answer only on a shared server (managed.tenancy: shared); a dedicated instance is managed through /managed/v1/features and /managed/v1/state",
	)
}

func (h *Handler) apply(w http.ResponseWriter, r *http.Request) {
	body, ok := managedroutes.ReadBody(w, r, h.l, maxEntryBody)
	if !ok {
		return
	}
	var entry EntryBody
	if err := sonic.Unmarshal(body, &entry); err != nil {
		managedroutes.WriteProblem(
			w,
			h.l,
			http.StatusBadRequest,
			"the body is not a shared tenant entry",
		)
		return
	}
	desired := entry.Desired
	var source *tenantarchive.Access
	if entry.ImportTarget != nil {
		access, aerr := entry.importAccess(r.PathValue("instanceId"))
		if aerr != nil {
			managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, aerr.Error())
			return
		}
		source = access
	}
	status, err := h.lifecycle.Apply(r.Context(), r.PathValue("instanceId"), &desired, source)
	switch {
	case errors.Is(err, cloudtenant.ErrInstanceIDMismatch),
		errors.Is(err, cloudtenant.ErrRevisionInvalid):
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		h.l.Error("cannot apply a tenant entry",
			zap.String("instanceId", desired.InstanceID),
			zap.Int64("revision", desired.Revision),
			zap.Error(err),
		)
		managedroutes.WriteProblem(w, h.l, http.StatusInternalServerError, "")
		return
	}
	managedroutes.WriteJSON(w, h.l, http.StatusOK, status)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, ok := h.limit(w, r)
	if !ok {
		return
	}
	page, err := h.lifecycle.List(r.Context(), r.URL.Query().Get("after"), limit)
	switch {
	case errors.Is(err, tenantlifecycle.ErrCursorInvalid):
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		h.l.Error("cannot list the tenants", zap.Error(err))
		managedroutes.WriteProblem(w, h.l, http.StatusInternalServerError, "")
		return
	}
	managedroutes.WriteJSON(w, h.l, http.StatusOK, page)
}

func (h *Handler) readings(w http.ResponseWriter, r *http.Request) {
	limit, ok := h.limit(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	from, ferr := time.Parse(time.RFC3339, query.Get("from"))
	to, terr := time.Parse(time.RFC3339, query.Get("to"))
	if ferr != nil || terr != nil {
		managedroutes.WriteProblem(
			w,
			h.l,
			http.StatusBadRequest,
			tenantlifecycle.ErrWindowInvalid.Error(),
		)
		return
	}
	page, err := h.lifecycle.Readings(r.Context(), &tenantlifecycle.ReadingsRequest{
		From:  from.Unix(),
		To:    to.Unix(),
		After: query.Get("after"),
		Limit: limit,
	})
	switch {
	case errors.Is(err, tenantlifecycle.ErrWindowInvalid),
		errors.Is(err, tenantlifecycle.ErrCursorInvalid):
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		h.l.Error("cannot read the tenants' usage", zap.Error(err))
		managedroutes.WriteProblem(w, h.l, http.StatusInternalServerError, "")
		return
	}
	managedroutes.WriteJSON(w, h.l, http.StatusOK, page)
}

func (h *Handler) setUsage(w http.ResponseWriter, r *http.Request) {
	body, ok := managedroutes.ReadBody(w, r, h.l, maxUsageBody)
	if !ok {
		return
	}
	var in usageRequest
	if err := sonic.Unmarshal(body, &in); err != nil || in.Tenants == nil {
		managedroutes.WriteProblem(
			w,
			h.l,
			http.StatusBadRequest,
			"the body must be {tenants: [{instanceId, window?, overage, meters}]}",
		)
		return
	}
	answer, err := h.lifecycle.SetUsage(r.Context(), in.Tenants)
	switch {
	case errors.Is(err, tenantlifecycle.ErrTooManyEntries):
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		h.l.Error("cannot record the tenants' usage answers", zap.Error(err))
		managedroutes.WriteProblem(w, h.l, http.StatusInternalServerError, "")
		return
	}
	managedroutes.WriteJSON(w, h.l, http.StatusOK, answer)
}

func (h *Handler) limit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return tenantlifecycle.DefaultPageSize, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > tenantlifecycle.MaxPageSize {
		managedroutes.WriteProblem(
			w,
			h.l,
			http.StatusBadRequest,
			"limit must be 1 to "+strconv.Itoa(tenantlifecycle.MaxPageSize),
		)
		return 0, false
	}
	return n, true
}

func (e *EntryBody) importAccess(instanceID string) (*tenantarchive.Access, error) {
	command := e.ImportCommand()
	if command == nil {
		return nil, errTargetFor
	}
	return e.ImportTarget.commandAccess(instanceID, command.ID, tenantcommand.KindImport)
}
