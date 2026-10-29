package workforcekit

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxCredentialField = 100

type ReasonedRecord struct {
	ID     pulid.ID
	Reason string
	Tenant pagination.TenantInfo
	UserID pulid.ID
}

func ReasonedRecordFrom(
	idKey, reasonKey string,
) func(*agenttoolports.ToolExecuteParams) (*ReasonedRecord, error) {
	return func(params *agenttoolports.ToolExecuteParams) (*ReasonedRecord, error) {
		id, err := toolparams.RequirePulid(params.Params, idKey)
		if err != nil {
			return nil, err
		}
		reason, err := toolparams.RequireBoundedText(params.Params, reasonKey, NoteChars)
		if err != nil {
			return nil, err
		}
		return &ReasonedRecord{
			ID:     id,
			Reason: reason,
			Tenant: toolparams.TenantFrom(*params),
			UserID: params.Actor.UserID,
		}, nil
	}
}

const ParamSafetyEventID = "safetyEventId"

const ParamTitle = "title"

type RecordDelete struct {
	ID     pulid.ID
	Tenant pagination.TenantInfo
	UserID pulid.ID
}

func RecordDeleteFrom(key string) func(*agenttoolports.ToolExecuteParams) (*RecordDelete, error) {
	return func(params *agenttoolports.ToolExecuteParams) (*RecordDelete, error) {
		id, err := toolparams.RequirePulid(params.Params, key)
		if err != nil {
			return nil, err
		}
		return &RecordDelete{
			ID:     id,
			Tenant: toolparams.TenantFrom(*params),
			UserID: params.Actor.UserID,
		}, nil
	}
}
