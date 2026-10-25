package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	checklistOnboardingCode  = "DRIVER-ONBOARDING"
	checklistOffboardingCode = "DRIVER-OFFBOARDING"
	checklistSeedDay         = int64(86400)
)

type WorkerChecklistSeed struct {
}

// WorkerChecklistSeed gives the development org a default onboarding and
// offboarding template and opens onboarding checklists for a few recent hires
// so the Checklist tab shows real progress, overdue items and auto-satisfied
// credentials.
//
// Depends on:
//   - WorkerCredential: credential types the checklist items wait for
//   - WorkerEmploymentEvent: the hire events checklists are attached to
var WorkerChecklist = Seeds.Add(seed.Seed{
	Name:        "worker_checklist",
	Kind:        seed.Fixture,
	Description: "Seeds onboarding/offboarding checklist templates and opens checklists for recent hires",
	After:       []*seed.Seed{WorkerCredential, WorkerEmploymentEvent},
	Run:         (&WorkerChecklistSeed{}).Run,
})

type checklistSeedRefs struct {
	orgID           pulid.ID
	buID            pulid.ID
	adminID         pulid.ID
	now             int64
	credentialTypes map[string]pulid.ID
	documentTypes   []*documenttype.DocumentType
}

func (s *WorkerChecklistSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}
	refs := &checklistSeedRefs{
		orgID:   org.ID,
		buID:    org.BusinessUnitID,
		adminID: admin.ID,
		now:     timeutils.NowUnix(),
	}
	if err = s.loadLookups(ctx, db, refs); err != nil {
		return err
	}

	onboarding, err := s.ensureTemplate(ctx, db, sr, s.onboardingTemplate(refs))
	if err != nil {
		return fmt.Errorf("ensure onboarding template: %w", err)
	}
	if _, err = s.ensureTemplate(ctx, db, sr, s.offboardingTemplate(refs)); err != nil {
		return fmt.Errorf("ensure offboarding template: %w", err)
	}

	return s.openForRecentHires(ctx, db, sr, refs, onboarding)
}

func (s *WorkerChecklistSeed) loadLookups(
	ctx context.Context,
	db *keel.DB,
	refs *checklistSeedRefs,
) error {
	types, err := workerdb.WorkerCredentialTypes.AllTenants().Where(
		workerdb.WorkerCredentialTypes.OrganizationID.Eq(refs.orgID),
		workerdb.WorkerCredentialTypes.BusinessUnitID.Eq(refs.buID),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load credential types: %w", err)
	}
	refs.credentialTypes = make(map[string]pulid.ID, len(types))
	for _, typ := range types {
		refs.credentialTypes[typ.Code] = typ.ID
	}

	docTypes, err := documenttypedb.DocumentTypes.AllTenants().Where(
		documenttypedb.DocumentTypes.OrganizationID.Eq(refs.orgID),
		documenttypedb.DocumentTypes.BusinessUnitID.Eq(refs.buID),
		documenttypedb.DocumentTypes.DocumentCategory.Eq(documenttype.CategoryWorker),
	).Order(documenttypedb.DocumentTypes.Name.Asc()).All(ctx, db)
	if err != nil {
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
	db *keel.DB,
	sr *seed.Run,
	template *worker.WorkerChecklistTemplate,
) (*worker.WorkerChecklistTemplate, error) {
	existing, err := workerdb.WorkerChecklistTemplates.AllTenants().Load(workerdb.WorkerChecklistTemplates.Items()).Where(
		workerdb.WorkerChecklistTemplates.OrganizationID.Eq(template.OrganizationID),
		workerdb.WorkerChecklistTemplates.BusinessUnitID.Eq(template.BusinessUnitID),
		workerdb.WorkerChecklistTemplates.Code.Eq(template.Code),
	).First(ctx, db)
	if err == nil {
		return existing, nil
	}

	if err = workerdb.WorkerChecklistTemplates.Create(ctx, db, template); err != nil {
		return nil, err
	}
	for i, item := range template.Items {
		item.TemplateID = template.ID
		item.OrganizationID = template.OrganizationID
		item.BusinessUnitID = template.BusinessUnitID
		item.SortOrder = int32(i) //nolint:gosec // item counts are tiny
	}
	if err = workerdb.WorkerChecklistTemplateItems.CreateMany(ctx, db, template.Items); err != nil {
		return nil, err
	}
	return template, nil
}

func (s *WorkerChecklistSeed) openForRecentHires(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *checklistSeedRefs,
	template *worker.WorkerChecklistTemplate,
) error {
	workers, err := workerdb.Workers.AllTenants().Load(workerdb.Workers.Profile()).Where(
		workerdb.Workers.OrganizationID.Eq(refs.orgID),
		workerdb.Workers.BusinessUnitID.Eq(refs.buID),
	).Order(workerdb.Workers.CreatedAt.Asc()).Limit(4).List(ctx, db)
	if err != nil {
		return fmt.Errorf("load workers: %w", err)
	}

	for index, wrk := range workers {

		startedAt := refs.now - int64(3+index*4)*checklistSeedDay
		checklist := template.Instantiate(wrk.ID, startedAt, refs.adminID, pulid.Nil)
		if err := workerdb.WorkerChecklists.Create(ctx, db, checklist); err != nil {
			return fmt.Errorf("insert checklist for %s: %w", wrk.ID, err)
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
		if err := workerdb.WorkerChecklistItems.CreateMany(ctx, db, checklist.Items); err != nil {
			return fmt.Errorf("insert checklist items for %s: %w", wrk.ID, err)
		}
	}
	return nil
}
