package workeremploymentresolver

import (
	"sort"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func recordRequest(
	authCtx *authctx.AuthContext,
	input gqlmodel.RecordWorkerEmploymentEventInput,
) (*workeremploymentservice.RecordRequest, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}
	fleetCodeID, err := optionalIDPtr(input.FleetCodeID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"fleetCodeId",
			errortypes.ErrInvalid,
			"Fleet code is invalid",
		)
	}
	managerID, err := optionalIDPtr(input.ManagerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"managerId",
			errortypes.ErrInvalid,
			"Manager is invalid",
		)
	}

	return &workeremploymentservice.RecordRequest{
		TenantInfo:  resolvercore.TenantInfo(authCtx),
		WorkerID:    workerID,
		Kind:        input.Kind,
		EffectiveAt: int64(input.EffectiveAt),
		Reason:      gqlscalar.StringValue(input.Reason),
		Notes:       gqlscalar.StringValue(input.Notes),
		DocumentID:  documentID,
		FleetCodeID: fleetCodeID,
		ManagerID:   managerID,
		DriverType:  input.DriverType,
		WorkerType:  input.WorkerType,
		Rate:        gqlscalar.StringValue(input.Rate),
		RateUnit:    gqlscalar.StringValue(input.RateUnit),
		LeaveType:   input.LeaveType,
		UserID:      authCtx.UserID,
	}, nil
}

func employmentValues(values map[string]string) []*gqlmodel.WorkerEmploymentValue {
	out := make([]*gqlmodel.WorkerEmploymentValue, 0, len(values))
	for key, value := range values {
		out = append(out, &gqlmodel.WorkerEmploymentValue{Key: key, Value: value})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func optionalIDPtr(value *string) (*pulid.ID, error) {
	if value == nil {
		return nil, nil //nolint:nilnil // absence is meaningful here
	}
	if *value == "" {
		id := pulid.Nil
		return &id, nil
	}
	id, err := pulid.MustParse(*value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
