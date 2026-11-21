package archiveengine

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/emoss08/trenova/internal/cloud/copybinary"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgerrcode"
)

const (
	keyBatch       = 1_000
	auditTable     = "audit_entries"
	auditTimestamp = "timestamp"
	reasonTaken    = "an email address or username in the archive is already used on this server"
)

var takenConstraints = map[string]bool{
	"idx_users_email_address": true,
	"idx_users_username":      true,
}

type column struct {
	index int
	typ   string
}

type tableReader struct {
	r       *importRun
	table   *archiveplan.PlannedTable
	probe   bool
	keyAt   []column
	keys    [][]string
	emails  []string
	names   []string
	slugs   []string
	mapped  []mappedColumn
	objects []column
	purpose []kernelports.SecretPurpose
	id      column
	unit    column
	email   column
	name    column
	slug    column
	created column
	audit   column
}

func (t *tableReader) at(name string) column {
	i := t.table.ColumnIndex(name)
	if i < 0 {
		return column{index: -1}
	}
	return column{index: i, typ: t.table.Types[i]}
}

func (r *importRun) preflight(ctx context.Context) error {
	if err := r.readParts(ctx, true); err != nil {
		return err
	}
	if err := r.checkTenant(); err != nil {
		return err
	}
	tenant := r.found.tenant
	exists, err := r.e.rows.TenantExists(ctx, tenant)
	if err != nil {
		return err
	}
	if exists {
		return tenantarchive.Refuse(tenantarchive.RefusalExists,
			"the archive's organization or business unit is already on this server")
	}
	holding, err := r.e.rows.Holding(ctx, r.plan, tenant)
	if err != nil {
		return err
	}
	if len(holding) > 0 {
		return tenantarchive.Refuse(
			tenantarchive.RefusalExists,
			"this server already holds rows of the archive's organization in "+strings.Join(
				holding,
				", ",
			),
		)
	}
	keys := referenceKeys(r.plan)
	for global, values := range r.found.naturals {
		list := make([][]string, 0, len(values))
		for _, v := range values {
			list = append(list, v)
		}
		ids, rerr := r.e.rows.Resolve(ctx, global, keys[global], list)
		if rerr != nil {
			return rerr
		}
		r.mapping[global] = ids
	}
	return r.checkDocuments(ctx)
}

func (r *importRun) readParts(ctx context.Context, probe bool) error {
	r.sealed = r.found.objects
	for i, table := range r.plan.Tables {
		if err := r.readTable(ctx, table, &r.manifest.Tables[i], probe); err != nil {
			return err
		}
		if r.req.Beat != nil {
			r.req.Beat()
		}
	}
	return nil
}

func (r *importRun) readTable(
	ctx context.Context,
	table *archiveplan.PlannedTable,
	entry *tenantarchive.Table,
	probe bool,
) error {
	reader, closeReader, err := r.part(ctx, entry.Key,
		archivestream.Sum{Bytes: entry.SizeBytes, SHA256: entry.SHA256}, true)
	if err != nil {
		return err
	}
	defer closeReader()
	t := r.readerFor(table, probe)
	var rows int64
	if t == nil {
		rows, err = copybinary.Count(io.Discard, reader, len(table.Columns))
	} else {
		rows, err = copybinary.Rewrite(io.Discard, reader, len(table.Columns),
			func(row *copybinary.Row) error { return t.visit(ctx, row) })
		if err == nil {
			err = t.flush(ctx)
		}
	}
	if err != nil {
		return partRefusal(entry.Key, err)
	}
	if rows != entry.Rows {
		return tenantarchive.Refuse(tenantarchive.RefusalChecksum, fmt.Sprintf(
			"the archive's %s part holds %d rows and its manifest %d",
			entry.Name,
			rows,
			entry.Rows,
		))
	}
	return nil
}

func (r *importRun) readerFor(table *archiveplan.PlannedTable, probe bool) *tableReader {
	t := &tableReader{r: r, table: table, probe: probe}
	none := column{index: -1}
	t.id, t.unit, t.email, t.name, t.slug, t.created, t.audit = none, none, none, none, none, none, none
	useful := false
	if table.Name == archiveplan.Organizations {
		t.id, t.unit = t.at(archiveplan.IDColumn), t.at(archiveplan.BusinessUnitColumn)
		useful = true
	}
	if !probe {
		if useful {
			return t
		}
		return nil
	}
	if !table.ScopedKey() {
		for _, c := range table.PrimaryKey {
			t.keyAt = append(t.keyAt, t.at(c))
		}
		useful = true
	}
	switch table.Name {
	case archiveplan.Users:
		t.id, t.email, t.name, t.created = t.at(archiveplan.IDColumn), t.at("email_address"),
			t.at("username"), t.at("created_at")
		useful = true
	case archiveplan.Organizations:
		t.slug = t.at("login_slug")
	case auditTable:
		t.audit = t.at(auditTimestamp)
		useful = true
	}
	if t.mapped = mappedColumns(table); len(t.mapped) > 0 {
		useful = true
	}
	for _, oc := range r.e.registry.objectColumns(dbscope.Tenant{}) {
		if oc.table == table.Name {
			if c := t.at(oc.column); c.index >= 0 {
				t.objects = append(t.objects, c)
				t.purpose = append(t.purpose, oc.purpose)
				useful = true
			}
		}
	}
	if !useful {
		return nil
	}
	return t
}

