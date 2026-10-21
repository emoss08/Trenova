// Package storedobject reads and writes a document's bytes as stored: sealed
// as a streaming blob under the associated data that binds them to their
// tenant and path. Every reader and writer of document bytes goes through it,
// so none can read a stored object without the check that it is the tenant's,
// and none holds a whole document to encrypt or decrypt it.
package storedobject

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"maps"

	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const CryptoVersion = int16(2)

var (
	ErrNotSealed    = errors.New("the stored object is not a sealed blob")
	ErrTooLarge     = errors.New("the stored object is larger than the reader allows")
	ErrNotAvailable = errors.New("document encryption is not configured")
)

// AAD is the associated data a document object is sealed under.
func AAD(tenantInfo pagination.TenantInfo, storagePath string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     storagePath,
	}
}

// Object is an opened stored object: Body yields the plaintext and fails if
// the object was cut short or altered, so its last bytes are trusted only once
// it has returned io.EOF.
type Object struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
	Metadata    map[string]string
}

type openedBody struct {
	io.Reader
	closer io.Closer
}

func (b *openedBody) Close() error {
	return b.closer.Close()
}

// Open downloads the object at key and opens it under aad as it is read.
func Open(
	ctx context.Context,
	storage kernelports.Client,
	encryption *encryptionservice.Service,
	key string,
	aad encryptionservice.AAD,
) (*Object, error) {
	object, found, err := OpenOptional(ctx, storage, encryption, key, aad)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errortypes.NewDatabaseError("Failed to download document").
			WithInternal(kernelports.ErrObjectNotFound)
	}
	return object, nil
}

// OpenOptional is Open for an object that may not exist yet: a missing object
// is reported as not found rather than as an error.
func OpenOptional(
	ctx context.Context,
	storage kernelports.Client,
	encryption *encryptionservice.Service,
	key string,
	aad encryptionservice.AAD,
) (*Object, bool, error) {
	if encryption == nil {
		return nil, false, errortypes.NewBusinessError("Document encryption is not configured").
			WithInternal(ErrNotAvailable)
	}
	download, err := storage.Download(ctx, key)
	if errors.Is(err, kernelports.ErrObjectNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, errortypes.NewDatabaseError("Failed to download document").
			WithInternal(err)
	}
	if download == nil || download.Body == nil {
		return nil, false, errortypes.NewDatabaseError("Failed to download document")
	}

	buffered := bufio.NewReaderSize(download.Body, 64<<10)
	prefix, err := buffered.Peek(encryptionservice.BlobPrefixSize())
	if err != nil || !encryptionservice.IsBlob(prefix) {
		_ = download.Body.Close()
		return nil, false, errortypes.NewBusinessError("Stored document is not encrypted").
			WithInternal(ErrNotSealed)
	}
	sealedSize := download.Size
	if sealedSize <= 0 {
		sealedSize = -1
	}
	opened, err := encryption.OpenBlob(ctx, buffered, sealedSize, aad)
	if err != nil {
		_ = download.Body.Close()
		return nil, false, errortypes.NewBusinessError("Failed to decrypt document").
			WithInternal(err)
	}

	return &Object{
		Body:        &openedBody{Reader: opened, closer: download.Body},
		Size:        opened.Size(),
		ContentType: download.ContentType,
		Metadata:    download.Metadata,
	}, true, nil
}

// Read opens the object and reads all of it, refusing one larger than limit.
// It is for objects a caller must hold whole anyway: renders, previews, files
// handed to a model.
func Read(
	ctx context.Context,
	storage kernelports.Client,
	encryption *encryptionservice.Service,
	key string,
	aad encryptionservice.AAD,
	limit int64,
) ([]byte, error) {
	data, found, err := ReadOptional(ctx, storage, encryption, key, aad, limit)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errortypes.NewDatabaseError("Failed to download document").
			WithInternal(kernelports.ErrObjectNotFound)
	}
	return data, nil
}

func ReadOptional(
	ctx context.Context,
	storage kernelports.Client,
	encryption *encryptionservice.Service,
	key string,
	aad encryptionservice.AAD,
	limit int64,
) (data []byte, found bool, err error) {
	object, opened, openErr := OpenOptional(ctx, storage, encryption, key, aad)
	if openErr != nil || !opened {
		return nil, opened, openErr
	}
	defer object.Body.Close()

	if object.Size > limit {
		return nil, false, errortypes.NewBusinessError("Stored document is too large to read").
			WithInternal(fmt.Errorf("%w: %d bytes over %d", ErrTooLarge, object.Size, limit))
	}
	data, err = io.ReadAll(io.LimitReader(object.Body, limit+1))
	if err != nil {
		return nil, false, errortypes.NewBusinessError("Failed to decrypt document").
			WithInternal(err)
	}
	if int64(len(data)) > limit {
		return nil, false, errortypes.NewBusinessError("Stored document is too large to read").
			WithInternal(ErrTooLarge)
	}
	return data, true, nil
}

