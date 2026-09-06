package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	checklistOnboardingCode  = "DRIVER-ONBOARDING"
	checklistOffboardingCode = "DRIVER-OFFBOARDING"
	checklistSeedDay         = int64(86400)
)

type WorkerChecklistSeed struct {
	seedhelpers.BaseSeed
}

// WorkerChecklistSeed gives the development org a default onboarding and
// offboarding template and opens onboarding checklists for a few recent hires
// so the Checklist tab shows real progress, overdue items and auto-satisfied
// credentials.
//
// Depends on:
//   - WorkerCredential: credential types the checklist items wait for
//   - WorkerEmploymentEvent: the hire events checklists are attached to
func NewWorkerChecklistSeed() *WorkerChecklistSeed {
	seed := &WorkerChecklistSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"WorkerChecklist",
		"1.0.0",
		"Seeds onboarding/offboarding checklist templates and opens checklists for recent hires",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedWorkerCredential, seedhelpers.SeedWorkerEmploymentEvent)
	return seed
}

type checklistSeedRefs struct {
	orgID           pulid.ID
	buID            pulid.ID
	adminID         pulid.ID
	now             int64
	credentialTypes map[string]pulid.ID
	documentTypes   []*documenttype.DocumentType
}

func (s *WorkerChecklistSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}
			refs := &checklistSeedRefs{
				orgID:   org.ID,
				buID:    org.BusinessUnitID,
				adminID: admin.ID,
				now:     timeutils.NowUnix(),
			}
			if err = s.loadLookups(ctx, tx, refs); err != nil {
				return err
			}

			onboarding, err := s.ensureTemplate(ctx, tx, sc, s.onboardingTemplate(refs))
			if err != nil {
				return fmt.Errorf("ensure onboarding template: %w", err)
			}
			if _, err = s.ensureTemplate(ctx, tx, sc, s.offboardingTemplate(refs)); err != nil {
				return fmt.Errorf("ensure offboarding template: %w", err)
			}

			return s.openForRecentHires(ctx, tx, sc, refs, onboarding)
		},
	)
}

func (s *WorkerChecklistSeed) loadLookups(
	ctx context.Context,
	tx bun.Tx,
	refs *checklistSeedRefs,
) error {
	types := make([]*worker.WorkerCredentialType, 0, 16)
	cols := buncolgen.WorkerCredentialTypeColumns
	if err := tx.NewSelect().
		Model(&types).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Scan(ctx); err != nil {
		return fmt.Errorf("load credential types: %w", err)
	}
	refs.credentialTypes = make(map[string]pulid.ID, len(types))
	for _, typ := range types {
		refs.credentialTypes[typ.Code] = typ.ID
	}

	docTypes := make([]*documenttype.DocumentType, 0, 8)
	dcols := buncolgen.DocumentTypeColumns
	if err := tx.NewSelect().
		Model(&docTypes).
		Where(dcols.OrganizationID.Eq(), refs.orgID).
		Where(dcols.BusinessUnitID.Eq(), refs.buID).
		Where(dcols.DocumentCategory.Eq(), documenttype.CategoryWorker).
		Order(dcols.Name.OrderAsc()).
		Scan(ctx); err != nil {
		return fmt.Errorf("load document types: %w", err)
	}
	refs.documentTypes = docTypes
	return nil
}

func (s *WorkerChecklistSeed) credentialItem(
	refs *checklistSeedRefs,
	code, label string,
	due int32,
) *worker.WorkerChecklistTemplateItem {
	return &worker.WorkerChecklistTemplateItem{
		Label:            label,
		Kind:             worker.ChecklistItemCredential,
		Required:         true,
		DueOffsetDays:    due,
		Owner:            worker.ChecklistOwnerSafety,
		CredentialTypeID: refs.credentialTypes[code],
	}
}

func (s *WorkerChecklistSeed) onboardingTemplate(refs *checklistSeedRefs) *worker.WorkerChecklistTemplate {
	items := []*worker.WorkerChecklistTemplateItem{
		s.credentialItem(refs, "CDL", "CDL copied to the driver qualification file", 1),
		s.credentialItem(refs, "MED_CARD", "Medical examiner's certificate on file", 3),
		s.credentialItem(refs, "MVR", "Motor vehicle record pulled and reviewed", 5),
		{
			Label:         "Pre-employment drug & alcohol test result",
			Description:   "Negative result received from the collection site.",
			Kind:          worker.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 5,
			Owner:         worker.ChecklistOwnerSafety,
		},
		{
			Label:         "Road test and evaluation signed off",
			Kind:          worker.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 7,
			Owner:         worker.ChecklistOwnerSafety,
		},
		{
			Label:         "Employee handbook acknowledged",
			Kind:          worker.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 7,
			Owner:         worker.ChecklistOwnerHR,
		},
		{
			Label:         "Direct deposit and tax forms collected",
			Kind:          worker.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 3,
			Owner:         worker.ChecklistOwnerPayroll,
		},
		{
			Label:         "Fuel card issued",
			Kind:          worker.ChecklistItemEquipment,
			Required:      true,
			DueOffsetDays: 2,
			Owner:         worker.ChecklistOwnerFleet,
		},
		{
			Label:         "ELD tablet and login provisioned",
			Kind:          worker.ChecklistItemEquipment,
			Required:      true,
			DueOffsetDays: 2,
			Owner:         worker.ChecklistOwnerIT,
		},
		{
			Label:         "Invited to Dash",
			Description:   "Completes itself once the driver has portal access.",
			Kind:          worker.ChecklistItemPortalAccess,
			Required:      false,
			DueOffsetDays: 1,
			Owner:         worker.ChecklistOwnerDispatch,
		},
	}
	if len(refs.documentTypes) > 0 {
		items = append(items, &worker.WorkerChecklistTemplateItem{
			Label:          "Signed employment application on file",
			Kind:           worker.ChecklistItemDocument,
			Required:       false,
			DueOffsetDays:  3,
			Owner:          worker.ChecklistOwnerHR,
			DocumentTypeID: refs.documentTypes[0].ID,
		})
	}
	return &worker.WorkerChecklistTemplate{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		Code:           checklistOnboardingCode,
		Name:           "Driver onboarding",
		Description:    "Everything a new driver needs before their first dispatch.",
		Kind:           worker.ChecklistKindOnboarding,
		Trigger:        worker.ChecklistTriggerHired,
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		Items:          items,
	}
}

