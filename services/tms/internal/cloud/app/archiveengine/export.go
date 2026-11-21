package archiveengine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/archivecrypt"
	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/emoss08/trenova/internal/cloud/copybinary"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/pkg/dbscope"
)

type ExportRequest struct {
	Purpose    tenantarchive.Purpose
	InstanceID string
	Tenant     dbscope.Tenant
	Access     *tenantarchive.Access
	Progress   Reporter
	Beat       func()
}

type ExportResult struct {
	Manifest  *tenantarchive.Manifest
	Location  string
	SHA256    string
	SizeBytes int64
}

type exportRun struct {
	e        *Engine
	req      *ExportRequest
	bucket   archiveports.Bucket
	key      *archivecrypt.DataKey
	manifest *tenantarchive.Manifest
	natural  map[string]map[string]string
	progress tenantcommand.Progress
}

func (e *Engine) Export(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	if err := req.Access.CheckWrite(req.InstanceID, req.Purpose); err != nil {
		return nil, tenantarchive.Refuse(tenantarchive.RefusalLocation, err.Error())
	}
	bucket, err := e.buckets.Open(&req.Access.Location, &req.Access.Credentials)
	if err != nil {
		return nil, err
	}
	run := &exportRun{
		e:       e,
		req:     req,
		bucket:  bucket,
		natural: make(map[string]map[string]string),
		manifest: &tenantarchive.Manifest{
			FormatVersion:  tenantarchive.FormatVersion,
			SchemaVersion:  e.rows.SchemaVersion(),
			ReleaseVersion: e.release,
			InstanceID:     req.InstanceID,
			TakenAt:        e.now(),
			Purpose:        req.Purpose,
			Tables:         []tenantarchive.Table{},
			Documents:      []tenantarchive.Document{},
		},
	}
	if req.Purpose.Encrypted() {
		if err = run.keyUp(); err != nil {
			return nil, err
		}
	}
	if _, err = bucket.DeletePrefix(ctx, req.Access.Location.Prefix); err != nil {
		return nil, fmt.Errorf("clear the archive location before writing: %w", err)
	}
	if err = run.tables(ctx); err != nil {
		return nil, err
	}
	if err = run.documents(ctx); err != nil {
		return nil, err
	}
	return run.finish(ctx)
}

func (r *exportRun) keyUp() error {
	current := r.req.Access.CurrentKey()
	key, err := archivecrypt.NewDataKey()
	if err != nil {
		return err
	}
	wrapped, err := archivecrypt.Wrap(key, current.Material, r.req.InstanceID)
	if err != nil {
		return err
	}
	r.key = key
	r.manifest.Encryption = &tenantarchive.Encryption{
		Algorithm:      tenantarchive.EncryptionAlgorithm,
		SegmentBytes:   tenantarchive.SegmentBytes,
		KeyID:          current.ID,
		WrappedDataKey: wrapped,
	}
	return nil
}

func (r *exportRun) tables(ctx context.Context) error {
	snap, err := r.e.rows.OpenSnapshot(ctx, &archiveports.SnapshotRequest{
		Tenant:   r.req.Tenant,
		Purpose:  r.req.Purpose,
		Envelope: r.e.registry.tables(),
	})
	if err != nil {
		return err
	}
	defer snap.Close(ctx)

	outside, err := snap.Outside(ctx)
	if err != nil {
		return err
	}
	if len(outside) > 0 {
		names := make([]string, 0, len(outside))
		for _, d := range outside {
			names = append(
				names,
				fmt.Sprintf("%s (%d rows through %s)", d.Table, d.Rows, d.Constraint),
			)
		}
		return tenantarchive.Refuse(
			tenantarchive.RefusalTenant,
			"the tenant's rows refer to records of another organization, which an archive cannot carry: "+
				strings.Join(
					names,
					", ",
				),
		)
	}
	references, err := snap.References(ctx)
	if err != nil {
		return err
	}
	for _, ref := range references {
		byID := make(map[string]string, len(ref.Rows))
		for id, values := range ref.Rows {
			byID[id] = encodeNatural(values)
		}
		r.natural[ref.Table] = byID
	}
	plan := snap.Plan()
	r.manifest.OrganizationName = snap.OrganizationName()
	if r.req.Purpose == tenantarchive.PurposeExport {
		r.manifest.Omitted = append(r.manifest.Omitted, plan.Omitted...)
	}
	r.progress = tenantcommand.Progress{
		Phase:       tenantcommand.PhaseTables,
		TablesTotal: len(plan.Tables),
	}
	r.req.Progress.report(&r.progress)
	for _, table := range plan.Tables {
		entry, terr := r.table(ctx, snap, table)
		if terr != nil {
			return terr
		}
		r.manifest.Tables = append(r.manifest.Tables, *entry)
		r.progress.TablesDone++
		r.progress.Bytes += entry.SizeBytes
		r.req.Progress.report(&r.progress)
		if r.req.Beat != nil {
			r.req.Beat()
		}
	}
	return nil
}

