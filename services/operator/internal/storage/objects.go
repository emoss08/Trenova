package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	emptySHA256    = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	metaPrefix     = "X-Amz-Meta-"
	objectTimeout  = 15 * time.Minute
	maxMetaHeaders = 2048
)

var ErrNoDocumentsBucket = errors.New(
	"the instance has no object storage of its own, so there is nowhere to keep the file",
)

type Object struct {
	Key         string
	Body        io.Reader
	Size        int64
	SHA256      string
	ContentType string
	Metadata    map[string]string
}

type ObjectInfo struct {
	Size     int64
	Metadata map[string]string
}

type Documents struct {
	client    *http.Client
	kube      kubernetes.Interface
	namespace string
	private   string
	now       func() time.Time
}

type DocumentsParams struct {
	Client          *http.Client
	Kube            kubernetes.Interface
	Namespace       string
	PrivateEndpoint string
	Now             func() time.Time
}

func NewDocuments(p *DocumentsParams) *Documents {
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: objectTimeout}
	}
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Documents{
		client: client, kube: p.Kube, namespace: p.Namespace, private: p.PrivateEndpoint, now: now,
	}
}

func (d *Documents) bucket(ctx context.Context) (Bucket, error) {
	secret, err := d.kube.CoreV1().
		Secrets(d.namespace).
		Get(ctx, values.DocumentsSecret, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return Bucket{}, ErrNoDocumentsBucket
	}
	if err != nil {
		return Bucket{}, fmt.Errorf("read the documents bucket's Secret: %w", err)
	}
	b := Bucket{
		Endpoint:        string(secret.Data[KeyEndpoint]),
		Region:          string(secret.Data[KeyRegion]),
		Name:            string(secret.Data[KeyBucket]),
		AccessKeyID:     string(secret.Data[KeyAccessKey]),
		SecretAccessKey: string(secret.Data[KeySecretKey]),
	}
	if d.private != "" {
		b.Endpoint = d.private
	}
	if b.Endpoint == "" || b.Name == "" || b.AccessKeyID == "" || b.SecretAccessKey == "" {
		return Bucket{}, ErrNoDocumentsBucket
	}
	return b, nil
}

func (d *Documents) Put(ctx context.Context, obj *Object) error {
	b, err := d.bucket(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, objectURL(&b, obj.Key), obj.Body)
	if err != nil {
		return fmt.Errorf("the upload request: %w", err)
	}
	req.ContentLength = obj.Size
	contentType := obj.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Content-Sha256", obj.SHA256)
	size := 0
	for k, v := range obj.Metadata {
		size += len(k) + len(v)
		req.Header.Set(metaPrefix+k, v)
	}
	if size > maxMetaHeaders {
		return errors.New("the object's metadata is larger than object storage accepts")
	}
	if err = d.sign(ctx, req, &b, obj.SHA256); err != nil {
		return err
	}
	res, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload to the documents bucket: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxErrorBody))
	if res.StatusCode/100 != 2 {
		return &ObjectError{Status: res.StatusCode, Op: "upload"}
	}
	return nil
}

func (d *Documents) Head(ctx context.Context, key string) (ObjectInfo, bool, error) {
	b, err := d.bucket(ctx)
	if err != nil {
		return ObjectInfo{}, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, objectURL(&b, key), http.NoBody)
	if err != nil {
		return ObjectInfo{}, false, fmt.Errorf("the lookup request: %w", err)
	}
	req.Header.Set("X-Amz-Content-Sha256", emptySHA256)
	if err = d.sign(ctx, req, &b, emptySHA256); err != nil {
		return ObjectInfo{}, false, err
	}
	res, err := d.client.Do(req)
	if err != nil {
		return ObjectInfo{}, false, fmt.Errorf("look up an object in the documents bucket: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	switch {
	case res.StatusCode == http.StatusNotFound:
		return ObjectInfo{}, false, nil
	case res.StatusCode/100 != 2:
		return ObjectInfo{}, false, &ObjectError{Status: res.StatusCode, Op: "lookup"}
	}
	info := ObjectInfo{Size: res.ContentLength, Metadata: map[string]string{}}
	for k, v := range res.Header {
		if name, ok := strings.CutPrefix(k, metaPrefix); ok && len(v) > 0 {
			info.Metadata[strings.ToLower(name)] = v[0]
		}
	}
	return info, true, nil
}

func (d *Documents) sign(ctx context.Context, req *http.Request, b *Bucket, payload string) error {
	creds := aws.Credentials{AccessKeyID: b.AccessKeyID, SecretAccessKey: b.SecretAccessKey}
	if err := v4.NewSigner().
		SignHTTP(ctx, creds, req, payload, "s3", b.Region, d.now().UTC()); err != nil {
		return fmt.Errorf("sign the documents bucket request: %w", err)
	}
	return nil
}

type ObjectError struct {
	Status int
	Op     string
}

func (e *ObjectError) Error() string {
	return fmt.Sprintf(
		"the documents bucket refused the %s: %d %s",
		e.Op,
		e.Status,
		http.StatusText(e.Status),
	)
}

func (e *ObjectError) Retryable() bool {
	return e.Status == http.StatusTooManyRequests || e.Status >= http.StatusInternalServerError
}

func objectURL(b *Bucket, key string) string {
	segments := strings.Split(key, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.TrimSuffix(b.Endpoint, "/") + "/" + b.Name + "/" + strings.Join(segments, "/")
}
