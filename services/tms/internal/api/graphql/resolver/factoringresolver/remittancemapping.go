package factoringresolver

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/factoringremit"
	"github.com/emoss08/trenova/pkg/pagination"
)

func remittanceFieldToModel(field factoringremit.Field) (gqlmodel.FactoringRemittanceField, bool) {
	raw := field.String()
	first, size := utf8.DecodeRuneInString(raw)
	if size == 0 {
		return "", false
	}
	value := gqlmodel.FactoringRemittanceField(string(unicode.ToUpper(first)) + raw[size:])
	return value, value.IsValid()
}

func remittanceFieldFromModel(field gqlmodel.FactoringRemittanceField) factoringremit.Field {
	raw := string(field)
	first, size := utf8.DecodeRuneInString(raw)
	if size == 0 {
		return ""
	}
	return factoringremit.Field(string(unicode.ToLower(first)) + raw[size:])
}

func remittanceMappingToModel(mapping map[string]int) []*gqlmodel.FactoringRemittanceColumn {
	out := make([]*gqlmodel.FactoringRemittanceColumn, 0, len(mapping))
	for _, field := range factoringremit.Fields() {
		column, ok := mapping[field.String()]
		if !ok {
			continue
		}
		value, valid := remittanceFieldToModel(field)
		if !valid {
			continue
		}
		out = append(out, &gqlmodel.FactoringRemittanceColumn{Field: value, Column: column})
	}
	return out
}

func remittanceMappingFromModel(
	columns []*gqlmodel.FactoringRemittanceColumnInput,
) (map[string]int, error) {
	if len(columns) == 0 {
		return nil, nil //nolint:nilnil // no mapping means the saved layout or the standard headers
	}
	mapping := make(map[string]int, len(columns))
	for _, column := range columns {
		if column == nil {
			continue
		}
		field := remittanceFieldFromModel(column.Field).String()
		if _, dup := mapping[field]; dup {
			return nil, errortypes.NewValidationError(
				"mapping",
				errortypes.ErrDuplicate,
				"Each field is mapped to one column",
			)
		}
		mapping[field] = column.Column
	}
	return mapping, nil
}

func remittanceConnectionToModel(
	result *pagination.CursorListResult[*factoring.Remittance],
) (*gqlmodel.FactoringRemittanceConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *factoring.Remittance, cursor string) *gqlmodel.FactoringRemittanceEdge {
			return &gqlmodel.FactoringRemittanceEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.FactoringRemittanceEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.FactoringRemittanceConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func remittanceLineConnectionToModel(
	result *pagination.CursorListResult[*factoring.RemittanceLine],
) (*gqlmodel.FactoringRemittanceLineConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *factoring.RemittanceLine, cursor string) *gqlmodel.FactoringRemittanceLineEdge {
			return &gqlmodel.FactoringRemittanceLineEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.FactoringRemittanceLineEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.FactoringRemittanceLineConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func nonEmptyEnum[T ~string](value T) *T {
	if strings.TrimSpace(string(value)) == "" {
		return nil
	}
	return &value
}

func validRemittanceKind(kind *factoring.RemittanceLineKind) factoring.RemittanceLineKind {
	if kind == nil || !slices.Contains(factoring.AllRemittanceLineKinds(), *kind) {
		return ""
	}
	return *kind
}

func remittanceVersionFromModel(
	in *gqlmodel.FactoringRemittanceVersionInput,
	authCtx *authctx.AuthContext,
) (*factoringports.RemittanceVersionRequest, error) {
	id, err := parseID(in.ID, "id")
	if err != nil {
		return nil, err
	}
	return &factoringports.RemittanceVersionRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		ID:         id,
		Version:    int64(in.Version),
	}, nil
}

// linesIn are a remittance's lines in the file's order, or those in the
// match statuses named.
func linesIn(
	lines []*factoring.RemittanceLine,
	statuses []factoring.RemittanceMatchStatus,
) []*factoring.RemittanceLine {
	out := make([]*factoring.RemittanceLine, 0, len(lines))
	for _, line := range lines {
		if line != nil && (len(statuses) == 0 || slices.Contains(statuses, line.MatchStatus)) {
			out = append(out, line)
		}
	}

	return out
}