// Source is an object as the imaging service reads it: each attempt downloads
// it again and opens it as it streams, so no copy of the document is held.
func Source(
	storage kernelports.Client,
	encryption *encryptionservice.Service,
	key string,
	aad encryptionservice.AAD,
	contentType string,
	size int64,
) kernelports.ImagingSource {
	return kernelports.ImagingSource{
		ContentType: contentType,
		Size:        size,
		Open: func(ctx context.Context) (io.ReadCloser, error) {
			object, err := Open(ctx, storage, encryption, key, aad)
			if err != nil {
				return nil, err
			}
			if object.Size >= 0 && object.Size != size {
				_ = object.Body.Close()
				return nil, fmt.Errorf(
					"%w: %s holds %d bytes and its record says %d",
					kernelports.ErrImagingInvalidDocument, key, object.Size, size,
				)
			}
			return object.Body, nil
		},
	}
}

// WriteRequest is one object to seal and store. When Scan is set the bytes
// are scanned for malware as they pass, and the object is completed only once
// the scanner has passed them: the last chunk is held back until the verdict,
// so an infected file never becomes an object.
type WriteRequest struct {
	Key         string
	ContentType string
	Metadata    map[string]string
	AAD         encryptionservice.AAD
	Body        io.Reader
	Size        int64
	Gate        contentscanports.Gate
	Scan        *contentscanports.StreamRequest
}

type Written struct {
	Info   *kernelports.FileInfo
	SHA256 string
	Scan   *contentscanports.ScanResult
}

type hashingReader struct {
	src    io.Reader
	digest hash.Hash
}

func (r *hashingReader) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	r.digest.Write(p[:n])
	return n, err
}

// Write seals req.Body as it is read and streams it to storage. Nothing holds
// the whole file: memory is a chunk on each side of the cipher. A refusal by
// the malware scanner is returned as the scanner's own error, for
// contentscanports.RefusalError.
func Write(
	ctx context.Context,
	storage kernelports.Client,
	encryption *encryptionservice.Service,
	req *WriteRequest,
) (*Written, error) {
	if encryption == nil {
		return nil, errortypes.NewBusinessError("Document encryption is not configured").
			WithInternal(ErrNotAvailable)
	}

	hashed := &hashingReader{src: req.Body, digest: sha256.New()}
	var plaintext io.Reader = hashed
	var gated *contentscanports.Gated
	if req.Scan != nil {
		scan := *req.Scan
		scan.Size = req.Size
		stream, err := contentscanports.Open(ctx, req.Gate, &scan)
		if err != nil {
			return nil, err
		}
		gated = contentscanports.NewGated(hashed, stream)
		plaintext = gated
	}

	sealed, err := encryption.SealBlob(ctx, plaintext, req.Size, req.AAD)
	if err != nil {
		if gated != nil {
			gated.Abort()
		}
		return nil, errortypes.NewBusinessError("Failed to encrypt document").WithInternal(err)
	}

	metadata := make(map[string]string, len(req.Metadata)+1)
	maps.Copy(metadata, req.Metadata)
	metadata["crypto_mode"] = encryptionservice.CryptoModeBlobV1

	info, err := storage.Upload(ctx, &kernelports.UploadParams{
		Key:         req.Key,
		ContentType: req.ContentType,
		Size:        sealed.Size(),
		Body:        sealed,
		Metadata:    metadata,
	})
	if gated != nil {
		if scanErr := gated.Err(); scanErr != nil {
			return nil, scanErr
		}
		if err != nil {
			gated.Abort()
		}
	}
	if err != nil {
		if errors.Is(err, encryptionservice.ErrBlobSize) {
			return nil, errortypes.NewConflictError(
				"Uploaded file size does not match the original file",
			).WithInternal(err)
		}
		return nil, errortypes.NewDatabaseError("Failed to store document").WithInternal(err)
	}

	written := &Written{Info: info, SHA256: hex.EncodeToString(hashed.digest.Sum(nil))}
	if gated != nil {
		written.Scan = gated.Result()
	}
	return written, nil
}
