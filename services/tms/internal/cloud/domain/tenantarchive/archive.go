package tenantarchive

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	FormatVersion = 1
	ManifestName  = "manifest.json"
	TableEncoding = "pg_copy_binary_zstd"
	SegmentBytes  = 1 << 20
	KeySize       = 32

	EncryptionAlgorithm = "aes256gcm_stream_v1"

	TablesDir    = "tables/"
	DocumentsDir = "documents/"

	DefaultContentType = "application/octet-stream"

	TenantsRoot = "tenants/"

	maxPrefixLength   = 512
	maxBucketLength   = 63
	maxEndpointLength = 255
	maxRegionLength   = 64
	maxKeyIDLength    = 64
	maxArchiveKeys    = 20
)

type Purpose string

const (
	PurposeFinal   = Purpose("final")
	PurposeExport  = Purpose("export")
	PurposeMove    = Purpose("move")
	PurposeNightly = Purpose("nightly")
)

func (p Purpose) Valid() bool {
	switch p {
	case PurposeFinal, PurposeExport, PurposeMove, PurposeNightly:
		return true
	default:
		return false
	}
}

func (p Purpose) Commanded() bool {
	return p == PurposeFinal || p == PurposeExport || p == PurposeMove
}

func (p Purpose) Encrypted() bool {
	return p != PurposeExport
}

func (p Purpose) Importable() bool {
	return p != PurposeExport
}

