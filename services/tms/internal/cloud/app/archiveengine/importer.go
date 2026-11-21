package archiveengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/cloud/archivecrypt"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/emoss08/trenova/internal/cloud/copybinary"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type ImportRequest struct {
	InstanceID   string
	ExpectSHA256 string
	Access       *tenantarchive.Access
	Tenant       *dbscope.Tenant
	Restoring    bool
	OwnerEmail   string
	Root         string
	Progress     Reporter
	Beat         func()
}

type ImportResult struct {
	Manifest         *tenantarchive.Manifest
	Tenant           dbscope.Tenant
	Owner            pulid.ID
	Rows             int64
	Documents        int
	ForeignRecovery  int64
	ManifestSHA256   string
	RecoveryReplaced bool
	Trails           *archiveports.TrailResult
}

type importRun struct {
	e        *Engine
	req      *ImportRequest
	bucket   archiveports.Bucket
	manifest *tenantarchive.Manifest
	digest   string
	key      *archivecrypt.DataKey
	plan     *archiveplan.Plan
	found    *found
	mapping  map[string]map[string]string
	sealed   map[string]kernelports.SecretPurpose
	progress tenantcommand.Progress
	writtenM sync.Mutex
	written  []string
}

type found struct {
	tenant    dbscope.Tenant
	orgs      int
	owner     pulid.ID
	earliest  int64
	firstUser pulid.ID
	firstLog  int64
	naturals  map[string]map[string][]string
	objects   map[string]kernelports.SecretPurpose
}

func (e *Engine) open(ctx context.Context, req *ImportRequest) (*importRun, error) {
	root := req.Root
	if root == "" {
		root = tenantarchive.TenantPrefix(req.InstanceID)
	}
	if err := req.Access.CheckReadUnder(root); err != nil {
		return nil, tenantarchive.Refuse(tenantarchive.RefusalLocation, err.Error())
	}
	bucket, err := e.buckets.Open(&req.Access.Location, &req.Access.Credentials)
	if err != nil {
		return nil, err
	}
	run := &importRun{
		e:       e,
		req:     req,
		bucket:  bucket,
		mapping: map[string]map[string]string{},
		found: &found{
			naturals: map[string]map[string][]string{},
			objects:  map[string]kernelports.SecretPurpose{},
		},
	}
	run.progress.Phase = tenantcommand.PhaseChecking
	req.Progress.report(&run.progress)
	if err = run.readManifest(ctx); err != nil {
		return nil, err
	}
	if run.plan, err = e.rows.PlanFor(ctx, run.manifest.Purpose, e.registry.tables()); err != nil {
		return nil, err
	}
	if err = run.plan.Matches(run.manifest); err != nil {
		return nil, err
	}
	return run, nil
}