func (s *WorkerChecklistSeed) offboardingTemplate(refs *checklistSeedRefs) *worker.WorkerChecklistTemplate {
	return &worker.WorkerChecklistTemplate{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		Code:           checklistOffboardingCode,
		Name:           "Driver offboarding",
		Description:    "Close out equipment, access and pay when a driver leaves.",
		Kind:           worker.ChecklistKindOffboarding,
		Trigger:        worker.ChecklistTriggerTerminated,
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		Items: []*worker.WorkerChecklistTemplateItem{
			{Label: "Fuel card returned and cancelled", Kind: worker.ChecklistItemEquipment, Required: true, DueOffsetDays: 2, Owner: worker.ChecklistOwnerFleet},
			{Label: "ELD tablet returned", Kind: worker.ChecklistItemEquipment, Required: true, DueOffsetDays: 2, Owner: worker.ChecklistOwnerIT},
			{Label: "Dash access revoked", Description: "Completes itself once portal access is removed.", Kind: worker.ChecklistItemPortalAccess, Required: true, DueOffsetDays: 1, Owner: worker.ChecklistOwnerDispatch},
			{Label: "Final settlement processed", Kind: worker.ChecklistItemTask, Required: true, DueOffsetDays: 14, Owner: worker.ChecklistOwnerPayroll},
			{Label: "Exit interview completed", Kind: worker.ChecklistItemTask, Required: false, DueOffsetDays: 7, Owner: worker.ChecklistOwnerHR},
		},
	}
}

func (s *WorkerChecklistSeed) ensureTemplate(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	template *worker.WorkerChecklistTemplate,
) (*worker.WorkerChecklistTemplate, error) {
	existing := new(worker.WorkerChecklistTemplate)
	cols := buncolgen.WorkerChecklistTemplateColumns
	err := tx.NewSelect().
		Model(existing).
		Relation(buncolgen.WorkerChecklistTemplateRelations.Items).
		Where(cols.OrganizationID.Eq(), template.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), template.BusinessUnitID).
		Where(cols.Code.Eq(), template.Code).
		Scan(ctx)
	if err == nil {
		return existing, nil
	}

	if _, err = tx.NewInsert().Model(template).Exec(ctx); err != nil {
		return nil, err
	}
	if err = sc.TrackCreated(ctx, "worker_checklist_templates", template.ID, s.Name()); err != nil {
		return nil, err
	}
	for i, item := range template.Items {
		item.TemplateID = template.ID
		item.OrganizationID = template.OrganizationID
		item.BusinessUnitID = template.BusinessUnitID
		item.SortOrder = int32(i) //nolint:gosec // item counts are tiny
	}
	if _, err = tx.NewInsert().Model(&template.Items).Exec(ctx); err != nil {
		return nil, err
	}
	return template, nil
}

func (s *WorkerChecklistSeed) openForRecentHires(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *checklistSeedRefs,
	template *worker.WorkerChecklistTemplate,
) error {
	workers := make([]*worker.Worker, 0, 8)
	cols := buncolgen.WorkerColumns
	if err := tx.NewSelect().
		Model(&workers).
		Relation(buncolgen.WorkerRelations.Profile).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(4).
		Scan(ctx); err != nil {
		return fmt.Errorf("load workers: %w", err)
	}

	for index, wrk := range workers {
		open, err := tx.NewSelect().
			Model((*worker.WorkerChecklist)(nil)).
			Where(buncolgen.WorkerChecklistColumns.WorkerID.Eq(), wrk.ID).
			Where(buncolgen.WorkerChecklistColumns.TemplateID.Eq(), template.ID).
			Exists(ctx)
		if err != nil {
			return err
		}
		if open {
			continue
		}

		startedAt := refs.now - int64(3+index*4)*checklistSeedDay
		checklist := template.Instantiate(wrk.ID, startedAt, refs.adminID, pulid.Nil)
		if _, err = tx.NewInsert().Model(checklist).Exec(ctx); err != nil {
			return fmt.Errorf("insert checklist for %s: %w", wrk.ID, err)
		}
		if err = sc.TrackCreated(ctx, "worker_checklists", checklist.ID, s.Name()); err != nil {
			return err
		}
		for i, item := range checklist.Items {
			item.ChecklistID = checklist.ID
			item.OrganizationID = checklist.OrganizationID
			item.BusinessUnitID = checklist.BusinessUnitID
			item.SortOrder = int32(i) //nolint:gosec // item counts are tiny
			if item.Kind == worker.ChecklistItemTask && i%2 == index%2 {
				completedAt := startedAt + checklistSeedDay
				item.Status = worker.ChecklistItemDone
				item.CompletedAt = &completedAt
				item.CompletedByID = refs.adminID
			}
		}
		if _, err = tx.NewInsert().Model(&checklist.Items).Exec(ctx); err != nil {
			return fmt.Errorf("insert checklist items for %s: %w", wrk.ID, err)
		}
	}
	return nil
}

func (s *WorkerChecklistSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *WorkerChecklistSeed) CanRollback() bool {
	return true
}