type mappedColumn struct {
	index  int
	global string
}

func mappedColumns(table *archiveplan.PlannedTable) []mappedColumn {
	out := make([]mappedColumn, 0, len(table.Mapped))
	for column, global := range table.Mapped {
		out = append(out, mappedColumn{index: table.ColumnIndex(column), global: global})
	}
	return out
}

func (r *exportRun) rowFixer(
	ctx context.Context,
	table *archiveplan.PlannedTable,
) (func(*copybinary.Row) error, error) {
	var sealer *rowSealer
	if r.req.Purpose != tenantarchive.PurposeExport {
		var err error
		if sealer, err = r.e.registry.sealerFor(table, r.e.secrets, r.req.Tenant); err != nil {
			return nil, err
		}
	}
	if sealer == nil && len(table.Mapped) == 0 {
		return nil, nil //nolint:nilnil // a table that needs no rewrite streams through untouched
	}
	maps := mappedColumns(table)
	return func(row *copybinary.Row) error {
		for _, m := range maps {
			id, present := textOf(row, m.index, table.Types[m.index])
			if !present {
				continue
			}
			natural, ok := r.natural[m.global][id]
			if !ok {
				return fmt.Errorf(
					"%s refers to a %s row the snapshot did not name",
					table.Name,
					m.global,
				)
			}
			setText(row, m.index, table.Types[m.index], natural)
		}
		if sealer != nil {
			return sealer.unseal(ctx, row)
		}
		return nil
	}, nil
}

func (r *exportRun) table(
	ctx context.Context,
	snap archiveports.Snapshot,
	table *archiveplan.PlannedTable,
) (*tenantarchive.Table, error) {
	part := table.Part()
	fix, err := r.rowFixer(ctx, table)
	if err != nil {
		return nil, err
	}
	upload, uploaded := r.upload(ctx, r.req.Access.Location.Key(part))
	sw, err := archivestream.NewWriter(upload, streamOptions(part, r.key, true))
	if err != nil {
		_ = upload.CloseWithError(err)
		<-uploaded
		return nil, err
	}
	copied, copyDone := io.Pipe()
	done := transform(copied, sw, len(table.Columns), fix)
	copyErr := snap.CopyOut(ctx, table, copyDone)
	_ = copyDone.CloseWithError(copyErr)
	result := <-done
	closeErr := sw.Close()
	_ = upload.CloseWithError(errors.Join(copyErr, result.err, closeErr))
	putErr := <-uploaded
	if err = errors.Join(copyErr, result.err, closeErr, putErr); err != nil {
		return nil, fmt.Errorf("archive %s: %w", table.Name, err)
	}
	sum := sw.Sum()
	return &tenantarchive.Table{
		Name:      tenantarchive.QualifiedTable(table.Name),
		Rows:      result.rows,
		Key:       part,
		SizeBytes: sum.Bytes,
		SHA256:    sum.SHA256,
		Encoding:  tenantarchive.TableEncoding,
	}, nil
}

func (r *exportRun) upload(
	ctx context.Context,
	key string,
) (body *io.PipeWriter, done <-chan error) {
	pr, pw := io.Pipe()
	result := make(chan error, 1)
	go func() {
		err := r.bucket.Put(ctx, key, pr)
		_ = pr.CloseWithError(err)
		result <- err
	}()
	return pw, result
}

func (r *exportRun) finish(ctx context.Context) (*ExportResult, error) {
	r.progress.Phase = tenantcommand.PhaseManifest
	r.req.Progress.report(&r.progress)
	r.manifest.Seal()
	body, err := r.manifest.Encode()
	if err != nil {
		return nil, err
	}
	if len(body) > tenantarchive.MaxManifestBytes {
		return nil, tenantarchive.Refuse(tenantarchive.RefusalManifest,
			"the tenant's manifest would be larger than an archive allows")
	}
	if err = r.bucket.Put(
		ctx,
		r.req.Access.Location.Key(tenantarchive.ManifestName),
		bytes.NewReader(body),
	); err != nil {
		return nil, err
	}
	r.progress.Phase = tenantcommand.PhaseFinished
	r.req.Progress.report(&r.progress)
	return &ExportResult{
		Manifest:  r.manifest,
		Location:  r.req.Access.Location.String(),
		SHA256:    tenantarchive.ManifestSHA256(body),
		SizeBytes: r.manifest.TotalSizeBytes + int64(len(body)),
	}, nil
}
