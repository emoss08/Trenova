package archiveengine

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/copybinary"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	envelopePrefix = "trenova-envelope:v1:"
	plainMarker    = "trenova-archive-unsealed:v1:"
)

var ErrUnsealed = errors.New("a sealed value of the tenant does not open with this server's key")

type sealedColumn struct {
	column *kernelports.EnvelopeColumn
	name   string
	extras []string
}

type registry struct {
	columns map[string][]sealedColumn
	objects []kernelports.EnvelopeObject
}

func newRegistry(sources *kernelports.EnvelopeSources) *registry {
	r := &registry{columns: make(map[string][]sealedColumn), objects: sources.Objects}
	for i := range sources.Columns {
		column := &sources.Columns[i]
		table, index, ok := keel.ColumnOf(column.Value)
		if !ok {
			continue
		}
		extras := make([]string, 0, len(column.Extra))
		for _, extra := range column.Extra {
			extras = append(extras, kernelports.EnvelopeColumnName(extra))
		}
		r.columns[table.Name] = append(r.columns[table.Name], sealedColumn{
			column: column,
			name:   table.Columns[index].Name,
			extras: extras,
		})
	}
	return r
}

func (r *registry) tables() map[string][]string {
	out := make(map[string][]string, len(r.columns))
	for table, columns := range r.columns {
		for _, c := range columns {
			out[table] = append(out[table], c.name)
		}
		slices.Sort(out[table])
	}
	return out
}

type objectColumn struct {
	table   string
	column  string
	purpose kernelports.SecretPurpose
}

func (r *registry) objectColumns(tenant dbscope.Tenant) []objectColumn {
	row := &kernelports.EnvelopeRow{Tenant: tenantInfo(tenant), Extra: map[string]string{}}
	out := make([]objectColumn, 0, len(r.objects))
	for i := range r.objects {
		table, index, ok := keel.ColumnOf(r.objects[i].Column.Value)
		if !ok {
			continue
		}
		scopes := r.objects[i].Column.Scopes(row, "")
		if len(scopes) == 0 {
			continue
		}
		out = append(out, objectColumn{
			table:   table.Name,
			column:  table.Columns[index].Name,
			purpose: scopes[0].Purpose,
		})
	}
	return out
}

func (r *registry) objectScopes(tenant dbscope.Tenant, key string) []kernelports.SecretScope {
	row := &kernelports.EnvelopeRow{Tenant: tenantInfo(tenant), Extra: map[string]string{}}
	out := make([]kernelports.SecretScope, 0, len(r.objects))
	for i := range r.objects {
		for _, scope := range r.objects[i].Column.Scopes(row, key) {
			if !slices.ContainsFunc(out, func(s kernelports.SecretScope) bool {
				return s.Purpose == scope.Purpose && s.ResourceID == scope.ResourceID
			}) {
				out = append(out, scope)
			}
		}
	}
	return out
}

func tenantInfo(t dbscope.Tenant) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: t.OrganizationID, BuID: t.BusinessUnitID}
}

type rowSealer struct {
	cipher  kernelports.SecretCipher
	tenant  dbscope.Tenant
	types   []string
	idIndex int
	specs   []rowSpec
}

type rowSpec struct {
	sealed  sealedColumn
	index   int
	extras  map[string]int
	jsonMap bool
}

func (r *registry) sealerFor(
	table *archiveplan.PlannedTable,
	cipher kernelports.SecretCipher,
	tenant dbscope.Tenant,
) (*rowSealer, error) {
	columns := r.columns[table.Name]
	if len(columns) == 0 {
		return nil, nil //nolint:nilnil // a table with no sealed column needs no sealer
	}
	s := &rowSealer{
		cipher:  cipher,
		tenant:  tenant,
		types:   table.Types,
		idIndex: table.ColumnIndex(archiveplan.IDColumn),
	}
	for _, c := range columns {
		index := table.ColumnIndex(c.name)
		if index < 0 {
			continue
		}
		spec := rowSpec{
			sealed:  c,
			index:   index,
			extras:  map[string]int{},
			jsonMap: c.column.JSONMap,
		}
		for _, extra := range c.extras {
			at := table.ColumnIndex(extra)
			if at < 0 {
				return nil, fmt.Errorf(
					"%s.%s needs %s to open, which is not archived",
					table.Name,
					c.name,
					extra,
				)
			}
			spec.extras[extra] = at
		}
		s.specs = append(s.specs, spec)
	}
	if len(s.specs) == 0 {
		return nil, nil //nolint:nilnil // a table with no sealed column needs no sealer
	}
	return s, nil
}

