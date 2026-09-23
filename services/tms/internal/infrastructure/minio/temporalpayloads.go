package minio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	// temporalPayloadPrefix keeps offloaded payloads in the bucket the service
	// already owns rather than a bucket of their own. On S3 a bucket name is
	// global, so a fixed one would collide with somebody else's.
	temporalPayloadPrefix = "temporal-payloads/"

	// temporalPayloadRetention outlives any history that can still reference a
	// payload: the namespace's retention plus the longest a workflow in this
	// service stays open. A payload read after it expires fails the workflow
	// task that needs it, so this errs long.
	temporalPayloadRetention = 30

	temporalPayloadRuleID = "trenova-temporal-payload-retention"

	claimKey    = "key"
	claimSHA256 = "sha256"

	// temporalPayloadDriverName is written into every claim in workflow
	// history. Renaming it strands every payload already stored under it.
	temporalPayloadDriverName = "trenova-object-storage"
	temporalPayloadDriverType = "s3-compatible"
)

var errPayloadIntegrity = errors.New("offloaded temporal payload failed its integrity check")

// TemporalPayloadStore keeps payloads too large for workflow history in object
// storage and leaves a claim in their place. It is the claim check pattern,
// done through the SDK's own external storage support so neither workflow nor
// activity code ever sees it: an agent's transcript can grow past what history
// allows without anything above this layer knowing.
type TemporalPayloadStore struct {
	client *Client
	l      *zap.Logger

	retention sync.Once
}

var _ converter.StorageDriver = (*TemporalPayloadStore)(nil)

func NewTemporalPayloadStore(client storage.Client) (*TemporalPayloadStore, error) {
	c, ok := client.(*Client)
	if !ok {
		return nil, fmt.Errorf("temporal payload store needs the object storage client, got %T", client)
	}

	return &TemporalPayloadStore{client: c, l: c.l.Named("temporal-payloads")}, nil
}

func (*TemporalPayloadStore) Name() string { return temporalPayloadDriverName }

func (*TemporalPayloadStore) Type() string { return temporalPayloadDriverType }

// Store writes each payload under a key derived from its content, so a retried
// store of the same payload overwrites itself instead of leaving a second copy.
func (s *TemporalPayloadStore) Store(
	ctx converter.StorageDriverStoreContext,
	payloads []*commonpb.Payload,
) ([]converter.StorageDriverClaim, error) {
	s.retention.Do(func() { s.ensureRetention(ctx.Context) })

	scope := payloadScope(ctx.Target)
	claims := make([]converter.StorageDriverClaim, 0, len(payloads))
	for _, payload := range payloads {
		encoded, err := proto.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode temporal payload: %w", err)
		}

		sum := sha256.Sum256(encoded)
		digest := hex.EncodeToString(sum[:])
		key := path.Join(temporalPayloadPrefix, scope, digest)

		_, err = s.client.client.PutObject(
			ctx.Context,
			s.client.bucket,
			key,
			bytes.NewReader(encoded),
			int64(len(encoded)),
			minio.PutObjectOptions{ContentType: "application/x-protobuf"},
		)
		if err != nil {
			return nil, fmt.Errorf("store temporal payload: %w", err)
		}

		claims = append(claims, converter.StorageDriverClaim{
			ClaimData: map[string]string{claimKey: key, claimSHA256: digest},
		})
	}

	return claims, nil
}

func (s *TemporalPayloadStore) Retrieve(
	ctx converter.StorageDriverRetrieveContext,
	claims []converter.StorageDriverClaim,
) ([]*commonpb.Payload, error) {
	payloads := make([]*commonpb.Payload, 0, len(claims))
	for _, claim := range claims {
		key := claim.ClaimData[claimKey]
		if key == "" {
			return nil, errors.New("temporal payload claim has no object key")
		}

		payload, err := s.retrieve(ctx.Context, key, claim.ClaimData[claimSHA256])
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}

	return payloads, nil
}

func (s *TemporalPayloadStore) retrieve(
	ctx context.Context,
	key, digest string,
) (*commonpb.Payload, error) {
	obj, err := s.client.client.GetObject(ctx, s.client.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetch temporal payload %s: %w", key, err)
	}
	defer obj.Close()

	encoded, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read temporal payload %s: %w", key, err)
	}

	// The key already names the content, but a workflow decoded from a
	// corrupted or substituted object would fail far from here and in a way
	// nobody could trace back to storage. Checking costs one hash.
	sum := sha256.Sum256(encoded)
	if digest != "" && hex.EncodeToString(sum[:]) != digest {
		return nil, fmt.Errorf("%w: %s", errPayloadIntegrity, key)
	}

	payload := new(commonpb.Payload)
	if err = proto.Unmarshal(encoded, payload); err != nil {
		return nil, fmt.Errorf("decode temporal payload %s: %w", key, err)
	}

	return payload, nil
}

// ensureRetention merges an expiry rule for offloaded payloads into the
// bucket's lifecycle. It merges rather than replaces because the bucket is the
// service's shared one, and replacing its lifecycle would silently drop every
// rule somebody else put there. A bucket whose lifecycle cannot be managed is
// logged and left alone: the payloads are still stored and still readable, they
// just are not reclaimed.
func (s *TemporalPayloadStore) ensureRetention(ctx context.Context) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	current, err := s.client.client.GetBucketLifecycle(ctx, s.client.bucket)
	if err != nil {
		var resp minio.ErrorResponse
		if !errors.As(err, &resp) || resp.Code != "NoSuchLifecycleConfiguration" {
			s.l.Warn("could not read the bucket lifecycle; offloaded temporal payloads will not expire",
				zap.String("bucket", s.client.bucket),
				zap.Error(err),
			)
			return
		}
		current = lifecycle.NewConfiguration()
	}

	merged := withPayloadRetention(current)
	if err = s.client.client.SetBucketLifecycle(ctx, s.client.bucket, merged); err != nil {
		s.l.Warn("could not set the bucket lifecycle; offloaded temporal payloads will not expire",
			zap.String("bucket", s.client.bucket),
			zap.Error(err),
		)
	}
}

func withPayloadRetention(current *lifecycle.Configuration) *lifecycle.Configuration {
	rules := make([]lifecycle.Rule, 0, len(current.Rules)+1)
	for _, rule := range current.Rules {
		if rule.ID != temporalPayloadRuleID {
			rules = append(rules, rule)
		}
	}

	rules = append(rules, lifecycle.Rule{
		ID:         temporalPayloadRuleID,
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: temporalPayloadPrefix},
		Expiration: lifecycle.Expiration{Days: lifecycle.ExpirationDays(temporalPayloadRetention)},
	})

	return &lifecycle.Configuration{Rules: rules}
}

// payloadScope files a payload under the execution it belongs to, so an
// operator looking at the bucket can tell whose data it is. The content hash
// alone would be enough to find it again.
func payloadScope(target converter.StorageDriverTargetInfo) string {
	switch t := target.(type) {
	case converter.StorageDriverWorkflowInfo:
		return path.Join(safeSegment(t.Namespace), safeSegment(t.WorkflowID))
	case converter.StorageDriverActivityInfo:
		return path.Join(safeSegment(t.Namespace), "activity", safeSegment(t.ActivityID))
	default:
		return "unscoped"
	}
}

// safeSegment keeps a workflow ID from escaping its directory. Workflow IDs in
// this service use '/' and ':' as separators, both of which are fine in an
// object key, but ".." and an empty segment are not.
func safeSegment(value string) string {
	if value == "" || value == "." || value == ".." {
		return "_"
	}

	return url.PathEscape(value)
}
