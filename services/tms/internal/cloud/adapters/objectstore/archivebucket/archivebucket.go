package archivebucket

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/shared/httpsafe"
	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const PartSize = 16 << 20

const (
	dialTimeout      = 10 * time.Second
	dialKeepAlive    = 30 * time.Second
	handshakeTimeout = 10 * time.Second
	headerTimeout    = 60 * time.Second
	idleTimeout      = 90 * time.Second
	maxIdle          = 16
)

var ErrEndpoint = errors.New(
	"the archive endpoint must be a host or an https URL with a host and no path",
)

type Params struct {
	fx.In

	Managed *managedconfig.Config
}

type Buckets struct {
	allowed   func(host string) bool
	insecure  bool
	policy    httpsafe.Policy
	transport http.RoundTripper
}

func New(p Params) archiveports.Buckets {
	return Buckets{
		allowed:   p.Managed.Archive.EndpointAllowed,
		transport: guardedTransport(httpsafe.Policy{}),
	}
}

type ConfiguredEndpoint struct {
	Endpoint    string
	Development bool
}

func ForConfiguredEndpoint(c ConfiguredEndpoint) archiveports.Buckets {
	want, _, err := endpointHost(c.Endpoint, c.Development)
	policy := httpsafe.Policy{AllowPrivateNetworks: c.Development}
	return Buckets{
		allowed: func(host string) bool {
			return err == nil && strings.EqualFold(strings.TrimSuffix(host, "."), hostName(want))
		},
		insecure:  c.Development,
		policy:    policy,
		transport: guardedTransport(policy),
	}
}

func guardedTransport(policy httpsafe.Policy) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: dialKeepAlive,
		Control:   httpsafe.DialControl(policy),
	}
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdle,
		IdleConnTimeout:       idleTimeout,
		TLSHandshakeTimeout:   handshakeTimeout,
		ResponseHeaderTimeout: headerTimeout,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
}

func hostName(host string) string {
	if name, _, err := net.SplitHostPort(host); err == nil {
		return strings.Trim(name, "[]")
	}
	return strings.Trim(host, "[]")
}

type bucket struct {
	client *minio.Client
	name   string
}

func (b Buckets) Open(
	location *tenantarchive.Location,
	creds *tenantarchive.Credentials,
) (archiveports.Bucket, error) {
	host, secure, err := endpointHost(location.Endpoint, b.insecure)
	if err != nil {
		return nil, tenantarchive.Refuse(tenantarchive.RefusalLocation, err.Error())
	}
	name := hostName(host)
	if httpsafe.ValidateHost(name, b.policy) != nil || !b.allowed(name) {
		return nil, tenantarchive.Refuse(tenantarchive.RefusalLocation,
			"the archive endpoint is not an object store this server may reach")
	}
	client, err := minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(creds.AccessKey, creds.SecretKey, creds.SessionToken),
		Secure:       secure,
		Region:       location.Region,
		BucketLookup: minio.BucketLookupAuto,
		Transport:    b.transport,
	})
	if err != nil {
		return nil, fmt.Errorf("open the archive location: %w", err)
	}
	return &bucket{client: client, name: location.Bucket}, nil
}

func endpointHost(raw string, insecure bool) (host string, secure bool, err error) {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		if raw == "" || strings.ContainsAny(raw, "/@?#") {
			return "", false, ErrEndpoint
		}
		return raw, true, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Path != "" && u.Path != "/") ||
		u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", false, ErrEndpoint
	}
	switch {
	case u.Scheme == "https":
		return u.Host, true, nil
	case u.Scheme == "http" && insecure:
		return u.Host, false, nil
	default:
		return "", false, ErrEndpoint
	}
}

func (b *bucket) Put(ctx context.Context, key string, body io.Reader) error {
	if _, err := b.client.PutObject(ctx, b.name, key, body, -1, minio.PutObjectOptions{
		ContentType:           "application/octet-stream",
		PartSize:              PartSize,
		NumThreads:            1,
		ConcurrentStreamParts: false,
		DisableContentSha256:  true,
	}); err != nil {
		return fmt.Errorf("write %s to the archive location: %w", key, err)
	}
	return nil
}

func (b *bucket) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := b.client.GetObject(ctx, b.name, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("read %s from the archive location: %w", key, err)
	}
	if _, err = object.Stat(); err != nil {
		_ = object.Close()
		if minio.ToErrorResponse(err).StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%s: %w", key, archiveports.ErrMissing)
		}
		return nil, fmt.Errorf("read %s from the archive location: %w", key, err)
	}
	return object, nil
}

func (b *bucket) List(
	ctx context.Context,
	prefix string,
	fn func(key string, size int64) error,
) error {
	listCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for object := range b.client.ListObjects(listCtx, b.name, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			return fmt.Errorf("list the archive location: %w", object.Err)
		}
		if err := fn(object.Key, object.Size); err != nil {
			return err
		}
	}
	return nil
}

func (b *bucket) Children(ctx context.Context, prefix string) ([]string, error) {
	listCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	out := make([]string, 0)
	for object := range b.client.ListObjects(listCtx, b.name, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	}) {
		if object.Err != nil {
			return nil, fmt.Errorf("list the archive location: %w", object.Err)
		}
		if strings.HasSuffix(object.Key, "/") {
			out = append(out, object.Key)
		}
	}
	return out, nil
}

func (b *bucket) DeletePrefix(ctx context.Context, prefix string) (int64, error) {
	if prefix == "" || !strings.HasSuffix(prefix, "/") {
		return 0, ErrEndpoint
	}
	listCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	objects := make(chan minio.ObjectInfo)
	var listErr error
	var queued int64
	go func() {
		defer close(objects)
		for object := range b.client.ListObjects(listCtx, b.name, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		}) {
			if object.Err != nil {
				listErr = object.Err
				cancel()
				return
			}
			select {
			case objects <- object:
				queued++
			case <-listCtx.Done():
				return
			}
		}
	}()
	var removeErr error
	var failed int64
	for result := range b.client.RemoveObjects(ctx, b.name, objects, minio.RemoveObjectsOptions{}) {
		failed++
		removeErr = result.Err
	}
	if listErr != nil {
		return queued - failed, fmt.Errorf("list %s: %w", prefix, listErr)
	}
	if removeErr != nil {
		return queued - failed, fmt.Errorf("delete under %s: %w", prefix, removeErr)
	}
	return queued - failed, nil
}