func (e *Engine) Verify(ctx context.Context, req *ImportRequest) (*tenantarchive.Manifest, error) {
	run, err := e.open(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = run.readParts(ctx, false); err != nil {
		return nil, err
	}
	if err = run.checkTenant(); err != nil {
		return nil, err
	}
	if err = run.checkDocuments(ctx); err != nil {
		return nil, err
	}
	return run.manifest, nil
}

func (e *Engine) Import(ctx context.Context, req *ImportRequest) (*ImportResult, error) {
	run, err := e.open(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = run.preflight(ctx); err != nil {
		return nil, err
	}
	if run.found.firstLog > 0 {
		if err = e.rows.EnsureAuditPartitions(ctx, run.found.firstLog); err != nil {
			return nil, err
		}
	}
	return run.load(ctx)
}

func (r *importRun) readManifest(ctx context.Context) error {
	body, err := r.bucket.Get(ctx, r.req.Access.Location.Key(tenantarchive.ManifestName))
	if errors.Is(err, archiveports.ErrMissing) {
		return tenantarchive.Refuse(tenantarchive.RefusalManifest,
			"the location holds no manifest.json, so the archive is not complete")
	}
	if err != nil {
		return err
	}
	defer body.Close()
	raw, err := io.ReadAll(io.LimitReader(body, tenantarchive.MaxManifestBytes+1))
	if err != nil {
		return fmt.Errorf("read the archive manifest: %w", err)
	}
	if len(raw) > tenantarchive.MaxManifestBytes {
		return tenantarchive.Refuse(
			tenantarchive.RefusalManifest,
			"the manifest is larger than 64 MiB",
		)
	}
	r.digest = tenantarchive.ManifestSHA256(raw)
	if r.req.ExpectSHA256 != "" && r.digest != r.req.ExpectSHA256 {
		return tenantarchive.Refuse(tenantarchive.RefusalChecksum,
			"the manifest's SHA-256 is not the one the command expects")
	}
	m, err := tenantarchive.DecodeManifest(raw)
	if err != nil {
		return err
	}
	if err = m.Check(); err != nil {
		return err
	}
	if err = r.admit(m); err != nil {
		return err
	}
	r.manifest = m
	return nil
}

func (r *importRun) admit(m *tenantarchive.Manifest) error {
	switch {
	case !m.Purpose.Importable():
		return tenantarchive.Refuse(tenantarchive.RefusalPurpose,
			"an export archive is the customer's copy without its secrets and cannot be imported")
	case m.InstanceID != r.req.InstanceID:
		return tenantarchive.Refuse(tenantarchive.RefusalInstance,
			"the archive is of another tenant than the one it is being loaded for")
	case m.SchemaVersion != r.e.rows.SchemaVersion():
		return tenantarchive.Refuse(tenantarchive.RefusalSchema, fmt.Sprintf(
			"the archive was taken at schema version %s and this server is at %s",
			m.SchemaVersion, r.e.rows.SchemaVersion()))
	}
	key, err := r.req.Access.KeyByID(m.Encryption.KeyID)
	if err != nil {
		return tenantarchive.Refuse(
			tenantarchive.RefusalKeyAbsent,
			"the archive was sealed with archive key "+m.Encryption.KeyID+", which the command does not carry",
		)
	}
	dataKey, err := archivecrypt.Unwrap(m.Encryption.WrappedDataKey, key.Material, m.InstanceID)
	if err != nil {
		return tenantarchive.Refuse(tenantarchive.RefusalKey, err.Error())
	}
	r.key = dataKey
	return nil
}

func (r *importRun) part(
	ctx context.Context,
	name string,
	sum archivestream.Sum,
	compress bool,
) (*archivestream.Reader, func(), error) {
	body, err := r.bucket.Get(ctx, r.req.Access.Location.Key(name))
	if errors.Is(err, archiveports.ErrMissing) {
		return nil, nil, tenantarchive.Refuse(tenantarchive.RefusalChecksum,
			"the archive is missing its part "+name)
	}
	if err != nil {
		return nil, nil, err
	}
	reader, err := archivestream.NewReader(body, sum, streamOptions(name, r.key, compress))
	if err != nil {
		_ = body.Close()
		return nil, nil, partRefusal(name, err)
	}
	return reader, func() {
		reader.Close()
		_ = body.Close()
	}, nil
}

func partRefusal(name string, err error) error {
	if err == nil {
		return nil
	}
	if refused(err) {
		return err
	}
	if errors.Is(err, archivestream.ErrStoredBytesChanged) ||
		errors.Is(err, archivestream.ErrExpandsTooFar) ||
		errors.Is(err, archivecrypt.ErrTampered) ||
		errors.Is(err, copybinary.ErrShape) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		strings.Contains(err.Error(), "zstd") ||
		strings.Contains(err.Error(), "magic number") {
		return tenantarchive.Refuse(
			tenantarchive.RefusalChecksum,
			"the archive's part "+name+" does not match its manifest: it was changed, cut short or sealed with another key",
		)
	}
	return err
}

func (r *importRun) checkTenant() error {
	f := r.found
	switch {
	case f.orgs != 1 || !f.tenant.Valid():
		return tenantarchive.Refuse(tenantarchive.RefusalManifest,
			"the archive does not hold exactly one organization")
	case r.req.Tenant != nil && (r.req.Tenant.OrganizationID != f.tenant.OrganizationID ||
		r.req.Tenant.BusinessUnitID != f.tenant.BusinessUnitID):
		return tenantarchive.Refuse(tenantarchive.RefusalInstance,
			"the archive holds another organization than the tenant being restored")
	default:
		return r.checkDocumentOwners()
	}
}

func (r *importRun) checkDocumentOwners() error {
	for i := range r.manifest.Documents {
		object, _ := tenantarchive.ObjectKey(r.manifest.Documents[i].Key)
		if _, ok := objectkey.Rebase(object, r.found.tenant.OrganizationID); !ok {
			return tenantarchive.Refuse(tenantarchive.RefusalManifest, fmt.Sprintf(
				"document %d is not stored under the tenant's own keys", i+1))
		}
	}
	return nil
}

func (r *importRun) wrote(key string) {
	r.writtenM.Lock()
	defer r.writtenM.Unlock()
	r.written = append(r.written, key)
}

func (r *importRun) checkDocuments(ctx context.Context) error {
	workers := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError().
		WithMaxGoroutines(DocumentConcurrency)
	for i := range r.manifest.Documents {
		doc := &r.manifest.Documents[i]
		workers.Go(func(ctx context.Context) error {
			body, err := r.bucket.Get(ctx, r.req.Access.Location.Key(doc.Key))
			if errors.Is(err, archiveports.ErrMissing) {
				return tenantarchive.Refuse(tenantarchive.RefusalChecksum,
					"the archive is missing its part "+doc.Key)
			}
			if err != nil {
				return err
			}
			defer body.Close()
			return partRefusal(
				doc.Key,
				archivestream.Verify(
					body,
					archivestream.Sum{Bytes: doc.SizeBytes, SHA256: doc.SHA256},
				),
			)
		})
	}
	return workers.Wait()
}

func (r *importRun) load(ctx context.Context) (*ImportResult, error) {
	tenant := r.found.tenant
	if err := r.e.clearObjects(ctx, tenant); err != nil {
		return nil, err
	}
	if err := r.putDocuments(ctx); err != nil {
		r.discardObjects(ctx)
		return nil, err
	}
	result, err := r.loadRows(ctx)
	if err != nil {
		r.discardObjects(ctx)
		return nil, err
	}
	return result, nil
}

func (r *importRun) discardObjects(ctx context.Context) {
	ctx = context.WithoutCancel(ctx)
	r.writtenM.Lock()
	written := slices.Clone(r.written)
	r.writtenM.Unlock()
	failed := 0
	for _, key := range written {
		if err := r.e.storage.Delete(ctx, key); err != nil &&
			!errors.Is(err, kernelports.ErrObjectNotFound) {
			failed++
		}
	}
	if failed > 0 {
		r.e.l.Error(
			"cannot remove every document an import that did not complete wrote; the next attempt removes them first",
			zap.Int("left", failed),
		)
	}
}

func (r *importRun) putDocuments(ctx context.Context) error {
	r.progress.Phase = tenantcommand.PhaseDocuments
	r.progress.DocumentsTotal = len(r.manifest.Documents)
	r.req.Progress.report(&r.progress)
	loader := &documentLoad{e: r.e, run: r, tenant: r.found.tenant}
	var mu sync.Mutex
	workers := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError().
		WithMaxGoroutines(DocumentConcurrency)
	for i := range r.manifest.Documents {
		doc := &r.manifest.Documents[i]
		workers.Go(func(ctx context.Context) error {
			if err := loader.put(ctx, doc); err != nil {
				return partRefusal(doc.Key, err)
			}
			mu.Lock()
			r.progress.DocumentsDone++
			snapshot := r.progress
			mu.Unlock()
			r.req.Progress.report(&snapshot)
			if r.req.Beat != nil {
				r.req.Beat()
			}
			return nil
		})
	}
	return workers.Wait()
}

func (r *importRun) loadRows(ctx context.Context) (*ImportResult, error) {
	r.progress.Phase = tenantcommand.PhaseLoading
	r.progress.TablesTotal = len(r.manifest.Tables)
	r.req.Progress.report(&r.progress)
	load, err := r.e.rows.OpenLoad(ctx, &archiveports.LoadRequest{
		Tenant:   r.found.tenant,
		Manifest: r.manifest,
		Envelope: r.e.registry.tables(),
		Beat:     r.req.Beat,
	})
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			load.Rollback(ctx)
		}
	}()
	result := &ImportResult{
		Manifest:       r.manifest,
		Tenant:         r.found.tenant,
		Owner:          r.found.owner,
		Documents:      len(r.manifest.Documents),
		ManifestSHA256: r.digest,
	}
	if result.Owner.IsNil() {
		result.Owner = r.found.firstUser
	}
	for i, table := range load.Plan().Tables {
		entry := &r.manifest.Tables[i]
		rows, terr := r.loadTable(ctx, load, table, entry)
		if terr != nil {
			return nil, loadRefusal(terr)
		}
		if rows != entry.Rows {
			return nil, tenantarchive.Refuse(tenantarchive.RefusalChecksum, fmt.Sprintf(
				"the archive's %s part holds %d rows and its manifest %d",
				entry.Name,
				rows,
				entry.Rows,
			))
		}
		result.Rows += rows
		r.progress.TablesDone++
		r.req.Progress.report(&r.progress)
	}
	if err = load.Finish(ctx); err != nil {
		return nil, loadRefusal(err)
	}
	if !r.req.Restoring {
		if result.ForeignRecovery, err = load.MarkForeignRecoveryCodes(ctx); err != nil {
			return nil, err
		}
		result.RecoveryReplaced = result.ForeignRecovery > 0
	}
	r.progress.Phase = tenantcommand.PhaseVerifying
	r.req.Progress.report(&r.progress)
	if err = r.verify(ctx, load); err != nil {
		return nil, err
	}
	if result.Trails, err = load.Trails(ctx, &archiveports.TrailRequest{
		Restoring:      r.req.Restoring,
		ManifestSHA256: r.digest,
		Manifest:       r.manifest,
		At:             r.e.clock(),
	}); err != nil {
		return nil, err
	}
	if err = load.Commit(ctx); err != nil {
		return nil, err
	}
	committed = true
	return result, nil
}

