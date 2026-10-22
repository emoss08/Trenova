//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/performancereviewrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerchecklistrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workercredentialrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerdqfrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerdrugalcoholrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerinjuryrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerleaverepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workersafetyrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workertrainingrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerchecklistservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const keyedRowsPerWorker = 3

// keyedCase is one list of a worker's own records read against real
// repositories: how to write a record for a worker, the service read the
// repository tool made, and the templates it read alongside.
type keyedCase struct {
	tool string
	// openArgs are the arguments that read only the open records (a list with
	// none has no closed ones), allArgs those that read every record, and
	// defaultOpen marks a list that reads only the open ones by default.
	openArgs    map[string]any
	allArgs     map[string]any
	defaultOpen bool
	// orgWide marks a list that reads every worker's records when it names
	// no worker, and openByDefaultUnscoped one that then reads only the open
	// ones unless asked for every one.
	orgWide               bool
	openByDefaultUnscoped bool
	// restricted are fields a record carries that an agent at the default
	// data access (Internal) is never shown, and internal those it is.
	restricted []string
	internal   []string
	// nested is a list every record carries rows of (an event's violations).
	nested string
	write  func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error)
	before func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error)
	// templates writes a record the list reads alongside under key, and one
	// it must leave out, and returns the first's id. keyed marks a read
	// alongside keyed by the worker the call names, made only when it names
	// one.
	key       string
	keyed     bool
	templates func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string) (pulid.ID, error)
	// activeTemplates counts what the read alongside should return, where
	// the tenant holds more than the test wrote (the system credential
	// types every organization is seeded with).
	activeTemplates func(ctx context.Context, tenant pagination.TenantInfo) (int, error)
}

