package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const (
	KeyAccessKey = "accessKey"
	KeySecretKey = "secretKey"
	KeyEndpoint  = "endpoint"
	KeyRegion    = "region"
	KeyBucket    = "bucket"

	KeyBackupAccessKeyID = "ACCESS_KEY_ID"
	KeyBackupSecretKey   = "ACCESS_SECRET_KEY"
	KeyBackupRegion      = "ACCESS_REGION"
	KeyBackupPath        = "destinationPath"

	labelManagedBy = "app.kubernetes.io/managed-by"
	labelPartOf    = "app.kubernetes.io/part-of"
	labelComponent = "app.kubernetes.io/component"
	annotationNote = "trenova.app/storage"
	annotationCORS = "trenova.app/cors-origin"
	managedBy      = "trenova-operator"
	note           = "the instance's bucket credential from the Trenova Cloud control plane; " +
		"restored from it when missing, never a Helm resource"

	noneRecheck = 10 * time.Minute
	corsRetry   = 5 * time.Minute
	corsTimeout = 30 * time.Second
)

var (
	ErrInvalid = errors.New("the instance's storage is not valid")

	bucketPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	regionPattern   = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)
	endpointPattern = regexp.MustCompile(`^https://[a-z0-9]([a-z0-9.-]*[a-z0-9])?(:\d{1,5})?$`)
	privatePattern  = regexp.MustCompile(`^https?://[a-z0-9]([a-z0-9.-]*[a-z0-9])?(:\d{1,5})?$`)
	prefixPattern   = regexp.MustCompile(`^([A-Za-z0-9._-]+/)*$`)
	keyPattern      = regexp.MustCompile(`^[\x21-\x7E]{16,256}$`)
)

type ControlPlane interface {
	Storage(ctx context.Context) (instanceapi.InstanceStorage, error)
}

type Bucket struct {
	Endpoint        string
	Region          string
	Name            string
	AccessKeyID     string
	SecretAccessKey string
}

type CORS interface {
	Allow(ctx context.Context, b *Bucket, origins []string) error
}

type Store struct {
	private      string
	client       kubernetes.Interface
	cp           ControlPlane
	cors         CORS
	namespace    string
	logger       *slog.Logger
	now          func() time.Time
	noneUntil    time.Time
	noneRevision int64
	corsAfter    time.Time
}

type Params struct {
	PrivateEndpoint string
	Client          kubernetes.Interface
	ControlPlane    ControlPlane
	CORS            CORS
	Namespace       string
	Logger          *slog.Logger
	Now             func() time.Time
}

func New(p *Params) *Store {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Store{
		private:   p.PrivateEndpoint,
		client:    p.Client,
		cp:        p.ControlPlane,
		cors:      p.CORS,
		namespace: p.Namespace,
		logger:    p.Logger,
		now:       now,
	}
}

func (s *Store) Ensure(
	ctx context.Context,
	hostnames []string,
	revision int64,
) (values.Storage, error) {
	if revision == s.noneRevision && s.now().Before(s.noneUntil) {
		return values.Storage{}, nil
	}
	secrets := s.client.CoreV1().Secrets(s.namespace)
	docs, docsErr := secrets.Get(ctx, values.DocumentsSecret, metav1.GetOptions{})
	backups, backupsErr := secrets.Get(ctx, values.BackupsSecret, metav1.GetOptions{})
	for _, err := range []error{docsErr, backupsErr} {
		if err != nil && !apierrors.IsNotFound(err) {
			return values.Storage{}, fmt.Errorf(
				"read the storage Secrets in %s: %w",
				s.namespace,
				err,
			)
		}
	}
	if docsErr == nil && backupsErr == nil {
		if loc, ok := held(docs.Data, backups.Data); ok {
			s.allowOrigins(ctx, docs, hostnames)
			loc.PrivateEndpoint = s.private
			return loc, nil
		}
		s.logger.Warn(
			"the storage Secrets are incomplete or invalid; restoring them from the control plane",
		)
	}
	fetched, err := s.cp.Storage(ctx)
	if errors.Is(err, controlplane.ErrNoStorage) {
		s.noneUntil, s.noneRevision = s.now().Add(noneRecheck), revision
		s.logger.Info(
			"the instance has no object storage of its own; documents stay in the chart's store, without backups",
			"revision",
			revision,
			"recheck",
			s.noneUntil,
		)
		return values.Storage{}, nil
	}
	s.noneUntil, s.noneRevision = time.Time{}, 0
	if err != nil {
		return values.Storage{}, fmt.Errorf("fetch the instance's storage: %w", err)
	}
	if err = Validate(&fetched); err != nil {
		return values.Storage{}, fmt.Errorf("the control plane's storage: %w", err)
	}
	if docs, err = s.write(
		ctx,
		values.DocumentsSecret,
		docs,
		docsErr,
		documentsData(&fetched),
		"storage",
	); err != nil {
		return values.Storage{}, err
	}
	if _, err = s.write(
		ctx,
		values.BackupsSecret,
		backups,
		backupsErr,
		backupsData(&fetched),
		"backup-storage",
	); err != nil {
		return values.Storage{}, err
	}
	s.allowOrigins(ctx, docs, hostnames)
	loc := location(&fetched)
	loc.PrivateEndpoint = s.private
	return loc, nil
}

