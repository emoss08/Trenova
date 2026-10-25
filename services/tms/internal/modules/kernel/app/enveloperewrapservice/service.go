package enveloperewrapservice

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultBatchSize  = 200
	maxReportedErrors = 20
)

var ErrStorageRequired = errors.New("object storage is required to rewrap sealed objects")

type Deps struct {
	Repository kernelports.EnvelopeRepository
	Rewrapper  kernelports.EnvelopeRewrapper
	Storage    kernelports.Client
	Logger     *zap.Logger
}

type Service struct {
	repo      kernelports.EnvelopeRepository
	rewrapper kernelports.EnvelopeRewrapper
	storage   kernelports.Client
	logger    *zap.Logger
}

func New(deps Deps) *Service {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repo:      deps.Repository,
		rewrapper: deps.Rewrapper,
		storage:   deps.Storage,
		logger:    logger,
	}
}

type Options struct {
	DryRun         bool
	BatchSize      int
	OrganizationID pulid.ID
}

type SourceReport struct {
	Name      string
	Scanned   int
	Rewrapped int
	Current   int
	Skipped   int
	NotBlobs  int
	Conflicts int
	Failed    int
	Errors    []string
}

type Report struct {
	DryRun  bool
	Sources []*SourceReport
}

func (r *Report) Failed() int {
	failed := 0
	for _, source := range r.Sources {
		failed += source.Failed
	}
	return failed
}

func (r *Report) NotBlobs() int {
	notBlobs := 0
	for _, source := range r.Sources {
		notBlobs += source.NotBlobs
	}
	return notBlobs
}

func (r *Report) Remaining() int {
	remaining := 0
	for _, source := range r.Sources {
		remaining += source.Failed + source.Conflicts
		if r.DryRun {
			remaining += source.Rewrapped
		}
	}
	return remaining
}

func (s *SourceReport) fail(row *kernelports.EnvelopeRow, err error) {
	s.Failed++
	if len(s.Errors) < maxReportedErrors {
		s.Errors = append(s.Errors, fmt.Sprintf("%s %s: %v", row.Tenant.OrgID, row.ID, err))
	}
}

type outcome int

const (
	outcomeCurrent outcome = iota
	outcomeRewrapped
	outcomeSkipped
	outcomeNotBlob
	outcomeConflict
)

func (s *SourceReport) record(result outcome) {
	switch result {
	case outcomeRewrapped:
		s.Rewrapped++
	case outcomeSkipped:
		s.Skipped++
	case outcomeNotBlob:
		s.NotBlobs++
	case outcomeConflict:
		s.Conflicts++
	case outcomeCurrent:
		s.Current++
	}
}

func (s *Service) Run(
	ctx context.Context,
	sources *kernelports.EnvelopeSources,
	opts Options,
) (*Report, error) {
	if len(sources.Objects) > 0 && s.storage == nil {
		return nil, ErrStorageRequired
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultBatchSize
	}

	tenants, err := s.repo.ListTenants(ctx)
	if err != nil {
		return nil, err
	}

	report := &Report{
		DryRun:  opts.DryRun,
		Sources: make([]*SourceReport, 0, len(sources.Columns)+len(sources.Objects)),
	}
	columns := make([]*SourceReport, len(sources.Columns))
	for i := range sources.Columns {
		columns[i] = &SourceReport{Name: sources.Columns[i].Name}
		report.Sources = append(report.Sources, columns[i])
	}
	objects := make([]*SourceReport, len(sources.Objects))
	for i := range sources.Objects {
		objects[i] = &SourceReport{Name: sources.Objects[i].Column.Name}
		report.Sources = append(report.Sources, objects[i])
	}

	for _, tenant := range tenants {
		if !opts.OrganizationID.IsNil() && tenant.OrgID != opts.OrganizationID {
			continue
		}
		tenantCtx := dbscope.WithTenant(ctx, tenant.DBTenant())
		for i := range sources.Columns {
			column := &sources.Columns[i]
			visit := func(row *kernelports.EnvelopeRow) {
				result, rowErr := s.rewrapRow(tenantCtx, column, row, opts.DryRun)
				if rowErr != nil {
					columns[i].fail(row, rowErr)
					return
				}
				columns[i].record(result)
			}
			if err = s.eachRow(
				tenantCtx,
				column,
				tenant,
				opts,
				visit,
				&columns[i].Scanned,
			); err != nil {
				return nil, err
			}
		}
		for i := range sources.Objects {
			object := &sources.Objects[i]
			visit := func(row *kernelports.EnvelopeRow) {
				result, rowErr := s.rewrapObject(tenantCtx, object, row, opts.DryRun)
				if rowErr != nil {
					objects[i].fail(row, rowErr)
					return
				}
				objects[i].record(result)
			}
			if err = s.eachRow(
				tenantCtx,
				&object.Column,
				tenant,
				opts,
				visit,
				&objects[i].Scanned,
			); err != nil {
				return nil, err
			}
		}
	}

	return report, nil
}