func (r *importRun) loadTable(
	ctx context.Context,
	load archiveports.Load,
	table *archiveplan.PlannedTable,
	entry *tenantarchive.Table,
) (int64, error) {
	reader, closeReader, err := r.part(ctx, entry.Key,
		archivestream.Sum{Bytes: entry.SizeBytes, SHA256: entry.SHA256}, true)
	if err != nil {
		return 0, err
	}
	defer closeReader()
	fn, err := r.rowFixer(ctx, table)
	if err != nil {
		return 0, err
	}
	if fn == nil {
		rows, cerr := load.CopyIn(ctx, table, reader)
		return rows, partRefusal(entry.Key, cerr)
	}
	pr, pw := io.Pipe()
	done := transform(reader, pw, len(table.Columns), fn)
	rows, copyErr := load.CopyIn(ctx, table, pr)
	_ = pr.CloseWithError(copyErr)
	result := <-done
	if refused(result.err) {
		return 0, result.err
	}
	if err = errors.Join(copyErr, result.err); err != nil {
		return 0, partRefusal(entry.Key, err)
	}
	return rows, nil
}

func (r *importRun) rowFixer(
	ctx context.Context,
	table *archiveplan.PlannedTable,
) (func(*copybinary.Row) error, error) {
	if !table.Rewrites() {
		return nil, nil //nolint:nilnil // a table that needs no rewrite streams through untouched
	}
	sealer, err := r.e.registry.sealerFor(table, r.e.secrets, r.found.tenant)
	if err != nil {
		return nil, err
	}
	keys := referenceKeys(r.plan)
	maps := mappedColumns(table)
	nulled := make([]int, 0, len(table.Nulled))
	for _, column := range table.Nulled {
		nulled = append(nulled, table.ColumnIndex(column))
	}
	return func(row *copybinary.Row) error {
		for _, m := range maps {
			typ := table.Types[m.index]
			value, present := textOf(row, m.index, typ)
			if !present {
				continue
			}
			natural, ok := decodeNatural(value, len(keys[m.global]))
			id, mapped := r.mapping[m.global][joinKey(natural)]
			if !ok || !mapped {
				return tenantarchive.Refuse(tenantarchive.RefusalReference, fmt.Sprintf(
					"%s refers to a %s row this server does not have", table.Name, m.global))
			}
			setText(row, m.index, typ, id)
		}
		for _, at := range nulled {
			row.SetNull(at)
		}
		if sealer != nil {
			return sealer.reseal(ctx, row)
		}
		return nil
	}, nil
}

