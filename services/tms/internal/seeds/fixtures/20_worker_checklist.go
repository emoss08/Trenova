package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist/workerchecklistdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential/workercredentialdb"
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
	types, err := workercredentialdb.WorkerCredentialTypes.AllTenants().Where(
		workercredentialdb.WorkerCredentialTypes.OrganizationID.Eq(refs.orgID),
		workercredentialdb.WorkerCredentialTypes.BusinessUnitID.Eq(refs.buID),
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
) *workerchecklist.WorkerChecklistTemplateItem {
	return &workerchecklist.WorkerChecklistTemplateItem{
		Label:            label,
		Kind:             workerchecklist.ChecklistItemCredential,
		Required:         true,
		DueOffsetDays:    due,
		Owner:            workerchecklist.ChecklistOwnerSafety,
		CredentialTypeID: refs.credentialTypes[code],
	}
}

func (s *WorkerChecklistSeed) onboardingTemplate(
	refs *checklistSeedRefs,
) *workerchecklist.WorkerChecklistTemplate {
	items := []*workerchecklist.WorkerChecklistTemplateItem{
		s.credentialItem(refs, "CDL", "CDL copied to the driver qualification file", 1),
		s.credentialItem(refs, "MED_CARD", "Medical examiner's certificate on file", 3),
		s.credentialItem(refs, "MVR", "Motor vehicle record pulled and reviewed", 5),
		{
			Label:         "Pre-employment drug & alcohol test result",
			Description:   "Negative result received from the collection site.",
			Kind:          workerchecklist.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 5,
			Owner:         workerchecklist.ChecklistOwnerSafety,
		},
		{
			Label:         "Road test and evaluation signed off",
			Kind:          workerchecklist.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 7,
			Owner:         workerchecklist.ChecklistOwnerSafety,
		},
		{
			Label:         "Employee handbook acknowledged",
			Kind:          workerchecklist.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 7,
			Owner:         workerchecklist.ChecklistOwnerHR,
		},
		{
			Label:         "Direct deposit and tax forms collected",
			Kind:          workerchecklist.ChecklistItemTask,
			Required:      true,
			DueOffsetDays: 3,
			Owner:         workerchecklist.ChecklistOwnerPayroll,
		},
		{
			Label:         "Fuel card issued",
			Kind:          workerchecklist.ChecklistItemEquipment,
			Required:      true,
			DueOffsetDays: 2,
			Owner:         workerchecklist.ChecklistOwnerFleet,
		},
		{
			Label:         "ELD tablet and login provisioned",
			Kind:          workerchecklist.ChecklistItemEquipment,
			Required:      true,
			DueOffsetDays: 2,
			Owner:         workerchecklist.ChecklistOwnerIT,
		},
		{
			Label:         "Invited to Dash",
			Description:   "Completes itself once the driver has portal access.",
			Kind:          workerchecklist.ChecklistItemPortalAccess,
			Required:      false,
			DueOffsetDays: 1,
			Owner:         workerchecklist.ChecklistOwnerDispatch,
		},
	}
	if len(refs.documentTypes) > 0 {
		items = append(items, &workerchecklist.WorkerChecklistTemplateItem{
			Label:          "Signed employment application on file",
			Kind:           workerchecklist.ChecklistItemDocument,
			Required:       false,
			DueOffsetDays:  3,
			Owner:          workerchecklist.ChecklistOwnerHR,
			DocumentTypeID: refs.documentTypes[0].ID,
		})
	}
	return &workerchecklist.WorkerChecklistTemplate{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		Code:           checklistOnboardingCode,
		Name:           "Driver onboarding",
		Description:    "Everything a new driver needs before their first dispatch.",
		Kind:           workerchecklist.ChecklistKindOnboarding,
		Trigger:        workerchecklist.ChecklistTriggerHired,
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		Items:          items,
	}
}

