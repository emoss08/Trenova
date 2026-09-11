package graphql

import (
	"context"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	featureAccessExtensionName   = "PlatformFeatureAccess"
	FeatureAccessErrorCode       = "FEATURE_NOT_ENTITLED"
	rejectionReasonFeatureAccess = "feature_access"
	rejectionReasonObserved      = "feature_access_observed"
	introspectionFieldPrefix     = "__"
	graphQLRoutePattern          = "/graphql"
	selectOptionsFieldName       = "selectOptions"
	selectOptionsInputArgument   = "input"
	selectOptionsResourceField   = "resource"
	deniedFeatureReason          = "your plan does not include this feature"
)

type FeatureAccessParams struct {
	fx.In

	Config     *config.Config
	Registry   *platformcatalog.Registry
	Authorizer services.AccessAuthorizer `optional:"true"`
	Logger     *zap.Logger
	Metrics    *metrics.Registry
}

type FeatureAccessExtension struct {
	cfg        *config.Config
	registry   *platformcatalog.Registry
	authorizer services.AccessAuthorizer
	metrics    *metrics.GraphQL
	l          *zap.Logger
	now        func() time.Time
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
} = (*FeatureAccessExtension)(nil)

func NewFeatureAccessExtension(p FeatureAccessParams) *FeatureAccessExtension {
	errcode.RegisterErrorType(FeatureAccessErrorCode, errcode.KindProtocol)

	return &FeatureAccessExtension{
		cfg:        p.Config,
		registry:   p.Registry,
		authorizer: p.Authorizer,
		metrics:    p.Metrics.GraphQL,
		l:          p.Logger.Named("api.graphql.featureaccess"),
		now:        time.Now,
	}
}

func (*FeatureAccessExtension) ExtensionName() string {
	return featureAccessExtensionName
}

func (e *FeatureAccessExtension) Validate(schema graphql.ExecutableSchema) error {
	unclassified := e.registry.UnclassifiedGraphQLSources(schemaSources(schema.Schema()))
	if len(unclassified) == 0 {
		return nil
	}

	names := make([]string, 0, len(unclassified))
	for _, source := range unclassified {
		names = append(names, string(source))
	}
	e.l.Error(
		"GraphQL schema sources are not mapped to a platform feature; "+
			"operations selecting them are denied while feature access is enforcing",
		zap.Strings("sources", names),
		zap.String("mode", string(e.mode())),
	)

	return nil
}

func (e *FeatureAccessExtension) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	if !e.cfg.Platform.ControlPlane.Enabled || e.mode() == config.GraphQLAccessModeDisabled {
		return nil
	}
	if opCtx == nil || opCtx.Operation == nil {
		return nil
	}

	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil
	}

	operation := graphQLOperationName(opCtx.Operation.Operation)
	if operation == "" {
		return nil
	}

	policies := e.rootFieldPolicies(opCtx, operation)
	if len(policies) == 0 {
		return nil
	}

	return e.authorizePolicies(ctx, opCtx, authCtx, policies)
}

type rootFieldPolicy struct {
	field  string
	source string
	policy platformcatalog.RoutePolicy
}

func (e *FeatureAccessExtension) rootFieldPolicies(
	opCtx *graphql.OperationContext,
	operation string,
) []rootFieldPolicy {
	fields := collectRootFields(opCtx)
	policies := make([]rootFieldPolicy, 0, len(fields))
	for _, field := range fields {
		source := rootFieldSource(field)
		policies = append(policies, rootFieldPolicy{
			field:  field.Name,
			source: source,
			policy: e.policyForRootField(opCtx, operation, field, source),
		})
	}

	return policies
}

func (e *FeatureAccessExtension) policyForRootField(
	opCtx *graphql.OperationContext,
	operation string,
	field *ast.Field,
	source string,
) platformcatalog.RoutePolicy {
	if field.Name == selectOptionsFieldName {
		resource, ok := selectOptionResource(opCtx, field)
		if !ok {
			return platformcatalog.RoutePolicy{
				AccessClass: platformcatalog.RouteAccessClassUnclassified,
			}
		}

		return platformcatalog.PolicyForSelectOptionResource(resource)
	}

	return e.registry.PolicyForGraphQLRootField(
		operation,
		field.Name,
		platformcatalog.GraphQLSource(source),
	)
}