func ValidPrivateEndpoint(endpoint string) bool {
	return endpoint == "" || privatePattern.MatchString(endpoint)
}

func (s *Store) write(
	ctx context.Context,
	name string,
	cur *corev1.Secret,
	getErr error,
	data map[string][]byte,
	component string,
) (*corev1.Secret, error) {
	secrets := s.client.CoreV1().Secrets(s.namespace)
	if apierrors.IsNotFound(getErr) {
		created, err := secrets.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   s.namespace,
				Labels:      labels(nil, component),
				Annotations: annotations(nil),
			},
			Type: corev1.SecretTypeOpaque,
			Data: data,
		}, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("create the Secret %s/%s: %w", s.namespace, name, err)
		}
		s.logger.Info("storage Secret created", "secret", s.namespace+"/"+name)
		return created, nil
	}
	next := cur.DeepCopy()
	next.Labels = labels(next.Labels, component)
	next.Annotations = annotations(next.Annotations)
	merged := make(map[string][]byte, len(next.Data)+len(data))
	maps.Copy(merged, next.Data)
	maps.Copy(merged, data)
	next.Data = merged
	updated, err := secrets.Update(ctx, next, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("update the Secret %s/%s: %w", s.namespace, name, err)
	}
	s.logger.Info("storage Secret restored", "secret", s.namespace+"/"+name)
	return updated, nil
}

func (s *Store) allowOrigins(ctx context.Context, docs *corev1.Secret, hostnames []string) {
	origins := make([]string, 0, len(hostnames))
	for _, h := range hostnames {
		if h != "" && !slices.Contains(origins, "https://"+h) {
			origins = append(origins, "https://"+h)
		}
	}
	recorded := strings.Join(origins, ",")
	if s.cors == nil || len(origins) == 0 || docs.Annotations[annotationCORS] == recorded ||
		s.now().Before(s.corsAfter) {
		return
	}
	run, cancel := context.WithTimeout(ctx, corsTimeout)
	defer cancel()
	endpoint := string(docs.Data[KeyEndpoint])
	if s.private != "" {
		endpoint = s.private
	}
	err := s.cors.Allow(run, &Bucket{
		Endpoint:        endpoint,
		Region:          string(docs.Data[KeyRegion]),
		Name:            string(docs.Data[KeyBucket]),
		AccessKeyID:     string(docs.Data[KeyAccessKey]),
		SecretAccessKey: string(docs.Data[KeySecretKey]),
	}, origins)
	if err != nil {
		s.corsAfter = s.now().Add(corsRetry)
		s.logger.Warn("cannot let the browser read the documents bucket; retrying", "error", err)
		return
	}
	next := docs.DeepCopy()
	if next.Annotations == nil {
		next.Annotations = map[string]string{}
	}
	next.Annotations[annotationCORS] = recorded
	if _, err = s.client.CoreV1().
		Secrets(s.namespace).
		Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		s.logger.Warn("cannot record the documents bucket's CORS rules", "error", err)
		return
	}
	s.logger.Info("the documents bucket admits the instance's origins", "origins", recorded)
}