//nolint:funlen // one entry per keyed list
func keyedListCases(services *agentreadServices) []keyedCase {
	now := timeutils.NowUnix()
	var caseNumber int32
	tenantOf := func(tenant *agentreadtest.Tenant) (pulid.ID, pulid.ID) {
		return tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	}
	return []keyedCase{
		{
			tool: "worker_list_checklists", key: "templatesStartedByHand",
			openArgs: map[string]any{}, allArgs: includeClosed, defaultOpen: true,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status, completedAt := worker.ChecklistStatusOpen, (*int64)(nil)
				if closed {
					status, completedAt = worker.ChecklistStatusCompleted, &now
				}
				row, err := services.checklists.Create(ctx, &worker.WorkerChecklist{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					Name: "Checklist " + label, Kind: worker.ChecklistKindOnboarding,
					Status: status, StartedAt: now, CompletedAt: completedAt,
					Items: []*worker.WorkerChecklistItem{{
						Label: "Sign the handbook", Kind: worker.ChecklistItemTask,
						Required: true, Owner: worker.ChecklistOwnerHR,
						Status: worker.ChecklistItemPending, Note: "ask " + label,
					}},
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error) {
				rows, err := services.checklistService.ListForWorker(ctx, tenant, workerID, includeClosed)
				return idsOf(rows, err)
			},
			templates: func(ctx context.Context, tenant *agentreadtest.Tenant, _ pulid.ID, label string) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				manual, err := services.checklists.CreateTemplate(ctx, &worker.WorkerChecklistTemplate{
					BusinessUnitID: bu, OrganizationID: org, Code: "M" + label, Name: "Manual " + label,
					Kind: worker.ChecklistKindCustom, Trigger: worker.ChecklistTriggerManual,
					Status: domaintypes.StatusActive,
				})
				if err != nil {
					return pulid.Nil, err
				}
				if _, err = services.checklists.CreateTemplate(ctx, &worker.WorkerChecklistTemplate{
					BusinessUnitID: bu, OrganizationID: org, Code: "R" + label, Name: "Retired " + label,
					Kind: worker.ChecklistKindCustom, Trigger: worker.ChecklistTriggerManual,
					Status: domaintypes.StatusInactive,
				}); err != nil {
					return pulid.Nil, err
				}
				return manual.ID, nil
			},
		},
		{
			tool:     "worker_list_training",
			openArgs: map[string]any{}, allArgs: includeClosed, defaultOpen: true,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				course, err := services.training.CreateCourse(ctx, &worker.TrainingCourse{
					BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive,
					Code: "C" + label, Name: "Course " + label, Category: worker.TrainingCategorySafety,
					Delivery: worker.TrainingDeliveryOnline, DurationMinutes: 30,
				})
				if err != nil {
					return pulid.Nil, err
				}
				status := worker.TrainingStatusAssigned
				if closed {
					status = worker.TrainingStatusWaived
				}
				row, err := services.training.Create(ctx, &worker.WorkerTrainingRecord{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					CourseID: course.ID, Status: status, AssignedAt: now,
					WaivedReason: waivedReason(closed),
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error) {
				rows, err := services.trainingService.ListForWorker(ctx, tenant, workerID, includeClosed)
				return idsOf(rows, err)
			},
		},
		{
			tool: "worker_list_performance_reviews", key: "templates",
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, _ bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				template, err := services.reviews.CreateTemplate(ctx, reviewTemplate(tenant, "W"+label, domaintypes.StatusActive))
				if err != nil {
					return pulid.Nil, err
				}
				row, err := services.reviews.CreateReview(ctx, &worker.PerformanceReview{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					TemplateID: template.ID, Status: worker.ReviewStatusDraft,
					Title: "Review " + label, PeriodStart: now - 86400*90, PeriodEnd: now,
					Summary: "Steady " + label,
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, _ bool) ([]string, error) {
				rows, err := services.reviewService.ListReviews(ctx, tenant, workerID, nil)
				return idsOf(rows, err)
			},
			templates: func(ctx context.Context, tenant *agentreadtest.Tenant, _ pulid.ID, label string) (pulid.ID, error) {
				active, err := services.reviews.CreateTemplate(ctx, reviewTemplate(tenant, "A"+label, domaintypes.StatusActive))
				if err != nil {
					return pulid.Nil, err
				}
				if _, err = services.reviews.CreateTemplate(ctx,
					reviewTemplate(tenant, "I"+label, domaintypes.StatusInactive)); err != nil {
					return pulid.Nil, err
				}
				return active.ID, nil
			},
		},
		{
			tool: "worker_list_injuries", orgWide: true,
			openArgs: map[string]any{"openOnly": true}, allArgs: map[string]any{},
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status := worker.InjuryCaseOpen
				if closed {
					status = worker.InjuryCaseClosed
				}
				caseNumber++
				row, err := services.injuries.CreateInjury(ctx, &worker.WorkerInjury{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					CaseNumber: caseNumber, CaseYear: 2026, Status: status,
					Classification: worker.CaseOtherRecordable, OccurredAt: now,
					Description: "Strained back lifting a tarp " + label, BodyPart: "Lower back",
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error) {
				rows, err := services.injuryService.ListInjuries(ctx, &workerports.ListWorkerInjuriesRequest{
					TenantInfo: tenant, WorkerID: workerID, OpenOnly: !includeClosed,
				})
				return idsOf(rows, err)
			},
		},
		{
			tool: "worker_list_dot_tests", orgWide: true,
			openArgs: map[string]any{"openOnly": true}, allArgs: map[string]any{},
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status := worker.DOTTestStatusScheduled
				if closed {
					status = worker.DOTTestStatusCancelled
				}
				row, err := services.tests.CreateTest(ctx, &worker.WorkerDOTTest{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					TestType: worker.DOTTestRandom, Substance: worker.DOTSubstanceDrug,
					Status: status, Result: worker.DOTResultPending, IsDOT: true,
					Reason: "Selected in the random draw " + label,
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error) {
				rows, err := services.testService.ListTests(ctx, &workerports.ListWorkerDOTTestsRequest{
					TenantInfo: tenant, WorkerID: workerID, OpenOnly: !includeClosed,
				})
				return idsOf(rows, err)
			},
		},
		{
			tool: "worker_list_leave_cases", orgWide: true, restricted: []string{"reason"},
			openArgs: map[string]any{"openOnly": true}, allArgs: map[string]any{},
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status, closedAt := worker.LeaveCaseApproved, (*int64)(nil)
				if closed {
					status, closedAt = worker.LeaveCaseClosed, &now
				}
				row, err := services.leave.CreateCase(ctx, &worker.WorkerLeaveCase{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					LeaveType: worker.LeaveTypeMedical, Status: status,
					Frequency: worker.LeaveContinuous, Reason: "Knee surgery " + label,
					RequestedAt: now, StartsAt: now, DecidedAt: &now, ClosedAt: closedAt,
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error) {
				rows, err := services.leaveService.ListCases(ctx, &workerports.ListLeaveCasesRequest{
					TenantInfo: tenant, WorkerID: workerID, OpenOnly: !includeClosed,
				})
				return idsOf(rows, err)
			},
		},
		{
			tool: "worker_list_employment_verifications", orgWide: true,
			restricted: []string{"findings", "hadDrugAlcoholViolations"},
			openArgs:   map[string]any{"outstandingOnly": true}, allArgs: map[string]any{},
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status, received := worker.VerificationRequested, (*int64)(nil)
				if closed {
					status, received = worker.VerificationReceived, &now
				}
				row, err := services.dqf.CreateVerification(ctx, &worker.WorkerEmploymentVerification{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					EmployerName: "Prior Freight " + label, WasDOTRegulated: true, Status: status,
					Method: worker.VerificationByEmail, Findings: "No accidents reported " + label,
					RequestedAt: &now, ResponseReceivedAt: received,
					ContactPhone: "555-0100",
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error) {
				rows, err := services.dqfService.ListVerifications(ctx,
					&workerports.ListEmploymentVerificationsRequest{
						TenantInfo: tenant, WorkerID: workerID, OutstandingOnly: !includeClosed,
					})
				return idsOf(rows, err)
			},
		},
		{
			tool: "worker_list_safety_events", orgWide: true, openByDefaultUnscoped: true,
			nested: "violations",
			key:    "recognitions", keyed: true,
			openArgs: map[string]any{"openOnly": true}, allArgs: map[string]any{"openOnly": false},
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status, closedAt := worker.SafetyEventStatusOpen, (*int64)(nil)
				if closed {
					status, closedAt = worker.SafetyEventStatusClosed, &now
				}
				event, err := services.safety.CreateEvent(ctx, &worker.WorkerSafetyEvent{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					Kind: worker.SafetyEventInspection, Severity: worker.SafetySeverityMinor,
					Status: status, OccurredAt: now, Points: 2, ClosedAt: closedAt,
					Description: "Level 2 at the Gary scale " + label,
				})
				if err != nil {
					return pulid.Nil, err
				}
				_, err = services.safety.CreateViolation(ctx, &worker.WorkerSafetyViolation{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					SafetyEventID: event.ID, Basic: worker.BasicVehicleMaintenance,
					Code: "393.9", Description: "Inoperable lamp " + label, SeverityWeight: 2,
				})
				return created(event, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeClosed bool) ([]string, error) {
				rows, err := services.safetyService.ListEvents(ctx, &workerports.ListWorkerSafetyEventsRequest{
					TenantInfo: tenant, WorkerID: workerID, OpenOnly: !includeClosed,
				})
				return idsOf(rows, err)
			},
			templates: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := services.safety.CreateRecognition(ctx, &worker.WorkerRecognition{
					BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
					Kind: worker.RecognitionKindTenure, Title: "Ten years " + label,
					OccurredAt: now, VisibleToWorker: true,
				})
				return created(row, err)
			},
		},
		{
			tool: "worker_list_credentials", key: "credentialTypes", openByDefaultUnscoped: true,
			openArgs: map[string]any{}, allArgs: map[string]any{"includeArchived": true},
			defaultOpen: true,
			internal: []string{
				"workerId", "credentialTypeId", "status", "issuingAuthority", "issuedAt",
				"expiresAt", "daysUntilExpiry", "health",
			},
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, workerID pulid.ID, label string, closed bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				kind, err := services.credentials.CreateType(ctx, credentialType(bu, org, "K"+label))
				if err != nil {
					return pulid.Nil, err
				}
				issued, expires := now-86400*300, now+86400*10
				status, archivedAt := worker.CredentialStatusActive, (*int64)(nil)
				if closed {
					status, archivedAt = worker.CredentialStatusArchived, &now
				}
				row, err := services.credentials.Create(ctx, &workerports.CreateWorkerCredentialRequest{
					Entity: &worker.WorkerCredential{
						BusinessUnitID: bu, OrganizationID: org, WorkerID: workerID,
						CredentialTypeID: kind.ID, Status: status, Number: "D-" + label,
						IssuingAuthority: "IL", IssuedAt: &issued, ExpiresAt: &expires,
						ArchivedAt: archivedAt, Notes: "Renewal booked " + label,
					},
				})
				return created(row, err)
			},
			before: func(ctx context.Context, tenant pagination.TenantInfo, workerID pulid.ID, includeArchived bool) ([]string, error) {
				rows, err := services.credentialService.List(ctx, &workercredentialservice.ListRequest{
					TenantInfo: tenant, WorkerID: workerID, IncludeArchived: includeArchived,
				})
				return idsOf(rows, err)
			},
			templates: func(ctx context.Context, tenant *agentreadtest.Tenant, _ pulid.ID, label string) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				kind, err := services.credentials.CreateType(ctx, credentialType(bu, org, "T"+label))
				return created(kind, err)
			},
			activeTemplates: func(ctx context.Context, tenant pagination.TenantInfo) (int, error) {
				types, err := services.credentials.ListActiveTypes(ctx, tenant)
				return len(types), err
			},
		},
	}
}

// credentialType is an active, required kind of credential a worker holds.
func credentialType(bu, org pulid.ID, code string) *worker.WorkerCredentialType {
	return &worker.WorkerCredentialType{
		BusinessUnitID: bu, OrganizationID: org, Code: code, Name: "Credential " + code,
		Category: worker.CredentialCategoryOther, Status: domaintypes.StatusActive,
		IsRequired: true, RenewalWindowDays: 30,
	}
}

func waivedReason(closed bool) string {
	if closed {
		return "Covered by a prior employer's course"
	}

	return ""
}

func reviewTemplate(tenant *agentreadtest.Tenant, code string, status domaintypes.Status) *worker.PerformanceReviewTemplate {
	return &worker.PerformanceReviewTemplate{
		BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
		Code: code, Name: "Template " + code, Status: status,
		Items: []worker.ReviewItem{{Key: "safety", Label: "Safety", Weight: 1}},
	}
}

func idsOf[T interface{ GetID() pulid.ID }](rows []T, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.GetID().String())
	}

	return out, nil
}

// agentreadServices are the repositories and services the keyed lists read
// through, and the old tools read.
type agentreadServices struct {
	checklists        workerChecklistRepo
	training          workerTrainingRepo
	reviews           reviewRepo
	injuries          injuryRepo
	tests             dotTestRepo
	checklistService  *workerchecklistservice.Service
	trainingService   *workertrainingservice.Service
	reviewService     *performancereviewservice.Service
	injuryService     *workerinjuryservice.Service
	testService       *workerdrugalcoholservice.Service
	leave             leaveRepo
	dqf               dqfRepo
	leaveService      *workerleaveservice.Service
	dqfService        *workerdqfservice.Service
	safety            workerports.WorkerSafetyRepository
	safetyService     *workersafetyservice.Service
	credentials       workerports.WorkerCredentialRepository
	credentialService *workercredentialservice.Service
}

type (
	workerChecklistRepo = interface {
		Create(context.Context, *worker.WorkerChecklist) (*worker.WorkerChecklist, error)
		CreateTemplate(context.Context, *worker.WorkerChecklistTemplate) (*worker.WorkerChecklistTemplate, error)
	}
	workerTrainingRepo = interface {
		Create(context.Context, *worker.WorkerTrainingRecord) (*worker.WorkerTrainingRecord, error)
		CreateCourse(context.Context, *worker.TrainingCourse) (*worker.TrainingCourse, error)
	}
	injuryRepo = interface {
		CreateInjury(context.Context, *worker.WorkerInjury) (*worker.WorkerInjury, error)
	}
	dotTestRepo = interface {
		CreateTest(context.Context, *worker.WorkerDOTTest) (*worker.WorkerDOTTest, error)
	}
	leaveRepo = interface {
		CreateCase(context.Context, *worker.WorkerLeaveCase) (*worker.WorkerLeaveCase, error)
	}
	dqfRepo = interface {
		CreateVerification(
			context.Context, *worker.WorkerEmploymentVerification,
		) (*worker.WorkerEmploymentVerification, error)
	}
	reviewRepo = interface {
		CreateReview(context.Context, *worker.PerformanceReview) (*worker.PerformanceReview, error)
		CreateTemplate(context.Context, *worker.PerformanceReviewTemplate) (*worker.PerformanceReviewTemplate, error)
	}
)

type keyedTenant struct {
	ctx      context.Context
	info     pagination.TenantInfo
	workerID pulid.ID
	open     []string
	all      []string
	template pulid.ID
}

func writeKeyedRows(
	t *testing.T,
	c *keyedCase,
	tenant *agentreadtest.Tenant,
	workers interface {
		Create(context.Context, *worker.Worker) (*worker.Worker, error)
	},
	label string,
) *keyedTenant {
	t.Helper()

	out := &keyedTenant{
		ctx: readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil),
		info: pagination.TenantInfo{
			OrgID: tenant.Actor.OrganizationID, BuID: tenant.Actor.BusinessUnitID,
			UserID: tenant.Actor.UserID,
		},
	}
	driver, err := workers.Create(out.ctx, testWorker(tenant, "K"+label, false, pulid.Nil))
	require.NoError(t, err, c.tool)
	out.workerID = driver.ID
	for idx := range keyedRowsPerWorker {
		closed := idx == keyedRowsPerWorker-1
		id, writeErr := c.write(out.ctx, tenant, driver.ID, fmt.Sprintf("%s%d", label, idx), closed)
		require.NoError(t, writeErr, c.tool)
		out.all = append(out.all, id.String())
		if !closed || c.openArgs == nil {
			out.open = append(out.open, id.String())
		}
	}
	if c.templates != nil {
		out.template, err = c.templates(out.ctx, tenant, driver.ID, label)
		require.NoError(t, err, c.tool)
	}

	return out
}

