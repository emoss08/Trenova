package supportaccessgraphql

import (
	"context"
	"net/http"
	"path"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessservice"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
)

const (
	extensionName         = "TrenovaSupportSession"
	ReadOnlyErrorCode     = "SUPPORT_SESSION_READ_ONLY"
	DeniedErrorCode       = "SUPPORT_SESSION_DENIED"
	introspectionPrefix   = "__"
	readOnlyRefusalDetail = "this Trenova support session is read-only; elevate to write to make changes"
	deniedRefusalDetail   = "Trenova support sessions cannot do this, even with write access"
)

func init() {
	errcode.RegisterErrorType(ReadOnlyErrorCode, errcode.KindUser)
	errcode.RegisterErrorType(DeniedErrorCode, errcode.KindUser)
}

type refusalRecorder interface {
	RecordRefusal(ctx context.Context, req *supportaccessservice.RefusalRequest)
}

type Params struct {
	fx.In

	Service *supportaccessservice.Service
}

type Extension struct {
	recorder refusalRecorder
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
} = (*Extension)(nil)

func New(p Params) *Extension {
	return &Extension{recorder: p.Service}
}

func (*Extension) ExtensionName() string {
	return extensionName
}

func (*Extension) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (e *Extension) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	active, ok := supportctx.ActiveFrom(ctx)
	if !ok || opCtx == nil || opCtx.Operation == nil ||
		opCtx.Operation.Operation != ast.Mutation {
		return nil
	}

	for _, field := range rootFields(opCtx.Operation.SelectionSet) {
		decision := supportctx.DecideMutation(supportctx.MutationRequest{
			FieldName:   field.Name,
			Source:      sourceOf(field),
			WriteActive: active.WriteActive(),
		})
		if decision == supportctx.DecisionAllow {
			continue
		}

		if e.recorder != nil {
			e.recorder.RecordRefusal(ctx, &supportaccessservice.RefusalRequest{
				Active:  active,
				Method:  http.MethodPost,
				Route:   "/graphql",
				Subject: "the " + field.Name + " mutation",
				Denied:  decision == supportctx.DecisionDenied,
			})
		}

		return refuse(ctx, field.Name, decision)
	}

	return nil
}

func refuse(ctx context.Context, field string, decision supportctx.Decision) *gqlerror.Error {
	if status, found := gqlctx.ResponseStatusFrom(ctx); found {
		status.Override(http.StatusForbidden, 0)
	}

	detail, code := readOnlyRefusalDetail, ReadOnlyErrorCode
	if decision == supportctx.DecisionDenied {
		detail, code = deniedRefusalDetail, DeniedErrorCode
	}

	err := gqlerror.Errorf("%s: %s", field, detail)
	errcode.Set(err, code)

	return err
}

func rootFields(selections ast.SelectionSet) []*ast.Field {
	fields := make([]*ast.Field, 0, len(selections))
	visited := make(map[string]struct{})

	var walk func(ast.SelectionSet)
	walk = func(set ast.SelectionSet) {
		for _, selection := range set {
			switch typed := selection.(type) {
			case *ast.Field:
				if !strings.HasPrefix(typed.Name, introspectionPrefix) {
					fields = append(fields, typed)
				}
			case *ast.InlineFragment:
				walk(typed.SelectionSet)
			case *ast.FragmentSpread:
				if typed.Definition == nil {
					continue
				}
				if _, seen := visited[typed.Name]; seen {
					continue
				}
				visited[typed.Name] = struct{}{}
				walk(typed.Definition.SelectionSet)
			}
		}
	}
	walk(selections)

	return fields
}

func sourceOf(field *ast.Field) string {
	if field.Definition == nil || field.Definition.Position == nil ||
		field.Definition.Position.Src == nil {
		return ""
	}

	return path.Base(field.Definition.Position.Src.Name)
}