func selectOptionResource(
	opCtx *graphql.OperationContext,
	field *ast.Field,
) (string, bool) {
	argument := field.Arguments.ForName(selectOptionsInputArgument)
	if argument == nil || argument.Value == nil {
		return "", false
	}

	switch argument.Value.Kind {
	case ast.ObjectValue:
		child := argument.Value.Children.ForName(selectOptionsResourceField)
		if child == nil {
			return "", false
		}
		if child.Kind == ast.Variable {
			return variableString(opCtx, child.Raw)
		}

		return child.Raw, child.Raw != ""
	case ast.Variable:
		return objectVariableString(opCtx, argument.Value.Raw, selectOptionsResourceField)
	default:
		return "", false
	}
}

func variableString(opCtx *graphql.OperationContext, name string) (string, bool) {
	raw, ok := opCtx.Variables[name]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)

	return value, ok && value != ""
}

func objectVariableString(
	opCtx *graphql.OperationContext,
	name string,
	key string,
) (string, bool) {
	raw, ok := opCtx.Variables[name]
	if !ok {
		return "", false
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return "", false
	}
	value, ok := object[key].(string)

	return value, ok && value != ""
}

func (e *FeatureAccessExtension) authorizePolicies(
	ctx context.Context,
	opCtx *graphql.OperationContext,
	authCtx *authctx.AuthContext,
	policies []rootFieldPolicy,
) *gqlerror.Error {
	checked := make(map[platformcatalog.FeatureKey]struct{}, len(policies))
	for _, entry := range policies {
		switch entry.policy.AccessClass {
		case platformcatalog.RouteAccessClassAccountShell:
			continue
		case platformcatalog.RouteAccessClassProduct:
		case platformcatalog.RouteAccessClassUnclassified:
			fallthrough
		default:
			if err := e.rejectUnclassified(ctx, opCtx, entry); err != nil {
				return err
			}
			continue
		}

		if _, seen := checked[entry.policy.FeatureKey]; seen {
			continue
		}
		checked[entry.policy.FeatureKey] = struct{}{}

		if err := e.authorizeFeature(ctx, opCtx, authCtx, entry); err != nil {
			return err
		}
	}

	return nil
}

func (e *FeatureAccessExtension) rejectUnclassified(
	ctx context.Context,
	opCtx *graphql.OperationContext,
	entry rootFieldPolicy,
) *gqlerror.Error {
	e.l.Warn(
		"GraphQL root field is not mapped to a platform feature",
		zap.String("operation", opCtx.OperationName),
		zap.String("field", entry.field),
		zap.String("source", entry.source),
		zap.String("request_id", gqlctx.RequestID(ctx)),
	)
	return e.refuse(ctx, entry.field, "field is not mapped to a licensed product feature")
}

func (e *FeatureAccessExtension) authorizeFeature(
	ctx context.Context,
	opCtx *graphql.OperationContext,
	authCtx *authctx.AuthContext,
	entry rootFieldPolicy,
) *gqlerror.Error {
	if e.authorizer == nil {
		e.l.Error(
			"control-plane authorization is not configured for GraphQL feature access",
			zap.String("featureKey", string(entry.policy.FeatureKey)),
		)
		return e.refuse(ctx, entry.field, "control-plane authorization is not configured")
	}

	checkedAt := e.now().Unix()
	reason := ""
	checkFailed := false
	authorizing := e.registry.AuthorizingFeatures(
		entry.policy.FeatureKey,
		e.cfg.Platform.ControlPlane.HonorLegacyGrants(),
	)
	for _, featureKey := range authorizing {
		result, err := e.authorizer.AuthorizeAccess(ctx, &services.AccessAuthorizeRequest{
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
			PrincipalID:    authCtx.PrincipalID,
			UserID:         authCtx.UserID,
			APIKeyID:       authCtx.APIKeyID,
			HTTPMethod:     http.MethodPost,
			HTTPPath:       graphQLRoutePattern,
			RoutePattern:   graphQLRoutePattern,
			FeatureKey:     featureKey,
			CheckedAt:      checkedAt,
		})
		if err != nil {
			checkFailed = true
			e.l.Error(
				"GraphQL feature access check failed",
				zap.String("featureKey", string(featureKey)),
				zap.String("request_id", gqlctx.RequestID(ctx)),
				zap.Error(err),
			)
			continue
		}
		if result.Allowed {
			return nil
		}
		if reason == "" {
			reason = result.Reason
		}
	}

	e.l.Warn(
		"GraphQL feature access denied",
		zap.String("operation", opCtx.OperationName),
		zap.String("field", entry.field),
		zap.String("featureKey", string(entry.policy.FeatureKey)),
		zap.String("reason", reason),
		zap.String("organizationID", authCtx.OrganizationID.String()),
		zap.String("request_id", gqlctx.RequestID(ctx)),
	)
	if reason == "" {
		if checkFailed {
			return e.refuse(ctx, entry.field, "feature entitlement could not be verified")
		}
		reason = deniedFeatureReason
	}

	return e.refuse(ctx, entry.field, reason)
}

