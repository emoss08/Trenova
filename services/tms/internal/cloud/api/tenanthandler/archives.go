package tenanthandler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/managedroutes"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"go.uber.org/zap"
)

const (
	ArchivesPath       = "/managed/v1/tenants/{instanceId}/archives"
	ArchivePath        = "/managed/v1/tenants/{instanceId}/archives/{commandId}"
	RestoresPath       = "/managed/v1/tenants/{instanceId}/restores"
	RestorePath        = "/managed/v1/tenants/{instanceId}/restores/{commandId}"
	ImportPath         = "/managed/v1/tenants/{instanceId}/imports/{commandId}"
	ArchiveStoragePath = "/managed/v1/tenants/archive-storage"

	maxCommandBody = 16 << 10
)

var (
	errKeyEncoding = errors.New("each archive key must be standard base64 of 32 bytes")
	errTargetFor   = errors.New(
		"the target must name this tenant, this command, its kind and its access (write for an archive, read for a restore or an import)",
	)
)

type ArchiveKey struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Current bool   `json:"current"`
}

type Target struct {
	InstanceID      string       `json:"instanceId"`
	CommandID       string       `json:"commandId"`
	Kind            string       `json:"kind"`
	Access          string       `json:"access"`
	Endpoint        string       `json:"endpoint"`
	Region          string       `json:"region"`
	Bucket          string       `json:"bucket"`
	Prefix          string       `json:"prefix"`
	AccessKeyID     string       `json:"accessKeyId"`
	SecretAccessKey string       `json:"secretAccessKey"`
	SessionToken    string       `json:"sessionToken,omitempty"`
	ExpiresAt       *time.Time   `json:"expiresAt,omitempty"`
	ArchiveKeys     []ArchiveKey `json:"archiveKeys,omitempty"`
}

func (t *Target) archiveAccess() (*tenantarchive.Access, error) {
	access := &tenantarchive.Access{
		Location: tenantarchive.Location{
			Endpoint: strings.TrimSpace(t.Endpoint),
			Region:   strings.TrimSpace(t.Region),
			Bucket:   strings.TrimSpace(t.Bucket),
			Prefix:   t.Prefix,
		},
		Credentials: tenantarchive.Credentials{
			AccessKey:    t.AccessKeyID,
			SecretKey:    t.SecretAccessKey,
			SessionToken: t.SessionToken,
		},
	}
	for _, k := range t.ArchiveKeys {
		material, err := base64.StdEncoding.DecodeString(k.Key)
		if err != nil || len(material) != tenantarchive.KeySize {
			return nil, errKeyEncoding
		}
		access.Keys = append(
			access.Keys,
			tenantarchive.Key{ID: k.ID, Material: material, Current: k.Current},
		)
	}
	return access, nil
}

func (t *Target) commandAccess(
	instanceID, commandID string,
	kind tenantcommand.Kind,
) (*tenantarchive.Access, error) {
	want := "read"
	if kind == tenantcommand.KindArchive {
		want = "write"
	}
	if t.InstanceID != instanceID || t.CommandID != commandID || t.Kind != string(kind) ||
		t.Access != want {
		return nil, errTargetFor
	}
	return t.archiveAccess()
}

type ArchiveBody struct {
	CommandID string `json:"commandId"`
	Purpose   string `json:"purpose"`
	Target    Target `json:"target"`
}

type RestoreBody struct {
	CommandID    string `json:"commandId"`
	ExpectSHA256 string `json:"expectSha256"`
	Target       Target `json:"target"`
}

type StorageBody struct {
	Endpoint        string       `json:"endpoint"`
	Region          string       `json:"region"`
	Bucket          string       `json:"bucket"`
	AccessKeyID     string       `json:"accessKeyId"`
	SecretAccessKey string       `json:"secretAccessKey"`
	SessionToken    string       `json:"sessionToken,omitempty"`
	ExpiresAt       *time.Time   `json:"expiresAt,omitempty"`
	ArchiveKeys     []ArchiveKey `json:"archiveKeys"`
}

