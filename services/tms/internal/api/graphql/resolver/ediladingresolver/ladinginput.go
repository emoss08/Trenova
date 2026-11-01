package ediladingresolver

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const maxListLimit = 100

func listLimit(limit *int) int {
	if limit == nil || *limit <= 0 || *limit > maxListLimit {
		return maxListLimit
	}

	return *limit
}

type versionActionFunc func(
	context.Context,
	*edilading.VersionRequest,
	*permissionports.RequestActor,
) (*ladingproject.LadingVersion, error)

func (r *MutationResolver) versionAction(
	ctx context.Context,
	id string,
	action versionActionFunc,
) (*ladingproject.LadingVersion, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceEDI, permission.OpUpdate)
	if err != nil {
		return nil, err
	}
	versionID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return action(ctx, &edilading.VersionRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		VersionID:  versionID,
	}, actorutil.FromAuthContext(authCtx))
}

func runInputFromGraphQL(input *gqlmodel.EDILadingRunInput) (lading.RunInput, error) {
	runInput := lading.RunInput{X12: input.X12, CSV: input.CSV}
	var err error
	if runInput.JSON, err = rawJSON("input.json", input.JSON); err != nil {
		return lading.RunInput{}, err
	}
	if runInput.Document, err = rawJSON("input.document", input.Document); err != nil {
		return lading.RunInput{}, err
	}

	return runInput, nil
}

func rawJSON(field string, value any) (jsonutils.RawJSON, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Input is not valid JSON",
		)
	}

	return encoded, nil
}

func createProjectRequest(
	input *gqlmodel.CreateEDILadingProjectInput,
) (*edilading.CreateProjectRequest, error) {
	partnerID, err := gqlid.ParseIDField("partnerId", input.PartnerID)
	if err != nil {
		return nil, err
	}
	req := &edilading.CreateProjectRequest{
		PartnerID:   partnerID,
		Role:        input.Role,
		Sets:        input.Sets,
		Separators:  input.Separators,
		Usage:       gqlscalar.StringValue(input.Usage),
		Acknowledge: gqlscalar.StringValue(input.Acknowledge),
		AckWithin:   gqlscalar.IntValue(input.AckWithin),
	}
	pairs := []struct {
		field string
		raw   []string
		dest  **[2]string
	}{
		{field: "them", raw: input.Them, dest: &req.Them},
		{field: "us", raw: input.Us, dest: &req.Us},
		{field: "group", raw: input.Group, dest: &req.Group},
	}
	for _, pair := range pairs {
		if pair.raw == nil {
			continue
		}
		if len(pair.raw) != 2 {
			return nil, errortypes.NewValidationError(
				pair.field,
				errortypes.ErrInvalid,
				"Give exactly two values",
			)
		}
		*pair.dest = &[2]string{pair.raw[0], pair.raw[1]}
	}

	return req, nil
}
