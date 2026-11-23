package dispatcheligibility

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerclearanceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type ClearanceCheckParams struct {
	fx.In

	ControlRepo dispatchcontrolports.DispatchControlRepository
}

type ClearanceCheck struct {
	controlRepo dispatchcontrolports.DispatchControlRepository
}

func NewClearanceCheck(p ClearanceCheckParams) workerclearanceports.DispatchCheck {
	return &ClearanceCheck{controlRepo: p.ControlRepo}
}

func (c *ClearanceCheck) Verdicts(
	ctx context.Context,
	req *workerclearanceports.DispatchRequest,
) (*workercompliance.DispatchVerdicts, error) {
	control, err := c.controlRepo.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: req.TenantInfo,
	})
	if errortypes.IsNotFoundError(err) {
		control = dispatchcontrol.NewDefaultDispatchControl(
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
		)
	} else if err != nil {
		return nil, fmt.Errorf("read dispatch control for clearance: %w", err)
	}
	return ClearanceVerdicts(req.Worker, control, req.Now, req.Probes), nil
}

func ClearanceVerdicts(
	wrk *worker.Worker,
	control *dispatchcontrol.DispatchControl,
	now int64,
	probes []workercompliance.DispatchProbe,
) *workercompliance.DispatchVerdicts {
	verdicts := &workercompliance.DispatchVerdicts{
		RefusedToday: []workercompliance.DispatchRequirement{},
		Stops:        make([]workercompliance.DispatchStop, 0, len(probes)),
	}
	if wrk == nil || wrk.Profile == nil || control == nil {
		return verdicts
	}

	today := EvaluateWorkerCompliance(WorkerComplianceInput{
		Worker:  wrk,
		Control: control,
		Now:     now,
	})
	for i := range today.Findings {
		if today.Findings[i].Severity == dispatchfinding.SeverityBlock {
			verdicts.RefusedToday = append(
				verdicts.RefusedToday,
				requirementOf(today.Findings[i].Code),
			)
		}
	}

	for _, probe := range probes {
		lapsed, ok := lapsedAt(wrk, probe)
		if !ok {
			continue
		}
		at := probe.At + 1
		general := blockingFinding(EvaluateWorkerCompliance(WorkerComplianceInput{
			Worker:  lapsed,
			Control: control,
			Now:     at,
		}), probe.Requirement)
		if general != nil {
			verdicts.Stops = append(verdicts.Stops, stopOf(probe, general, false))
			continue
		}
		hazmat := blockingFinding(EvaluateWorkerCompliance(WorkerComplianceInput{
			Worker:               lapsed,
			Control:              control,
			HasHazmatCommodities: true,
			Now:                  at,
		}), probe.Requirement)
		if hazmat != nil {
			verdicts.Stops = append(verdicts.Stops, stopOf(probe, hazmat, true))
		}
	}
	return verdicts
}

func stopOf(
	probe workercompliance.DispatchProbe,
	finding *dispatchfinding.Finding,
	hazmatOnly bool,
) workercompliance.DispatchStop {
	return workercompliance.DispatchStop{
		Requirement: probe.Requirement,
		At:          probe.At,
		HazmatOnly:  hazmatOnly,
		Regulation:  finding.Regulation,
	}
}

func blockingFinding(
	eval *Evaluation,
	requirement workercompliance.DispatchRequirement,
) *dispatchfinding.Finding {
	for i := range eval.Findings {
		if eval.Findings[i].Severity == dispatchfinding.SeverityBlock &&
			requirementOf(eval.Findings[i].Code) == requirement {
			return &eval.Findings[i]
		}
	}
	return nil
}

func lapsedAt(wrk *worker.Worker, probe workercompliance.DispatchProbe) (*worker.Worker, bool) {
	profile := *wrk.Profile
	at := probe.At
	switch probe.Requirement {
	case workercompliance.DispatchRequirementLicense:
		profile.LicenseExpiry = at
	case workercompliance.DispatchRequirementMedicalCard:
		profile.MedicalCardExpiry = &at
	case workercompliance.DispatchRequirementPhysical:
		profile.PhysicalDueDate = &at
	case workercompliance.DispatchRequirementMVR:
		profile.MVRDueDate = &at
	case workercompliance.DispatchRequirementHazmat:
		profile.HazmatExpiry = &at
	case workercompliance.DispatchRequirementClearinghouseQuery:
		profile.NextClearinghouseQueryDue = &at
	case workercompliance.DispatchRequirementDrugAlcohol,
		workercompliance.DispatchRequirementOther:
		return nil, false
	default:
		return nil, false
	}
	lapsed := *wrk
	lapsed.Profile = &profile
	return &lapsed, true
}

func requirementOf(code string) workercompliance.DispatchRequirement {
	switch code {
	case dispatchfinding.CodeLicenseExpired:
		return workercompliance.DispatchRequirementLicense
	case dispatchfinding.CodeMedicalCardExpired:
		return workercompliance.DispatchRequirementMedicalCard
	case dispatchfinding.CodePhysicalOverdue:
		return workercompliance.DispatchRequirementPhysical
	case dispatchfinding.CodeMVRDueDatePassed:
		return workercompliance.DispatchRequirementMVR
	case dispatchfinding.CodeHazmatExpired:
		return workercompliance.DispatchRequirementHazmat
	case dispatchfinding.CodeClearinghouseOverdue:
		return workercompliance.DispatchRequirementClearinghouseQuery
	case dispatchfinding.CodeDrugAlcoholProhibited:
		return workercompliance.DispatchRequirementDrugAlcohol
	default:
		return workercompliance.DispatchRequirementOther
	}
}