var (
	ErrLocationInvalid = errors.New(
		"the archive location must name an endpoint, a bucket and a prefix under tenants/<instanceId>/ that ends in a slash",
	)
	ErrCredentialsMissing = errors.New("the archive location needs an access key and a secret key")
	ErrKeyInvalid         = fmt.Errorf(
		"archiveKeys must hold at most %d keys of %d bytes, each with an id of 1 to %d characters, exactly one current",
		maxArchiveKeys,
		KeySize,
		maxKeyIDLength,
	)
	ErrKeyNotExpected = errors.New(
		"an export archive is the customer's copy and is not encrypted, so it takes no archive key",
	)
	ErrKeyUnknown = errors.New(
		"the archive was sealed with an archive key the command does not carry",
	)
	ErrNightlyPrefix = errors.New("the nightly archive storage's prefix must be " + TenantsRoot)

	bucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.\-]{1,61}[a-z0-9]$`)
	keyIDPattern  = regexp.MustCompile(`^\S{1,64}$`)
	segmentChars  = regexp.MustCompile(`^[A-Za-z0-9._\-]+$`)
)

type Location struct {
	Endpoint string `json:"endpoint"`
	Region   string `json:"region,omitempty"`
	Bucket   string `json:"bucket"`
	Prefix   string `json:"prefix"`
}

func (l *Location) Check(instanceID string) error {
	return l.CheckUnder(TenantPrefix(instanceID))
}

func (l *Location) CheckUnder(root string) error {
	if strings.TrimSpace(l.Endpoint) == "" || len(l.Endpoint) > maxEndpointLength ||
		strings.ContainsAny(l.Endpoint, " \t\r\n@?#") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(l.Endpoint)), "http://") ||
		len(l.Region) > maxRegionLength ||
		!bucketPattern.MatchString(l.Bucket) || len(l.Bucket) > maxBucketLength {
		return ErrLocationInvalid
	}
	if !ValidPrefixUnder(l.Prefix, root) {
		return ErrLocationInvalid
	}
	return nil
}

func ValidPrefixUnder(prefix, root string) bool {
	if len(prefix) > maxPrefixLength || !strings.HasSuffix(prefix, "/") ||
		!strings.HasPrefix(prefix, root) {
		return false
	}
	for _, segment := range strings.Split(strings.TrimSuffix(prefix, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." ||
			!segmentChars.MatchString(segment) {
			return false
		}
	}
	return true
}

func (l *Location) String() string {
	return "s3://" + l.Bucket + "/" + l.Prefix
}

func (l *Location) SameBucket(other *Location) bool {
	return other != nil && l.Bucket == other.Bucket &&
		strings.EqualFold(endpointHost(l.Endpoint), endpointHost(other.Endpoint))
}

func endpointHost(endpoint string) string {
	host := strings.TrimSpace(endpoint)
	if _, rest, found := strings.Cut(host, "://"); found {
		host = rest
	}
	return strings.TrimSuffix(strings.TrimSuffix(host, "/"), ".")
}

func (l *Location) Key(name string) string {
	return l.Prefix + name
}

type Credentials struct {
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	SessionToken string `json:"sessionToken,omitempty"`
}

func (c *Credentials) Check() error {
	if strings.TrimSpace(c.AccessKey) == "" || strings.TrimSpace(c.SecretKey) == "" {
		return ErrCredentialsMissing
	}
	return nil
}

type Key struct {
	ID       string `json:"id"`
	Material []byte `json:"material"`
	Current  bool   `json:"current"`
}

type Access struct {
	Location    Location    `json:"location"`
	Credentials Credentials `json:"credentials"`
	Keys        []Key       `json:"keys,omitempty"`
}

func (a *Access) checkKeys() error {
	if len(a.Keys) > maxArchiveKeys {
		return ErrKeyInvalid
	}
	current := 0
	for i := range a.Keys {
		k := &a.Keys[i]
		if !keyIDPattern.MatchString(k.ID) || len(k.Material) != KeySize {
			return ErrKeyInvalid
		}
		if k.Current {
			current++
		}
	}
	if len(a.Keys) > 0 && current != 1 {
		return ErrKeyInvalid
	}
	return nil
}

func (a *Access) CheckWrite(instanceID string, purpose Purpose) error {
	if err := a.checkPlace(instanceID); err != nil {
		return err
	}
	if err := a.checkKeys(); err != nil {
		return err
	}
	switch {
	case purpose.Encrypted() && len(a.Keys) == 0:
		return ErrKeyInvalid
	case !purpose.Encrypted() && len(a.Keys) > 0:
		return ErrKeyNotExpected
	default:
		return nil
	}
}

func (a *Access) CheckRead(instanceID string) error {
	return a.CheckReadUnder(TenantPrefix(instanceID))
}

func (a *Access) CheckReadUnder(root string) error {
	if err := a.Location.CheckUnder(root); err != nil {
		return err
	}
	if err := a.Credentials.Check(); err != nil {
		return err
	}
	return a.checkKeys()
}

func (a *Access) checkPlace(instanceID string) error {
	if err := a.Location.Check(instanceID); err != nil {
		return err
	}
	return a.Credentials.Check()
}

func (a *Access) CurrentKey() *Key {
	for i := range a.Keys {
		if a.Keys[i].Current {
			return &a.Keys[i]
		}
	}
	return nil
}

func (a *Access) KeyByID(id string) (*Key, error) {
	for i := range a.Keys {
		if a.Keys[i].ID == id {
			return &a.Keys[i], nil
		}
	}
	return nil, ErrKeyUnknown
}

func TenantPrefix(instanceID string) string {
	return TenantsRoot + instanceID + "/"
}

func ImportPrefix(importID string) string {
	return "imports/" + importID + "/"
}

func CommandPrefix(instanceID, archiveID string) string {
	return TenantPrefix(instanceID) + "archives/" + archiveID + "/"
}

func NightlyRoot(instanceID string) string {
	return TenantPrefix(instanceID) + "nightly/"
}

func NightlyPrefix(instanceID string, day time.Time) string {
	return NightlyUnder(TenantsRoot, instanceID, day)
}

func NightlyUnder(root, instanceID string, day time.Time) string {
	return root + instanceID + "/nightly/" + day.UTC().Format(time.DateOnly) + "/"
}

func (a *Access) CheckNightlyRoot() error {
	if a.Location.Prefix != TenantsRoot {
		return ErrNightlyPrefix
	}
	return nil
}

func NightlyDay(prefix, instanceID string) (time.Time, bool) {
	rest, ok := strings.CutPrefix(prefix, NightlyRoot(instanceID))
	if !ok {
		return time.Time{}, false
	}
	day, err := time.Parse(time.DateOnly, strings.TrimSuffix(rest, "/"))
	if err != nil || strings.Count(rest, "/") != 1 {
		return time.Time{}, false
	}
	return day, true
}

func ValidCommandID(raw string) bool {
	parsed, err := uuid.Parse(raw)
	return err == nil && parsed.String() == raw
}

func TablePart(position int, table string) string {
	return fmt.Sprintf("%s%04d-%s.copy.zst", TablesDir, position, table)
}

func DocumentPart(objectKey string) string {
	return DocumentsDir + objectKey
}

func ObjectKey(part string) (string, bool) {
	return strings.CutPrefix(part, DocumentsDir)
}

func QualifiedTable(name string) string {
	return "public." + name
}

func BareTable(qualified string) (string, bool) {
	return strings.CutPrefix(qualified, "public.")
}

func boundedText(s string, limit int) bool {
	return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= limit
}