func (t *tableReader) text(row *copybinary.Row, c column) (string, bool) {
	if c.index < 0 {
		return "", false
	}
	return textOf(row, c.index, c.typ)
}

func (t *tableReader) visit(ctx context.Context, row *copybinary.Row) error {
	f := t.r.found
	if t.table.Name == archiveplan.Organizations {
		id, _ := t.text(row, t.id)
		unit, _ := t.text(row, t.unit)
		f.orgs++
		f.tenant = dbscope.Tenant{OrganizationID: pulid.ID(id), BusinessUnitID: pulid.ID(unit)}
		if slug, ok := t.text(row, t.slug); ok && slug != "" {
			t.slugs = append(t.slugs, slug)
		}
	}
	if !t.probe {
		return nil
	}
	if len(t.keyAt) > 0 {
		key := make([]string, 0, len(t.keyAt))
		for _, c := range t.keyAt {
			value, err := keyText(row, c.index, c.typ)
			if err != nil {
				return fmt.Errorf("%s: %w", t.table.Name, err)
			}
			key = append(key, value)
		}
		t.keys = append(t.keys, key)
	}
	t.users(row)
	if t.audit.index >= 0 {
		if ts, ok := intOf(row, t.audit.index, baseType(t.audit.typ)); ok &&
			(f.firstLog == 0 || ts < f.firstLog) {
			f.firstLog = ts
		}
	}
	if err := t.naturals(row); err != nil {
		return err
	}
	for i, c := range t.objects {
		if key, ok := t.text(row, c); ok && key != "" {
			f.objects[key] = t.purpose[i]
		}
	}
	if len(t.keys) >= keyBatch || len(t.emails)+len(t.names) >= keyBatch ||
		len(t.slugs) >= keyBatch {
		return t.flush(ctx)
	}
	return nil
}

func (t *tableReader) users(row *copybinary.Row) {
	if t.email.index < 0 {
		return
	}
	f := t.r.found
	id, _ := t.text(row, t.id)
	if email, ok := t.text(row, t.email); ok {
		lower := strings.ToLower(email)
		t.emails = append(t.emails, lower)
		if owner := strings.ToLower(strings.TrimSpace(t.r.req.OwnerEmail)); owner != "" &&
			owner == lower {
			f.owner = pulid.ID(id)
		}
	}
	if name, ok := t.text(row, t.name); ok {
		t.names = append(t.names, strings.ToLower(name))
	}
	if t.created.index < 0 {
		return
	}
	if created, ok := intOf(row, t.created.index, baseType(t.created.typ)); ok &&
		(f.firstUser.IsNil() || created < f.earliest) {
		f.firstUser, f.earliest = pulid.ID(id), created
	}
}

func (t *tableReader) naturals(row *copybinary.Row) error {
	keys := referenceKeys(t.r.plan)
	for _, m := range t.mapped {
		value, ok := textOf(row, m.index, t.table.Types[m.index])
		if !ok {
			continue
		}
		natural, decoded := decodeNatural(value, len(keys[m.global]))
		if !decoded {
			return tenantarchive.Refuse(tenantarchive.RefusalManifest, fmt.Sprintf(
				"%s holds a malformed reference to %s", t.table.Name, m.global))
		}
		byKey := t.r.found.naturals[m.global]
		if byKey == nil {
			byKey = map[string][]string{}
			t.r.found.naturals[m.global] = byKey
		}
		byKey[joinKey(natural)] = natural
	}
	return nil
}

func (t *tableReader) flush(ctx context.Context) error {
	rows := t.r.e.rows
	if len(t.keys) > 0 {
		found, err := rows.ExistingKeys(ctx, t.table, t.keys)
		if err != nil {
			return err
		}
		if found > 0 {
			return tenantarchive.Refuse(tenantarchive.RefusalExists, fmt.Sprintf(
				"%d records of %s in the archive already exist on this server",
				found,
				t.table.Name,
			))
		}
		t.keys = t.keys[:0]
	}
	if len(t.emails) > 0 || len(t.names) > 0 {
		taken, err := rows.TakenIdentities(ctx, t.emails, t.names)
		if err != nil {
			return err
		}
		if taken {
			return tenantarchive.Refuse(tenantarchive.RefusalTaken, reasonTaken)
		}
		t.emails, t.names = t.emails[:0], t.names[:0]
	}
	if len(t.slugs) > 0 && t.probe {
		taken, err := rows.TakenSlugs(ctx, t.slugs)
		if err != nil {
			return err
		}
		if taken {
			return tenantarchive.Refuse(tenantarchive.RefusalTaken,
				"the organization's sign-in address is already used on this server")
		}
		t.slugs = t.slugs[:0]
	}
	return nil
}

func constraintRefusal(err error) (*tenantarchive.RefusalError, bool) {
	switch dberror.ExtractCode(err) {
	case pgerrcode.UniqueViolation:
		name := dberror.ExtractConstraintName(err)
		if takenConstraints[name] {
			return tenantarchive.Refuse(tenantarchive.RefusalTaken, reasonTaken), true
		}
		return tenantarchive.Refuse(tenantarchive.RefusalExists,
			"a record in the archive already exists on this server ("+name+")"), true
	case pgerrcode.ForeignKeyViolation:
		return tenantarchive.Refuse(
			tenantarchive.RefusalReference,
			"a record in the archive refers to one that is not there ("+dberror.ExtractConstraintName(
				err,
			)+")",
		), true
	default:
		return nil, false
	}
}