func (s *Service) eachRow(
	ctx context.Context,
	column *kernelports.EnvelopeColumn,
	tenant pagination.TenantInfo,
	opts Options,
	visit func(row *kernelports.EnvelopeRow),
	scanned *int,
) error {
	after := ""
	for {
		rows, err := s.repo.ListRows(ctx, &kernelports.ListEnvelopeRowsRequest{
			Column: column,
			Tenant: tenant,
			After:  after,
			Limit:  opts.BatchSize,
		})
		if err != nil {
			return err
		}
		for _, row := range rows {
			*scanned++
			visit(row)
		}
		if len(rows) < opts.BatchSize {
			return nil
		}
		after = rows[len(rows)-1].ID
	}
}

func (s *Service) rewrapRow(
	ctx context.Context,
	column *kernelports.EnvelopeColumn,
	row *kernelports.EnvelopeRow,
	dryRun bool,
) (outcome, error) {
	var (
		replacement string
		changed     bool
		err         error
	)
	if column.JSONMap {
		replacement, changed, err = s.rewrapMap(ctx, column, row)
	} else {
		if !s.rewrapper.IsEnvelope(row.Sealed) {
			return outcomeSkipped, nil
		}
		replacement, changed, err = s.rewrapper.Rewrap(ctx, &kernelports.EnvelopeRewrapRequest{
			Sealed: row.Sealed,
			Scopes: column.Scopes(row, ""),
			Rebind: column.Rebind,
		})
	}
	if err != nil {
		return outcomeCurrent, err
	}
	if !changed {
		return outcomeCurrent, nil
	}
	if dryRun {
		return outcomeRewrapped, nil
	}

	swapped, err := s.repo.Swap(ctx, &kernelports.SwapEnvelopeRequest{
		Column:      column,
		Row:         row,
		Replacement: replacement,
	})
	if err != nil {
		return outcomeCurrent, err
	}
	if !swapped {
		return outcomeConflict, nil
	}
	return outcomeRewrapped, nil
}

func (s *Service) rewrapMap(
	ctx context.Context,
	column *kernelports.EnvelopeColumn,
	row *kernelports.EnvelopeRow,
) (encoded string, changed bool, err error) {
	fields := make(map[string]any)
	if err = sonic.UnmarshalString(row.Sealed, &fields); err != nil {
		return "", false, fmt.Errorf("decode sealed fields: %w", err)
	}

	for field, value := range fields {
		sealed, ok := value.(string)
		if !ok || !s.rewrapper.IsEnvelope(sealed) {
			continue
		}
		replacement, fieldChanged, rewrapErr := s.rewrapper.Rewrap(
			ctx,
			&kernelports.EnvelopeRewrapRequest{
				Sealed: sealed,
				Scopes: column.Scopes(row, field),
				Rebind: column.Rebind,
			},
		)
		if rewrapErr != nil {
			return "", false, fmt.Errorf("field %s: %w", field, rewrapErr)
		}
		if fieldChanged {
			fields[field] = replacement
			changed = true
		}
	}
	if !changed {
		return row.Sealed, false, nil
	}

	encoded, err = sonic.MarshalString(fields)
	if err != nil {
		return "", false, fmt.Errorf("encode sealed fields: %w", err)
	}
	return encoded, true, nil
}