type StorageAnswer struct {
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

func (h *Handler) routeArchives(mux *http.ServeMux) {
	mux.HandleFunc("POST "+ArchivesPath, h.sharedOnly(h.startArchive))
	mux.HandleFunc("GET "+ArchivePath, h.sharedOnly(h.command(tenantcommand.KindArchive)))
	mux.HandleFunc("POST "+RestoresPath, h.sharedOnly(h.startRestore))
	mux.HandleFunc("GET "+RestorePath, h.sharedOnly(h.command(tenantcommand.KindRestore)))
	mux.HandleFunc("GET "+ImportPath, h.sharedOnly(h.command(tenantcommand.KindImport)))
	mux.HandleFunc("PUT "+ArchiveStoragePath, h.sharedOnly(h.setStorage))
}

func (h *Handler) startArchive(w http.ResponseWriter, r *http.Request) {
	var body ArchiveBody
	if !h.decode(w, r, &body, "the body must be {commandId, purpose, target}") {
		return
	}
	instanceID := r.PathValue("instanceId")
	access, err := body.Target.commandAccess(instanceID, body.CommandID, tenantcommand.KindArchive)
	if err != nil {
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.lifecycle.StartArchive(r.Context(), instanceID, &tenantlifecycle.ArchiveRequest{
		CommandID: body.CommandID,
		Purpose:   tenantarchive.Purpose(body.Purpose),
		Access:    access,
	})
	h.answerCommand(w, instanceID, body.CommandID, view, err)
}

func (h *Handler) startRestore(w http.ResponseWriter, r *http.Request) {
	var body RestoreBody
	if !h.decode(w, r, &body, "the body must be {commandId, expectSha256, target}") {
		return
	}
	instanceID := r.PathValue("instanceId")
	access, err := body.Target.commandAccess(instanceID, body.CommandID, tenantcommand.KindRestore)
	if err != nil {
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
		return
	}
	view, err := h.lifecycle.StartRestore(r.Context(), instanceID, &tenantlifecycle.RestoreRequest{
		CommandID:    body.CommandID,
		ExpectSHA256: body.ExpectSHA256,
		Access:       access,
	})
	h.answerCommand(w, instanceID, body.CommandID, view, err)
}

func (h *Handler) command(kind tenantcommand.Kind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceID, commandID := r.PathValue("instanceId"), r.PathValue("commandId")
		view, err := h.lifecycle.Command(r.Context(), instanceID, commandID, kind)
		if err != nil {
			h.commandProblem(w, instanceID, commandID, err)
			return
		}
		managedroutes.WriteJSON(w, h.l, http.StatusOK, view)
	}
}

func (h *Handler) setStorage(w http.ResponseWriter, r *http.Request) {
	var body StorageBody
	if !h.decode(
		w,
		r,
		&body,
		"the body must be {endpoint, region, bucket, accessKeyId, secretAccessKey, archiveKeys}",
	) {
		return
	}
	target := Target{
		Endpoint:        body.Endpoint,
		Region:          body.Region,
		Bucket:          body.Bucket,
		AccessKeyID:     body.AccessKeyID,
		SecretAccessKey: body.SecretAccessKey,
		SessionToken:    body.SessionToken,
		ArchiveKeys:     body.ArchiveKeys,
	}
	access, err := target.archiveAccess()
	if err != nil {
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
		return
	}
	var expires time.Time
	if body.ExpiresAt != nil {
		expires = *body.ExpiresAt
	}
	if err = h.lifecycle.SetNightlyGrant(r.Context(), access, expires); err != nil {
		h.commandProblem(w, "", "", err)
		return
	}
	managedroutes.WriteJSON(w, h.l, http.StatusOK, StorageAnswer{ExpiresAt: body.ExpiresAt})
}

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, into any, shape string) bool {
	raw, ok := managedroutes.ReadBody(w, r, h.l, maxCommandBody)
	if !ok {
		return false
	}
	if err := sonic.Unmarshal(raw, into); err != nil {
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, shape)
		return false
	}
	return true
}

func (h *Handler) answerCommand(
	w http.ResponseWriter,
	instanceID, commandID string,
	view *tenantcommand.View,
	err error,
) {
	if err != nil {
		h.commandProblem(w, instanceID, commandID, err)
		return
	}
	managedroutes.WriteJSON(w, h.l, http.StatusAccepted, view)
}

func (h *Handler) commandProblem(w http.ResponseWriter, instanceID, commandID string, err error) {
	switch {
	case errors.Is(err, cloudtenantports.ErrNotFound),
		errors.Is(err, cloudtenantports.ErrCommandNotFound):
		managedroutes.WriteProblem(w, h.l, http.StatusNotFound, err.Error())
	case errors.Is(err, tenantlifecycle.ErrCommandIDInvalid),
		errors.Is(err, tenantlifecycle.ErrPurposeInvalid),
		errors.Is(err, tenantlifecycle.ErrExpectInvalid),
		errors.Is(err, tenantlifecycle.ErrGrantExpiry),
		errors.Is(err, tenantarchive.ErrLocationInvalid),
		errors.Is(err, tenantarchive.ErrCredentialsMissing),
		errors.Is(err, tenantarchive.ErrKeyInvalid),
		errors.Is(err, tenantarchive.ErrKeyNotExpected):
		managedroutes.WriteProblem(w, h.l, http.StatusBadRequest, err.Error())
	case errors.Is(err, tenantlifecycle.ErrTenantBusy):
		managedroutes.WriteProblem(w, h.l, http.StatusLocked, err.Error())
	case errors.Is(err, tenantlifecycle.ErrCommandConflict),
		errors.Is(err, tenantlifecycle.ErrTenantState):
		managedroutes.WriteProblem(w, h.l, http.StatusConflict, err.Error())
	default:
		h.l.Error("cannot answer a tenant archive command",
			zap.String("instanceId", instanceID),
			zap.String("commandId", commandID),
			zap.Error(err),
		)
		managedroutes.WriteProblem(w, h.l, http.StatusInternalServerError, "")
	}
}
