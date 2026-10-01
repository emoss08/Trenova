package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramCredentialID     = "credentialId"     //nolint:gosec // G101: a parameter name, not a credential
	paramCredentialTypeID = "credentialTypeId" //nolint:gosec // G101: a parameter name, not a credential
	paramCredentialNumber = "number"
	paramIssuingAuthority = "issuingAuthority"
	paramIssuedAt         = "issuedAt"
	paramExpiresAt        = wfFieldExpiresAt
	paramRenew            = "renew"
	kindCredential        = "credential"
	maxCredentialField    = 100
)

var credentialFields = []string{
	wfFieldWorkerID, paramCredentialTypeID, fieldStatus, paramCredentialNumber,
	paramIssuingAuthority, paramIssuedAt, paramExpiresAt, wfFieldDocument, wfFieldNotes,
	"verifiedById", "verifiedAt", "archivedAt", "archiveReason",
}

type credentialKeeper interface {
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerCredential, error)
	PlanCreate(
		ctx context.Context,
		req *workercredentialservice.CreateRequest,
	) (*workercredentialservice.CreatePlan, error)
	Create(
		ctx context.Context,
		req *workercredentialservice.CreateRequest,
	) (*worker.WorkerCredential, error)
	PlanUpdate(
		ctx context.Context,
		entity *worker.WorkerCredential,
	) (*workercredentialservice.CredentialChange, error)
	Update(
		ctx context.Context,
		entity *worker.WorkerCredential,
		userID pulid.ID,
	) (*worker.WorkerCredential, error)
	PlanArchive(
		ctx context.Context,
		req *workercredentialservice.StatusRequest,
	) (*workercredentialservice.CredentialChange, error)
	Archive(
		ctx context.Context,
		req *workercredentialservice.StatusRequest,
	) (*worker.WorkerCredential, error)
	PlanAttachDocument(
		ctx context.Context,
		req *workercredentialservice.AttachDocumentRequest,
	) (*workercredentialservice.CredentialChange, error)
	AttachDocument(
		ctx context.Context,
		req *workercredentialservice.AttachDocumentRequest,
	) (*worker.WorkerCredential, error)
}

var _ credentialKeeper = (*workercredentialservice.Service)(nil)

func credentialToolProviders() []any {
	return []any{
		provideRecordWorkerCredentialTool,
		provideUpdateWorkerCredentialTool,
		provideAttachWorkerCredentialDocumentTool,
		provideArchiveWorkerCredentialTool,
	}
}

func credentialIDProperty() map[string]any {
	return idProperty("The credential, from list_worker_credentials or " +
		"list_expiring_credentials. Never guess one.")
}

func credentialRecord(credential *worker.WorkerCredential) toolpreview.Record {
	label := "Credential"
	if credential.CredentialType != nil {
		label = credential.CredentialType.Name
	}
	return wfRecord(permission.ResourceWorkerCredential, credential.ID, label,
		credential.Version)
}

func credentialFactProperties() map[string]any {
	return map[string]any{
		paramCredentialNumber: stringProperty("The number on the card or certificate, "+
			"exactly as printed.", maxCredentialField),
		paramIssuingAuthority: stringProperty("Who issued it, such as the state.",
			maxCredentialField),
		paramIssuedAt:   dayProperty("When it was issued."),
		paramExpiresAt:  dayProperty("When it expires."),
		wfParamDocument: wfDocumentProperty(),
		fieldNotes:      wfNoteProperty("Anything the credential should say."),
	}
}

func applyCredentialFacts(entity *worker.WorkerCredential, params map[string]any) error {
	var err error
	if number, numErr := optionalBoundedText(params, paramCredentialNumber,
		maxCredentialField); numErr != nil {
		return numErr
	} else if number != nil {
		entity.Number = *number
	}
	if authority, authErr := optionalBoundedText(params, paramIssuingAuthority,
		maxCredentialField); authErr != nil {
		return authErr
	} else if authority != nil {
		entity.IssuingAuthority = *authority
	}
	if notes, notesErr := optionalBoundedText(params, fieldNotes, wfNoteChars); notesErr != nil {
		return notesErr
	} else if notes != nil {
		entity.Notes = *notes
	}
	if issued, dayErr := optionalScheduleDay(params, paramIssuedAt); dayErr != nil {
		return dayErr
	} else if issued != nil {
		entity.IssuedAt = issued
	}
	if expires, dayErr := optionalScheduleDay(params, paramExpiresAt); dayErr != nil {
		return dayErr
	} else if expires != nil {
		entity.ExpiresAt = expires
	}
	documentID, err := optionalID(params, wfParamDocument)
	if err != nil {
		return err
	}
	if !documentID.IsNil() {
		entity.DocumentID = documentID
	}
	return nil
}