func Validate(v *instanceapi.InstanceStorage) error {
	var errs []error
	if !endpointPattern.MatchString(v.Endpoint) {
		errs = append(
			errs,
			fmt.Errorf("%w: endpoint must be an https URL without a path", ErrInvalid),
		)
	}
	if !regionPattern.MatchString(v.Region) {
		errs = append(errs, fmt.Errorf("%w: region must be 1 to 40 of a-z, 0-9 and -", ErrInvalid))
	}
	if !prefixPattern.MatchString(v.BackupsPrefix) {
		errs = append(
			errs,
			fmt.Errorf("%w: backupsPrefix must be path segments ending in /", ErrInvalid),
		)
	}
	for what, b := range map[string]instanceapi.StorageBucket{"documents": v.Documents, "backups": v.Backups} {
		if !bucketPattern.MatchString(b.Bucket) {
			errs = append(errs, fmt.Errorf("%w: the %s bucket's name", ErrInvalid, what))
		}
		if !keyPattern.MatchString(b.AccessKeyId) || !keyPattern.MatchString(b.SecretAccessKey) {
			errs = append(errs, fmt.Errorf("%w: the %s bucket's credential", ErrInvalid, what))
		}
	}
	if v.Documents.Bucket == v.Backups.Bucket || v.Documents.AccessKeyId == v.Backups.AccessKeyId {
		errs = append(
			errs,
			fmt.Errorf(
				"%w: documents and backups must not share a bucket or a credential",
				ErrInvalid,
			),
		)
	}
	return errors.Join(errs...)
}

func documentsData(v *instanceapi.InstanceStorage) map[string][]byte {
	return map[string][]byte{
		KeyAccessKey: []byte(v.Documents.AccessKeyId),
		KeySecretKey: []byte(v.Documents.SecretAccessKey),
		KeyEndpoint:  []byte(v.Endpoint),
		KeyRegion:    []byte(v.Region),
		KeyBucket:    []byte(v.Documents.Bucket),
	}
}

func backupsData(v *instanceapi.InstanceStorage) map[string][]byte {
	return map[string][]byte{
		KeyBackupAccessKeyID: []byte(v.Backups.AccessKeyId),
		KeyBackupSecretKey:   []byte(v.Backups.SecretAccessKey),
		KeyBackupRegion:      []byte(v.Region),
		KeyBackupPath:        []byte(destination(v)),
		KeyEndpoint:          []byte(v.Endpoint),
	}
}

func destination(v *instanceapi.InstanceStorage) string {
	return "s3://" + v.Backups.Bucket + "/" + strings.TrimPrefix(v.BackupsPrefix, "/")
}

func location(v *instanceapi.InstanceStorage) values.Storage {
	return values.Storage{
		Endpoint:        v.Endpoint,
		Region:          v.Region,
		DocumentsBucket: v.Documents.Bucket,
		BackupsPath:     destination(v),
	}
}

func held(docs, backups map[string][]byte) (values.Storage, bool) {
	path := string(backups[KeyBackupPath])
	bucket, prefix, _ := strings.Cut(strings.TrimPrefix(path, "s3://"), "/")
	v := instanceapi.InstanceStorage{
		Endpoint:      string(docs[KeyEndpoint]),
		Region:        string(docs[KeyRegion]),
		BackupsPrefix: prefix,
		Documents: instanceapi.StorageBucket{
			Bucket:          string(docs[KeyBucket]),
			AccessKeyId:     string(docs[KeyAccessKey]),
			SecretAccessKey: string(docs[KeySecretKey]),
		},
		Backups: instanceapi.StorageBucket{
			Bucket:          bucket,
			AccessKeyId:     string(backups[KeyBackupAccessKeyID]),
			SecretAccessKey: string(backups[KeyBackupSecretKey]),
		},
	}
	if !strings.HasPrefix(path, "s3://") || string(backups[KeyBackupRegion]) != v.Region ||
		string(backups[KeyEndpoint]) != v.Endpoint || Validate(&v) != nil {
		return values.Storage{}, false
	}
	return location(&v), true
}

func labels(existing map[string]string, component string) map[string]string {
	out := maps.Clone(existing)
	if out == nil {
		out = map[string]string{}
	}
	out[labelManagedBy] = managedBy
	out[labelPartOf] = "trenova"
	out[labelComponent] = component
	return out
}

func annotations(existing map[string]string) map[string]string {
	out := maps.Clone(existing)
	if out == nil {
		out = map[string]string{}
	}
	out[annotationNote] = note
	return out
}