func (s *rowSealer) envelopeRow(
	row *copybinary.Row,
	spec *rowSpec,
	sealed string,
) *kernelports.EnvelopeRow {
	out := &kernelports.EnvelopeRow{
		Tenant: tenantInfo(s.tenant),
		Sealed: sealed,
		Extra:  make(map[string]string, len(spec.extras)),
	}
	if s.idIndex >= 0 {
		out.ID, _ = textOf(row, s.idIndex, s.types[s.idIndex])
	}
	for name, at := range spec.extras {
		out.Extra[name], _ = textOf(row, at, s.types[at])
	}
	return out
}

func (s *rowSealer) unseal(ctx context.Context, row *copybinary.Row) error {
	return s.each(
		row,
		func(spec *rowSpec, er *kernelports.EnvelopeRow, field, value string) (string, error) {
			if !strings.HasPrefix(value, envelopePrefix) {
				return value, nil
			}
			for _, scope := range spec.sealed.column.Scopes(er, field) {
				plain, err := s.cipher.Open(ctx, value, &scope)
				if err == nil {
					return plainMarker + base64.StdEncoding.EncodeToString([]byte(plain)), nil
				}
			}
			return "", fmt.Errorf("%w: %s.%s", ErrUnsealed, spec.sealed.column.Name, field)
		},
	)
}

func (s *rowSealer) reseal(ctx context.Context, row *copybinary.Row) error {
	return s.each(
		row,
		func(spec *rowSpec, er *kernelports.EnvelopeRow, field, value string) (string, error) {
			encoded, ok := strings.CutPrefix(value, plainMarker)
			if !ok {
				return value, nil
			}
			plain, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return "", fmt.Errorf(
					"a sealed value of %s in the archive is malformed",
					spec.sealed.column.Name,
				)
			}
			scopes := spec.sealed.column.Scopes(er, field)
			if len(scopes) == 0 {
				return "", fmt.Errorf("%s names no binding to seal with", spec.sealed.column.Name)
			}
			return s.cipher.Seal(ctx, string(plain), &scopes[0])
		},
	)
}

type fieldFunc func(spec *rowSpec, row *kernelports.EnvelopeRow, field, value string) (string, error)

func (s *rowSealer) each(row *copybinary.Row, fn fieldFunc) error {
	for i := range s.specs {
		spec := &s.specs[i]
		typ := s.types[spec.index]
		value, present := textOf(row, spec.index, typ)
		if !present || value == "" {
			continue
		}
		er := s.envelopeRow(row, spec, value)
		if !spec.jsonMap {
			replaced, err := fn(spec, er, "", value)
			if err != nil {
				return err
			}
			setText(row, spec.index, typ, replaced)
			continue
		}
		fields := make(map[string]any)
		if err := jsonAPI.UnmarshalFromString(value, &fields); err != nil {
			return fmt.Errorf("decode the sealed fields of %s: %w", spec.sealed.column.Name, err)
		}
		for field, raw := range fields {
			text, ok := raw.(string)
			if !ok {
				continue
			}
			replaced, err := fn(spec, er, field, text)
			if err != nil {
				return err
			}
			fields[field] = replaced
		}
		encoded, err := jsonAPI.MarshalToString(fields)
		if err != nil {
			return fmt.Errorf("encode the sealed fields of %s: %w", spec.sealed.column.Name, err)
		}
		setText(row, spec.index, typ, encoded)
	}
	return nil
}