func credentialCreateFrom(
	params *serviceports.ToolExecuteParams,
) (*workercredentialservice.CreateRequest, error) {
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	typeID, err := requirePulid(params.Params, paramCredentialTypeID)
	if err != nil {
		return nil, err
	}
	renew, err := optionalBoolParam(params.Params, paramRenew, false)
	if err != nil {
		return nil, err
	}
	entity := &worker.WorkerCredential{
		OrganizationID:   params.OrganizationID,
		BusinessUnitID:   params.BusinessUnitID,
		WorkerID:         workerID,
		CredentialTypeID: typeID,
	}
	if err = applyCredentialFacts(entity, params.Params); err != nil {
		return nil, err
	}
	return &workercredentialservice.CreateRequest{
		Entity: entity,
		Renew:  renew,
		UserID: params.Actor.UserID,
	}, nil
}

func newRecordWorkerCredentialTool(credentials credentialKeeper) serviceports.AgentTool {
	properties := credentialFactProperties()
	properties[paramWorkerID] = workerProperty()
	properties[paramCredentialTypeID] = idProperty("The kind of credential, from " +
		"list_worker_credentials. Never guess one.")
	properties[paramRenew] = booleanProperty("Whether this renews the worker's current " +
		"credential of the same kind, which is archived as superseded. A second active " +
		"one is refused without it.")
	spec := withSchema(wfSpec(
		"record_worker_credential",
		"Record a driver's licence, medical card, endorsement, TWIC or other credential "+
			"from the card or certificate in hand, or renew the one on file. The worker's "+
			"compliance is worked out again from it. A person verifies it against the "+
			"original; recording is not verifying.",
		"Adds a credential to the worker's file inside Trenova, which can change whether "+
			"they may be dispatched; nothing is sent, and archive_worker_credential retires it.",
		permission.ResourceWorkerCredential,
		permission.OpCreate,
	), properties, paramWorkerID, paramCredentialTypeID)
	spec.searchTerms = []string{"licence", "license", "medical card", "endorsement", "renewal"}

	return newReportingReceivableTool(spec, receivablePlan[
		*workercredentialservice.CreateRequest, *workercredentialservice.CreatePlan,
	]{
		request: credentialCreateFrom,

		plan: func(
			ctx context.Context,
			req *workercredentialservice.CreateRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workercredentialservice.CreatePlan, error) {
			return credentials.PlanCreate(ctx, req)
		},
		refused: func(*workercredentialservice.CreateRequest) string {
			return "Would record a credential on the worker's file."
		},
		render: func(
			_ *workercredentialservice.CreateRequest,
			plan *workercredentialservice.CreatePlan,
		) (*agent.ToolPreview, error) {
			created, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerCredential, pulid.Nil,
					credentialRecord(plan.Credential).Label, 0),
				plan.Credential, wfOptions(credentialFields...)...)
			if err != nil {
				return nil, err
			}
			changes := []*agent.RecordChange{created}
			summary := fmt.Sprintf("Would record %s on the worker's file.",
				credentialRecord(plan.Credential).Label)
			if plan.Superseded != nil {
				archived := *plan.Superseded
				archived.Status = worker.CredentialStatusArchived
				superseded, supErr := toolpreview.Changed(credentialRecord(plan.Superseded),
					plan.Superseded, &archived, wfOptions(fieldStatus)...)
				if supErr != nil {
					return nil, supErr
				}
				changes = append(changes, superseded)
				summary += " The one it renews is archived as superseded."
			}
			return toolpreview.Build(summary, changes...), nil
		},
		run: func(
			ctx context.Context,
			req *workercredentialservice.CreateRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := credentials.Create(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("recorded", kindCredential, paramCredentialID, created.ID,
				created.WorkerID), nil
		},
	})
}