func (s *Service) rewrapObject(
	ctx context.Context,
	object *kernelports.EnvelopeObject,
	row *kernelports.EnvelopeRow,
	dryRun bool,
) (outcome, error) {
	downloaded, err := s.storage.Download(ctx, row.Sealed)
	if err != nil {
		return outcomeCurrent, fmt.Errorf("download %s: %w", row.Sealed, err)
	}
	body := bufio.NewReaderSize(downloaded.Body, 64<<10)
	prefix, peekErr := body.Peek(s.rewrapper.BlobPrefixSize())
	if peekErr != nil && !errors.Is(peekErr, io.EOF) {
		_ = downloaded.Body.Close()
		return outcomeCurrent, fmt.Errorf("read %s: %w", row.Sealed, peekErr)
	}
	if peekErr != nil || !s.rewrapper.IsBlob(prefix) {
		_ = downloaded.Body.Close()
		s.logger.Warn("a stored object is not a streaming blob and cannot be read",
			zap.String("source", object.Column.Name),
			zap.String("key", row.Sealed),
			zap.String("organizationId", row.Tenant.OrgID.String()),
			zap.String("rowId", row.ID))
		return outcomeNotBlob, nil
	}

	rewrapped, err := s.rewrapper.RewrapBlob(ctx, body, object.Column.Scopes(row, row.Sealed))
	if err != nil {
		_ = downloaded.Body.Close()
		return outcomeCurrent, err
	}
	if !rewrapped.Changed {
		_ = downloaded.Body.Close()
		return outcomeCurrent, nil
	}
	if dryRun {
		_ = downloaded.Body.Close()
		return outcomeRewrapped, nil
	}

	chunks, size, err := spool(body)
	closeErr := downloaded.Body.Close()
	if err != nil {
		return outcomeCurrent, fmt.Errorf("read %s: %w", row.Sealed, err)
	}
	defer chunks.Close()
	if closeErr != nil {
		return outcomeCurrent, fmt.Errorf("read %s: %w", row.Sealed, closeErr)
	}

	present, err := s.repo.Present(ctx, &object.Column, row)
	if err != nil {
		return outcomeCurrent, err
	}
	if !present {
		return outcomeConflict, nil
	}

	info, err := s.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         row.Sealed,
		ContentType: downloaded.ContentType,
		Size:        int64(len(rewrapped.Header)) + size,
		Body:        io.MultiReader(bytes.NewReader(rewrapped.Header), chunks),
		Metadata:    downloaded.Metadata,
	})
	if err != nil {
		return outcomeCurrent, fmt.Errorf("upload %s: %w", row.Sealed, err)
	}

	if object.Version != nil && info != nil {
		if err = s.repo.SwapVersion(ctx, &kernelports.SwapEnvelopeVersionRequest{
			Object:      object,
			Row:         row,
			Expected:    row.Extra[kernelports.EnvelopeColumnName(object.Version)],
			Replacement: info.VersionID,
		}); err != nil {
			return outcomeCurrent, err
		}
	}
	return outcomeRewrapped, nil
}

type spooled struct {
	*os.File
}

func (f spooled) Close() error {
	name := f.Name()
	closeErr := f.File.Close()
	if err := os.Remove(name); err != nil && closeErr == nil {
		return err
	}
	return closeErr
}

func spool(src io.Reader) (io.ReadCloser, int64, error) {
	file, err := os.CreateTemp("", "trenova-rewrap-*")
	if err != nil {
		return nil, 0, err
	}
	out := spooled{File: file}
	size, err := io.Copy(file, src)
	if err != nil {
		_ = out.Close()
		return nil, 0, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		_ = out.Close()
		return nil, 0, err
	}
	return out, size, nil
}