func (e *FeatureAccessExtension) refuse(
	ctx context.Context,
	field string,
	reason string,
) *gqlerror.Error {
	if !e.enforcing() {
		e.metrics.RecordRejection(rejectionReasonObserved)
		return nil
	}

	e.metrics.RecordRejection(rejectionReasonFeatureAccess)
	if status, found := gqlctx.ResponseStatusFrom(ctx); found {
		status.Override(http.StatusForbidden, 0)
	}

	err := gqlerror.Errorf("%s: %s", field, reason)
	errcode.Set(err, FeatureAccessErrorCode)

	return err
}

func (e *FeatureAccessExtension) mode() config.GraphQLAccessMode {
	return e.cfg.Platform.ControlPlane.GetGraphQLAccessMode()
}

func (e *FeatureAccessExtension) enforcing() bool {
	return e.cfg.Platform.ControlPlane.Enabled &&
		e.mode() == config.GraphQLAccessModeEnforce
}

func collectRootFields(opCtx *graphql.OperationContext) []*ast.Field {
	fields := make([]*ast.Field, 0, len(opCtx.Operation.SelectionSet))
	seen := make(map[*ast.Field]struct{}, len(opCtx.Operation.SelectionSet))
	visitedFragments := make(map[string]struct{})

	var walk func(selectionSet ast.SelectionSet)
	walk = func(selectionSet ast.SelectionSet) {
		for _, selection := range selectionSet {
			switch typed := selection.(type) {
			case *ast.Field:
				if strings.HasPrefix(typed.Name, introspectionFieldPrefix) {
					continue
				}
				if _, duplicate := seen[typed]; duplicate {
					continue
				}
				seen[typed] = struct{}{}
				fields = append(fields, typed)
			case *ast.InlineFragment:
				walk(typed.SelectionSet)
			case *ast.FragmentSpread:
				if typed.Definition == nil {
					continue
				}
				if _, visited := visitedFragments[typed.Name]; visited {
					continue
				}
				visitedFragments[typed.Name] = struct{}{}
				walk(typed.Definition.SelectionSet)
			}
		}
	}
	walk(opCtx.Operation.SelectionSet)

	return fields
}

func rootFieldSource(field *ast.Field) string {
	if field.Definition == nil || field.Definition.Position == nil ||
		field.Definition.Position.Src == nil {
		return ""
	}

	return path.Base(field.Definition.Position.Src.Name)
}

func schemaSources(schema *ast.Schema) []platformcatalog.GraphQLSource {
	if schema == nil {
		return nil
	}

	seen := make(map[string]struct{})
	for _, definition := range []*ast.Definition{schema.Query, schema.Mutation} {
		if definition == nil {
			continue
		}
		for _, field := range definition.Fields {
			if strings.HasPrefix(field.Name, introspectionFieldPrefix) {
				continue
			}
			if field.Position == nil || field.Position.Src == nil {
				continue
			}
			seen[path.Base(field.Position.Src.Name)] = struct{}{}
		}
	}

	sources := make([]platformcatalog.GraphQLSource, 0, len(seen))
	for name := range seen {
		sources = append(sources, platformcatalog.GraphQLSource(name))
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i] < sources[j] })

	return sources
}

func graphQLOperationName(operation ast.Operation) string {
	switch operation {
	case ast.Query:
		return platformcatalog.GraphQLOperationQuery
	case ast.Mutation:
		return platformcatalog.GraphQLOperationMutation
	default:
		return ""
	}
}