type credentialEdit struct {
	id     pulid.ID
	params *serviceports.ToolExecuteParams
}

func (e *credentialEdit) entity(
	ctx context.Context,
	credentials credentialKeeper,
) (*worker.WorkerCredential, error) {
	current, err := credentials.Get(ctx, tenantFrom(*e.params), e.id)
	if err != nil {
		return nil, err
	}
	entity := *current
	entity.CredentialType = nil
	entity.Document = nil
	return &entity, applyCredentialFacts(&entity, e.params.Params)
}

func newUpdateWorkerCredentialTool(credentials credentialKeeper) serviceports.AgentTool {
	properties := credentialFactProperties()
	properties[paramCredentialID] = credentialIDProperty()
	spec := targeting(withSchema(wfSpec(
		"update_worker_credential",
		"Correct a credential on file: its number, issuer, dates, document or notes. Give "+
			"only what changes. Changing a fact a person verified clears the verification. "+
			"An archived credential is read-only; record a new one instead.",
		"Corrects a credential inside Trenova, which can change whether the worker may be "+
			"dispatched; nothing is sent, and it is corrected again the same way.",
		permission.ResourceWorkerCredential,
		permission.OpUpdate,
	), properties, paramCredentialID), paramCredentialID, permission.ResourceWorkerCredential)
	spec.searchTerms = []string{"fix credential", "correct license", "fix license"}

	return newReportingReceivableTool(spec, receivablePlan[
		*credentialEdit, *workercredentialservice.CredentialChange,
	]{
		request: func(params *serviceports.ToolExecuteParams) (*credentialEdit, error) {
			id, err := requirePulid(params.Params, paramCredentialID)
			if err != nil {
				return nil, err
			}
			return &credentialEdit{id: id, params: params}, nil
		},
		plan: func(
			ctx context.Context,
			edit *credentialEdit,
			_ *serviceports.ToolExecuteParams,
		) (*workercredentialservice.CredentialChange, error) {
			entity, err := edit.entity(ctx, credentials)
			if err != nil {
				return nil, err
			}
			return credentials.PlanUpdate(ctx, entity)
		},
		refused: func(*credentialEdit) string { return "Would correct a credential." },
		render: func(
			_ *credentialEdit,
			change *workercredentialservice.CredentialChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(credentialRecord(change.After), change.Before,
				change.After, wfOptions(credentialFields...)...)
			if err != nil {
				return nil, err
			}
			summary := "Would correct " + credentialRecord(change.After).Label + "."
			if change.Before.IsVerified() && !change.After.IsVerified() {
				summary += " Its verification is cleared, since a verified fact changes."
			}
			return toolpreview.Build(summary, recorded), nil
		},
		run: func(
			ctx context.Context,
			edit *credentialEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := edit.entity(ctx, credentials)
			if err != nil {
				return nil, err
			}
			updated, err := credentials.Update(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}
			return wfResult("updated", kindCredential, paramCredentialID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

func newAttachWorkerCredentialDocumentTool(credentials credentialKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"attach_worker_credential_document",
		"Attach a scan of the card or certificate, already filed on the worker, to a "+
			"credential. A new document on a verified credential clears the verification.",
		"Links a filed document to a credential inside Trenova; attaching another replaces it.",
		permission.ResourceWorkerCredential,
		permission.OpUpdate,
	), map[string]any{
		paramCredentialID: credentialIDProperty(),
		wfParamDocument:   wfDocumentProperty(),
	}, paramCredentialID, wfParamDocument), paramCredentialID,
		permission.ResourceWorkerCredential)
	spec.searchTerms = []string{"scanned card", "card scan"}

	return newReportingReceivableTool(spec, receivablePlan[
		*workercredentialservice.AttachDocumentRequest, *workercredentialservice.CredentialChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workercredentialservice.AttachDocumentRequest, error) {
			id, err := requirePulid(params.Params, paramCredentialID)
			if err != nil {
				return nil, err
			}
			documentID, err := requirePulid(params.Params, wfParamDocument)
			if err != nil {
				return nil, err
			}
			return &workercredentialservice.AttachDocumentRequest{
				ID:         id,
				DocumentID: documentID,
				TenantInfo: tenantFrom(*params),
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *workercredentialservice.AttachDocumentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workercredentialservice.CredentialChange, error) {
			return credentials.PlanAttachDocument(ctx, req)
		},
		refused: func(*workercredentialservice.AttachDocumentRequest) string {
			return "Would attach a document to a credential."
		},
		render: func(
			_ *workercredentialservice.AttachDocumentRequest,
			change *workercredentialservice.CredentialChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(credentialRecord(change.Before), change.Before,
				change.After, wfOptions(wfFieldDocument, "verifiedById", "verifiedAt")...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would attach the document to "+
				credentialRecord(change.Before).Label+".", recorded), nil
		},
		run: func(
			ctx context.Context,
			req *workercredentialservice.AttachDocumentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			saved, err := credentials.AttachDocument(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("attached a document to", kindCredential, paramCredentialID,
				saved.ID, saved.WorkerID), nil
		},
	})
}

func newArchiveWorkerCredentialTool(credentials credentialKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"archive_worker_credential",
		"Archive a credential that no longer applies, such as an endorsement the driver "+
			"surrendered, with the reason. A required credential archived with no "+
			"replacement leaves the worker out of compliance.",
		"Retires a credential inside Trenova, which can take the worker off dispatch; the "+
			"archived row stays on file, and a new credential is recorded to replace it.",
		permission.ResourceWorkerCredential,
		permission.OpArchive,
	), map[string]any{
		paramCredentialID: credentialIDProperty(),
		fieldReason:       stringProperty("Why it is archived.", wfShortChars),
	}, paramCredentialID, fieldReason), paramCredentialID, permission.ResourceWorkerCredential)
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReportingReceivableTool(spec, receivablePlan[
		*reasonedRecord, *workercredentialservice.CredentialChange,
	]{
		request: reasonedRecordFrom(paramCredentialID, fieldReason),
		plan: func(
			ctx context.Context,
			req *reasonedRecord,
			_ *serviceports.ToolExecuteParams,
		) (*workercredentialservice.CredentialChange, error) {
			return credentials.PlanArchive(ctx, &workercredentialservice.StatusRequest{
				ID:         req.id,
				TenantInfo: req.tenant,
				Reason:     req.reason,
				UserID:     req.userID,
			})
		},
		refused: func(*reasonedRecord) string { return "Would archive a credential." },
		render: func(
			_ *reasonedRecord,
			change *workercredentialservice.CredentialChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(credentialRecord(change.Before), change.Before,
				change.After, append(wfOptions(fieldStatus, "archivedAt", "archivedById",
					"archiveReason"), toolpreview.Volatile("archivedAt"))...)
			if err != nil {
				return nil, err
			}
			summary := "Would archive " + credentialRecord(change.Before).Label + "."
			if !change.Before.IsActive() {
				summary = credentialRecord(change.Before).Label +
					" is already archived; nothing would change."
			}
			return toolpreview.Build(summary, recorded), nil
		},
		run: func(
			ctx context.Context,
			req *reasonedRecord,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			archived, err := credentials.Archive(ctx, &workercredentialservice.StatusRequest{
				ID:         req.id,
				TenantInfo: req.tenant,
				Reason:     req.reason,
				UserID:     req.userID,
			})
			if err != nil {
				return nil, err
			}
			return wfResult("archived", kindCredential, paramCredentialID, archived.ID,
				archived.WorkerID), nil
		},
	})
}

func provideRecordWorkerCredentialTool(s *workercredentialservice.Service) serviceports.AgentTool {
	return newRecordWorkerCredentialTool(s)
}

func provideUpdateWorkerCredentialTool(s *workercredentialservice.Service) serviceports.AgentTool {
	return newUpdateWorkerCredentialTool(s)
}

func provideAttachWorkerCredentialDocumentTool(
	s *workercredentialservice.Service,
) serviceports.AgentTool {
	return newAttachWorkerCredentialDocumentTool(s)
}

func provideArchiveWorkerCredentialTool(
	s *workercredentialservice.Service,
) serviceports.AgentTool {
	return newArchiveWorkerCredentialTool(s)
}