func referenceKeys(plan *archiveplan.Plan) map[string][]string {
	out := make(map[string][]string, len(plan.References))
	for _, ref := range plan.References {
		out[ref.Table] = ref.Key
	}
	return out
}

func loadRefusal(err error) error {
	if refused(err) {
		return err
	}
	if reason, ok := constraintRefusal(err); ok {
		return reason
	}
	return err
}

func (r *importRun) verify(ctx context.Context, load archiveports.Load) error {
	counts, err := load.Counts(ctx)
	if err != nil {
		return err
	}
	for _, c := range counts {
		entry, ok := r.manifest.Table(tenantarchive.QualifiedTable(c.Table))
		if !ok || entry.Rows != c.Rows {
			return fmt.Errorf(
				"after the load %s holds %d rows of the tenant, not the archive's",
				c.Table,
				c.Rows,
			)
		}
	}
	dangling, err := load.Dangling(ctx)
	if err != nil {
		return err
	}
	if len(dangling) > 0 {
		names := make([]string, 0, len(dangling))
		for _, d := range dangling {
			names = append(names, fmt.Sprintf("%s.%s (%d rows)", d.Table, d.Columns, d.Rows))
		}
		return tenantarchive.Refuse(
			tenantarchive.RefusalReference,
			"after the load the tenant's rows refer to records that are not there: "+strings.Join(
				names,
				", ",
			),
		)
	}
	return r.verifyObjects(ctx)
}

func (r *importRun) verifyObjects(ctx context.Context) error {
	listed, err := r.e.listObjects(ctx, r.found.tenant)
	if err != nil {
		return err
	}
	present := make(map[string]struct{}, len(listed))
	for _, o := range listed {
		present[o.Key] = struct{}{}
	}
	for i := range r.manifest.Documents {
		key, _ := tenantarchive.ObjectKey(r.manifest.Documents[i].Key)
		if _, ok := present[key]; !ok {
			return fmt.Errorf("after the load the document %s is not in storage", key)
		}
	}
	return nil
}