func (s *WorkerChecklistSeed) offboardingTemplate(
	refs *checklistSeedRefs,
) *workerchecklist.WorkerChecklistTemplate {
	return &workerchecklist.WorkerChecklistTemplate{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		Code:           checklistOffboardingCode,
		Name:           "Driver offboarding",
		Description:    "Close out equipment, access and pay when a driver leaves.",
		Kind:           workerchecklist.ChecklistKindOffboarding,
		Trigger:        workerchecklist.ChecklistTriggerTerminated,
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		Items: []*workerchecklist.WorkerChecklistTemplateItem{
			{
				Label:         "Fuel card returned and cancelled",
				Kind:          workerchecklist.ChecklistItemEquipment,
				Required:      true,
				DueOffsetDays: 2,
				Owner:         workerchecklist.ChecklistOwnerFleet,
			},
			{
				Label:         "ELD tablet returned",
				Kind:          workerchecklist.ChecklistItemEquipment,
				Required:      true,
				DueOffsetDays: 2,
				Owner:         workerchecklist.ChecklistOwnerIT,
			},
			{
				Label:         "Dash access revoked",
				Description:   "Completes itself once portal access is removed.",
				Kind:          workerchecklist.ChecklistItemPortalAccess,
				Required:      true,
				DueOffsetDays: 1,
				Owner:         workerchecklist.ChecklistOwnerDispatch,
			},
			{
				Label:         "Final settlement processed",
				Kind:          workerchecklist.ChecklistItemTask,
				Required:      true,
				DueOffsetDays: 14,
				Owner:         workerchecklist.ChecklistOwnerPayroll,
			},
			{
				Label:         "Exit interview completed",
				Kind:          workerchecklist.ChecklistItemTask,
				Required:      false,
				DueOffsetDays: 7,
				Owner:         workerchecklist.ChecklistOwnerHR,
			},
		},
	}
}

func (s *WorkerChecklistSeed) ensureTemplate(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	template *workerchecklist.WorkerChecklistTemplate,
) (*workerchecklist.WorkerChecklistTemplate, error) {
	existing, err := workerchecklistdb.WorkerChecklistTemplates.AllTenants().
		Load(workerchecklistdb.WorkerChecklistTemplates.Items()).
		Where(
			workerchecklistdb.WorkerChecklistTemplates.OrganizationID.Eq(template.OrganizationID),
			workerchecklistdb.WorkerChecklistTemplates.BusinessUnitID.Eq(template.BusinessUnitID),
			workerchecklistdb.WorkerChecklistTemplates.Code.Eq(template.Code),
		).
		First(ctx, db)
	if err == nil {
		return existing, nil
	}

	if err = workerchecklistdb.WorkerChecklistTemplates.Create(ctx, db, template); err != nil {
		return nil, err
	}
	for i, item := range template.Items {
		item.TemplateID = template.ID
		item.OrganizationID = template.OrganizationID
		item.BusinessUnitID = template.BusinessUnitID
		item.SortOrder = int32(i) //nolint:gosec // item counts are tiny
	}
	if err = workerchecklistdb.WorkerChecklistTemplateItems.CreateMany(ctx, db, template.Items); err != nil {
		return nil, err
	}
	return template, nil
}

func (s *WorkerChecklistSeed) openForRecentHires(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *checklistSeedRefs,
	template *workerchecklist.WorkerChecklistTemplate,
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
		if err := workerchecklistdb.WorkerChecklists.Create(ctx, db, checklist); err != nil {
			return fmt.Errorf("insert checklist for %s: %w", wrk.ID, err)
		}
		for i, item := range checklist.Items {
			item.ChecklistID = checklist.ID
			item.OrganizationID = checklist.OrganizationID
			item.BusinessUnitID = checklist.BusinessUnitID
			item.SortOrder = int32(i) //nolint:gosec // item counts are tiny
			if item.Kind == workerchecklist.ChecklistItemTask && i%2 == index%2 {
				completedAt := startedAt + checklistSeedDay
				item.Status = workerchecklist.ChecklistItemDone
				item.CompletedAt = &completedAt
				item.CompletedByID = refs.adminID
			}
		}
		if err := workerchecklistdb.WorkerChecklistItems.CreateMany(ctx, db, checklist.Items); err != nil {
			return fmt.Errorf("insert checklist items for %s: %w", wrk.ID, err)
		}
	}
	return nil
}