func TestConvertedLists_KeyedListsReadWhatTheRepositoryToolsReadUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	_, resolvers := convertedListCases(conn)
	workers := workerrepository.New(workerrepository.Params{DB: conn, Logger: logger})
	checklists := workerchecklistrepository.New(workerchecklistrepository.Params{DB: conn, Logger: logger})
	training := workertrainingrepository.New(workertrainingrepository.Params{DB: conn, Logger: logger})
	reviews := performancereviewrepository.New(performancereviewrepository.Params{DB: conn, Logger: logger})
	services := &agentreadServices{
		checklists: checklists, training: training, reviews: reviews,
		checklistService: workerchecklistservice.New(workerchecklistservice.Params{
			Logger: logger, Repo: checklists, WorkerRepo: workers,
			CredentialRepo: workercredentialrepository.New(
				workercredentialrepository.Params{DB: conn, Logger: logger}),
			DocumentRepo: documentrepository.New(documentrepository.Params{DB: conn, Logger: logger}),
		}),
		trainingService: resolvers.WorkerTrainingService,
		reviewService: performancereviewservice.New(performancereviewservice.Params{
			Logger: logger, Repo: reviews, WorkerRepo: workers,
		}),
	}
	injuries := workerinjuryrepository.New(workerinjuryrepository.Params{DB: conn, Logger: logger})
	tests := workerdrugalcoholrepository.New(workerdrugalcoholrepository.Params{DB: conn, Logger: logger})
	services.injuries, services.tests = injuries, tests
	services.injuryService = workerinjuryservice.New(workerinjuryservice.Params{
		Logger: logger, Repo: injuries, WorkerRepo: workers,
	})
	services.testService = workerdrugalcoholservice.New(workerdrugalcoholservice.Params{
		Logger: logger, Repo: tests, WorkerRepo: workers,
	})
	resolvers.WorkerChecklistService = services.checklistService
	resolvers.PerformanceReviewService = services.reviewService
	leave := workerleaverepository.New(workerleaverepository.Params{DB: conn, Logger: logger})
	dqf := workerdqfrepository.New(workerdqfrepository.Params{DB: conn, Logger: logger})
	services.leave, services.dqf = leave, dqf
	services.leaveService = workerleaveservice.New(workerleaveservice.Params{
		Logger: logger, Repo: leave, WorkerRepo: workers,
	})
	services.dqfService = workerdqfservice.New(workerdqfservice.Params{
		Logger: logger, Repo: dqf, WorkerRepo: workers,
	})
	resolvers.WorkerLeaveService = services.leaveService
	resolvers.WorkerDQFService = services.dqfService
	resolvers.WorkerInjuryService = services.injuryService
	resolvers.WorkerDrugAlcoholService = services.testService
	services.safety = workersafetyrepository.New(workersafetyrepository.Params{DB: conn, Logger: logger})
	services.safetyService = workersafetyservice.New(workersafetyservice.Params{
		Logger: logger, Repo: services.safety, WorkerRepo: workers,
	})
	resolvers.WorkerSafetyService = services.safetyService
	services.credentials = workercredentialrepository.New(
		workercredentialrepository.Params{DB: conn, Logger: logger})
	services.credentialService = workercredentialservice.New(workercredentialservice.Params{
		Logger: logger, Repo: services.credentials, WorkerRepo: workers,
	})
	resolvers.WorkerCredentialService = services.credentialService

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: resolvers})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: resolvers})
	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")

	for _, c := range keyedListCases(services) {
		t.Run(c.tool, func(t *testing.T) {
			converted, err := agentquerytoolservice.ConvertedListTool(c.tool, reader, engine)
			require.NoError(t, err)
			ours := writeKeyedRows(t, &c, mine, workers, "A")
			other := writeKeyedRows(t, &c, theirs, workers, "B")
			actor := mine.Actor
			call := func(args map[string]any) map[string]any {
				return callTool(t, ours.ctx, converted, actor, args)
			}

			t.Run("what a call reads by default, as the repository tool read it", func(t *testing.T) {
				page := call(withWorker(ours.workerID, nil))
				want := ours.all
				if c.defaultOpen {
					want = ours.open
				}
				before, beforeErr := c.before(ours.ctx, ours.info, ours.workerID, !c.defaultOpen)
				require.NoError(t, beforeErr)
				assert.ElementsMatch(t, want, itemIDs(page))
				assert.ElementsMatch(t, before, itemIDs(page))
				assert.NotContains(t, page, "hasMore", "an unpaged list never pages")
			})

			if c.openArgs != nil {
				t.Run("the open records, and every record", func(t *testing.T) {
					open := call(withWorker(ours.workerID, c.openArgs))
					before, beforeErr := c.before(ours.ctx, ours.info, ours.workerID, false)
					require.NoError(t, beforeErr)
					assert.ElementsMatch(t, ours.open, itemIDs(open))
					assert.ElementsMatch(t, before, itemIDs(open))

					all := call(withWorker(ours.workerID, c.allArgs))
					before, beforeErr = c.before(ours.ctx, ours.info, ours.workerID, true)
					require.NoError(t, beforeErr)
					assert.ElementsMatch(t, ours.all, itemIDs(all))
					assert.ElementsMatch(t, before, itemIDs(all))
				})
			}

			t.Run("another tenant's worker reads nothing", func(t *testing.T) {
				page := call(withWorker(other.workerID, c.allArgs))
				assert.Empty(t, itemIDs(page))
			})

			if c.orgWide {
				t.Run("no worker reads every worker's, and nothing of another tenant", func(t *testing.T) {
					page := call(withWorker(pulid.Nil, c.allArgs))
					assert.ElementsMatch(t, ours.all, itemIDs(page))
					if c.keyed {
						assert.NotContains(t, page, c.key, "a read keyed by the worker needs one")
					}
				})

			}

			if c.templates != nil {
				t.Run("what is read alongside is ours alone, and only what it should be", func(t *testing.T) {
					page := call(map[string]any{"workerId": ours.workerID.String()})
					listed := page[c.key].([]any)
					ids := make([]string, 0, len(listed))
					for _, row := range listed {
						ids = append(ids, row.(map[string]any)["id"].(string))
					}
					assert.Contains(t, ids, ours.template.String())
					assert.NotContains(t, ids, other.template.String())
					want := 1 + keyedTemplatesFor(c.tool)
					if c.activeTemplates != nil {
						var countErr error
						want, countErr = c.activeTemplates(ours.ctx, ours.info)
						require.NoError(t, countErr)
					}
					assert.Len(t, ids, want, "only the active ones")
				})
			}

			if c.openByDefaultUnscoped {
				t.Run("no worker reads only the open records by default, and nothing of another tenant", func(t *testing.T) {
					page := call(map[string]any{})
					assert.ElementsMatch(t, ours.open, itemIDs(page))
					if c.key != "" && !c.keyed {
						assert.NotContains(t, page, c.key, "what is read beside one worker's needs one")
					}
				})
			}

			if c.tool == "worker_list_credentials" {
				t.Run("what falls due is walked page by page, with no repeats", func(t *testing.T) {
					var seen []string
					args := map[string]any{"limit": float64(1)}
					for range len(ours.open) + 1 {
						page := call(args)
						seen = append(seen, itemIDs(page)...)
						cursor, _ := page["nextCursor"].(string)
						if cursor == "" {
							break
						}
						args = map[string]any{"limit": float64(1), "after": cursor}
					}
					assert.ElementsMatch(t, ours.open, seen)
				})
			}

			if c.nested != "" {
				t.Run("each record carries its "+c.nested, func(t *testing.T) {
					page := call(withWorker(ours.workerID, c.allArgs))
					items, _ := page["items"].([]any)
					require.Len(t, items, len(ours.all))
					for _, item := range items {
						assert.NotEmpty(t, item.(map[string]any)[c.nested])
					}
				})
			}

			if len(c.internal) > 0 {
				t.Run("what is Internal reaches an agent at the default data access", func(t *testing.T) {
					data, queryErr := converted.Query(ours.ctx, &agentports.QueryToolParams{
						OrganizationID:    actor.OrganizationID,
						BusinessUnitID:    actor.BusinessUnitID,
						Actor:             actor,
						Timezone:          "America/Chicago",
						Params:            withWorker(ours.workerID, c.allArgs),
						RunID:             pulid.MustNew("ar_"),
						DataAccessCeiling: permission.SensitivityInternal,
					})
					require.NoError(t, queryErr)
					items, _ := asMap(t, data)["items"].([]any)
					require.Len(t, items, len(ours.all))
					for _, item := range items {
						for _, field := range c.internal {
							assert.Contains(t, item.(map[string]any), field)
						}
					}
				})
			}

			if len(c.restricted) > 0 {
				t.Run("what stays Restricted never reaches an agent at Internal", func(t *testing.T) {
					data, queryErr := converted.Query(ours.ctx, &agentports.QueryToolParams{
						OrganizationID:    actor.OrganizationID,
						BusinessUnitID:    actor.BusinessUnitID,
						Actor:             actor,
						Timezone:          "America/Chicago",
						Params:            withWorker(ours.workerID, c.allArgs),
						RunID:             pulid.MustNew("ar_"),
						DataAccessCeiling: permission.SensitivityInternal,
					})
					require.NoError(t, queryErr)
					page := asMap(t, data)
					withheld, _ := page["withheldByAccess"].([]any)
					items, _ := page["items"].([]any)
					require.Len(t, items, len(ours.all))
					for _, field := range c.restricted {
						assert.Contains(t, withheld, field)
						for _, item := range items {
							assert.NotContains(t, item.(map[string]any), field)
						}
					}
				})
			}

			t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
				hidden, hiddenErr := agentquerytoolservice.ConvertedListTool(c.tool, public, &publicRole{})
				require.NoError(t, hiddenErr)
				page := callTool(t, ours.ctx, hidden, actor, withWorker(ours.workerID, c.allArgs))
				withheld, _ := page["withheldByAccess"].([]any)
				require.NotEmpty(t, withheld)
				items, _ := page["items"].([]any)
				require.Len(t, items, len(ours.all))
				for _, item := range items {
					row, _ := item.(map[string]any)
					assert.Contains(t, row, "id")
					for _, field := range withheld {
						assert.NotContains(t, row, field)
					}
				}
			})
		})
	}
}

var includeClosed = map[string]any{"includeClosed": true}

// withWorker is a call's arguments with the worker it names, or none.
func withWorker(workerID pulid.ID, args map[string]any) map[string]any {
	out := make(map[string]any, len(args)+1)
	for key, value := range args {
		out[key] = value
	}
	if workerID.IsNotNil() {
		out["workerId"] = workerID.String()
	}

	return out
}

// keyedTemplatesFor is how many active templates a list's records wrote
// besides the one written for it: every performance review needs a template
// of its own.
func keyedTemplatesFor(tool string) int {
	if tool == "worker_list_performance_reviews" {
		return keyedRowsPerWorker
	}

	return 0
}
