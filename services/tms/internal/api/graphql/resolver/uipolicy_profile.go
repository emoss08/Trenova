package resolver

import (
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/zap"
)

// resolvedProfileToJSON renders the resolved mode profile for the wire.
//
// It lives here rather than in shipment.resolvers.go because gqlgen rewrites
// that file on every generate and drops anything that is not a resolver method.
//
// The struct's own `json` tags are the contract the client validates against
// with resolvedModeProfileSchema, so this marshals rather than restating the
// shape. TestResolvedPolicyJSON_MatchesTheClientContract pins the key names.
//
// A marshalling failure is logged rather than swallowed: dropping the profile
// silently is precisely how this field came to be missing from the response for
// so long, and every capability-driven behaviour in the form degrades to "no
// profile resolved" without saying anything.
func (r *queryResolver) resolvedProfileToJSON(
	policy *modeprofile.ResolvedPolicy,
) map[string]any {
	if policy == nil {
		return nil
	}

	rendered, err := jsonutils.ToJSON(policy)
	if err != nil {
		r.l.Error("failed to render resolved mode profile for the ui policy",
			zap.String("profileCode", policy.ProfileCode),
			zap.Error(err),
		)
		return nil
	}

	return rendered
}
