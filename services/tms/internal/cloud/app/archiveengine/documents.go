package archiveengine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/emoss08/trenova/internal/cloud/archivecrypt"
	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/sourcegraph/conc/pool"
)

const (
	peekSize = 64 << 10
)

func (e *Engine) lister() (kernelports.PrefixLister, error) {
	lister, ok := e.storage.(kernelports.PrefixLister)
	if !ok {
		return nil, ErrNoObjectListing
	}
	return lister, nil
}

func (e *Engine) listObjects(
	ctx context.Context,
	tenant dbscope.Tenant,
) ([]kernelports.ListedObject, error) {
	lister, err := e.lister()
	if err != nil {
		return nil, err
	}
	listed := make([]kernelports.ListedObject, 0)
	for _, prefix := range e.prefixes(tenant) {
		if err = lister.ListPrefix(ctx, prefix, func(o kernelports.ListedObject) error {
			listed = append(listed, o)
			return nil
		}); err != nil {
			return nil, fmt.Errorf("list the tenant's documents: %w", err)
		}
	}
	return listed, nil
}

func (r *exportRun) documents(ctx context.Context) error {
	listed, err := r.e.listObjects(ctx, r.req.Tenant)
	if err != nil {
		return err
	}
	r.progress.Phase = tenantcommand.PhaseDocuments
	r.progress.DocumentsTotal = len(listed)
	r.req.Progress.report(&r.progress)

	docs := make([]tenantarchive.Document, len(listed))
	var mu sync.Mutex
	workers := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError().
		WithMaxGoroutines(DocumentConcurrency)
	for i := range listed {
		object := listed[i]
		workers.Go(func(ctx context.Context) error {
			doc, derr := r.document(ctx, object)
			if derr != nil {
				return derr
			}
			mu.Lock()
			docs[i] = *doc
			r.progress.DocumentsDone++
			r.progress.Bytes += doc.SizeBytes
			snapshot := r.progress
			mu.Unlock()
			r.req.Progress.report(&snapshot)
			if r.req.Beat != nil {
				r.req.Beat()
			}
			return nil
		})
	}
	if err = workers.Wait(); err != nil {
		return err
	}
	r.manifest.Documents = docs
	return nil
}

func (r *exportRun) document(
	ctx context.Context,
	object kernelports.ListedObject,
) (*tenantarchive.Document, error) {
	download, err := r.e.storage.Download(ctx, object.Key)
	if err != nil {
		return nil, fmt.Errorf("read the document %s: %w", object.Key, err)
	}
	defer download.Body.Close()
	plain, size, err := r.e.openStored(ctx, download, r.req.Tenant, object.Key)
	if err != nil {
		return nil, err
	}
	part := tenantarchive.DocumentPart(object.Key)
	upload, uploaded := r.upload(ctx, r.req.Access.Location.Key(part))
	sw, err := archivestream.NewWriter(upload, streamOptions(part, r.key, false))
	if err != nil {
		_ = upload.CloseWithError(err)
		<-uploaded
		return nil, err
	}
	_, copyErr := io.Copy(sw, plain)
	closeErr := sw.Close()
	_ = upload.CloseWithError(errors.Join(copyErr, closeErr))
	putErr := <-uploaded
	if err = errors.Join(copyErr, closeErr, putErr); err != nil {
		return nil, fmt.Errorf("archive the document %s: %w", object.Key, err)
	}
	if size >= 0 && sw.Plain() != size {
		return nil, fmt.Errorf("the document %s was %d bytes, not %d", object.Key, sw.Plain(), size)
	}
	sum := sw.Sum()
	doc := &tenantarchive.Document{Key: part, SizeBytes: sum.Bytes, SHA256: sum.SHA256}
	if download.ContentType != tenantarchive.DefaultContentType {
		doc.ContentType = download.ContentType
	}
	return doc, nil
}

func (e *Engine) openStored(
	ctx context.Context,
	download *kernelports.DownloadResult,
	tenant dbscope.Tenant,
	key string,
) (io.Reader, int64, error) {
	buffered := bufio.NewReaderSize(download.Body, peekSize)
	prefix, _ := buffered.Peek(e.blobs.BlobPrefixSize())
	if !e.blobs.IsBlob(prefix) {
		return buffered, download.Size, nil
	}
	opened, err := e.blobs.OpenBlob(
		ctx,
		buffered,
		download.Size,
		e.registry.objectScopes(tenant, key),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("open the sealed document %s: %w", key, err)
	}
	return opened.Body, opened.Size, nil
}

type documentLoad struct {
	e      *Engine
	run    *importRun
	tenant dbscope.Tenant
}

func (d *documentLoad) put(ctx context.Context, doc *tenantarchive.Document) error {
	objectKey, _ := tenantarchive.ObjectKey(doc.Key)
	reader, closeReader, err := d.run.part(ctx, doc.Key,
		archivestream.Sum{Bytes: doc.SizeBytes, SHA256: doc.SHA256}, false)
	if err != nil {
		return err
	}
	defer closeReader()
	purpose, sealed := d.run.sealed[objectKey]
	if !sealed {
		return d.upload(ctx, &kernelports.UploadParams{
			Key: objectKey, ContentType: doc.StoredContentType(), Size: -1, Body: reader,
		})
	}
	size, ok := archivecrypt.PlainSize(doc.SizeBytes)
	if !ok || d.run.key == nil {
		return tenantarchive.Refuse(tenantarchive.RefusalChecksum,
			"the archive's part "+doc.Key+" is shorter than its encryption allows")
	}
	blob, err := d.e.blobs.SealBlob(ctx, reader, size, &kernelports.SecretScope{
		Purpose: purpose,
		Tenant: pagination.TenantInfo{
			OrgID: d.tenant.OrganizationID,
			BuID:  d.tenant.BusinessUnitID,
		},
		ResourceID: objectKey,
	})
	if err != nil {
		return fmt.Errorf("seal the document %s: %w", objectKey, err)
	}
	return d.upload(ctx, &kernelports.UploadParams{
		Key:         objectKey,
		ContentType: doc.StoredContentType(),
		Size:        blob.Size,
		Body:        blob.Body,
		Metadata: map[string]string{
			kernelports.CryptoModeMetadataKey: kernelports.CryptoModeBlobV1,
		},
	})
}

func (d *documentLoad) upload(ctx context.Context, params *kernelports.UploadParams) error {
	if _, err := d.e.storage.Upload(ctx, params); err != nil {
		return fmt.Errorf("write the document %s: %w", params.Key, err)
	}
	return nil
}

func (e *Engine) clearObjects(ctx context.Context, tenant dbscope.Tenant) error {
	deleter, ok := e.storage.(kernelports.PrefixDeleter)
	if !ok {
		return ErrNoObjectListing
	}
	for _, prefix := range e.prefixes(tenant) {
		if _, err := deleter.DeletePrefix(ctx, prefix); err != nil {
			return fmt.Errorf("clear the tenant's documents: %w", err)
		}
	}
	return nil
}
