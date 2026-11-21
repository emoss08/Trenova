package tenantarchive

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"

	"github.com/bytedance/sonic"
)

const (
	MaxManifestBytes      = 64 << 20
	maxManifestTables     = 2_000
	maxManifestDocuments  = 400_000
	maxManifestOmissions  = 500
	maxDocumentKey        = 1024
	maxContentType        = 255
	maxTableKey           = 300
	maxTableName          = 128
	maxOrganizationName   = 200
	maxVersionLength      = 64
	maxOmissionReason     = 200
	maxOmissionIdentifier = 128
)

var (
	sha256Pattern      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	contentTypePattern = regexp.MustCompile(
		`^[A-Za-z0-9][A-Za-z0-9!#$&^_.+-]{0,126}/[A-Za-z0-9][A-Za-z0-9!#$&^_.+-]{0,126}` +
			`(;[ \t]*[A-Za-z0-9!#$&^_.+-]+=("[^"\x00-\x1f]*"|[A-Za-z0-9!#$&^_.+-]+))*$`,
	)
	namePattern = regexp.MustCompile(`^public\.[a-z_][a-z0-9_]{0,62}$`)
)

type Encryption struct {
	Algorithm      string `json:"algorithm"`
	SegmentBytes   int    `json:"segmentBytes"`
	KeyID          string `json:"keyId"`
	WrappedDataKey string `json:"wrappedDataKey"`
}

type Table struct {
	Name      string `json:"name"`
	Rows      int64  `json:"rows"`
	Key       string `json:"key"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
	Encoding  string `json:"encoding"`
}

func (t *Table) Bare() string {
	name, _ := BareTable(t.Name)
	return name
}

type Document struct {
	Key         string `json:"key"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256"`
	ContentType string `json:"contentType,omitempty"`
}

func (d *Document) StoredContentType() string {
	if d.ContentType == "" {
		return DefaultContentType
	}
	return d.ContentType
}

type Omission struct {
	Table  string `json:"table"`
	Column string `json:"column,omitempty"`
	Reason string `json:"reason"`
}

type Manifest struct {
	FormatVersion    int         `json:"formatVersion"`
	SchemaVersion    string      `json:"schemaVersion"`
	ReleaseVersion   string      `json:"releaseVersion"`
	InstanceID       string      `json:"instanceId"`
	OrganizationName string      `json:"organizationName"`
	TakenAt          time.Time   `json:"takenAt"`
	Tables           []Table     `json:"tables"`
	Documents        []Document  `json:"documents"`
	TotalSizeBytes   int64       `json:"totalSizeBytes"`
	ContentSHA256    string      `json:"contentSha256"`
	Purpose          Purpose     `json:"purpose,omitempty"`
	Encryption       *Encryption `json:"encryption,omitempty"`
	Omitted          []Omission  `json:"omitted,omitempty"`
}

func (m *Manifest) Encode() ([]byte, error) {
	body, err := sonic.ConfigStd.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode the archive manifest: %w", err)
	}
	return body, nil
}

func DecodeManifest(body []byte) (*Manifest, error) {
	m := new(Manifest)
	if err := sonic.ConfigStd.Unmarshal(body, m); err != nil {
		return nil, Refuse(RefusalManifest, "manifest.json is not an archive manifest")
	}
	return m, nil
}

func (m *Manifest) Seal() {
	m.TotalSizeBytes = 0
	for i := range m.Tables {
		m.TotalSizeBytes += m.Tables[i].SizeBytes
	}
	for i := range m.Documents {
		m.TotalSizeBytes += m.Documents[i].SizeBytes
	}
	m.ContentSHA256 = m.ContentDigest()
}

func (m *Manifest) ContentDigest() string {
	h := sha256.New()
	for i := range m.Tables {
		h.Write([]byte(m.Tables[i].SHA256 + "  " + m.Tables[i].Key + "\n"))
	}
	for i := range m.Documents {
		h.Write([]byte(m.Documents[i].SHA256 + "  " + m.Documents[i].Key + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

type problems []string

func (p *problems) addf(format string, args ...any) {
	*p = append(*p, fmt.Sprintf(format, args...))
}

func (m *Manifest) Check() error {
	found := new(problems)
	m.checkHeader(found)
	m.checkEncryption(found)
	m.checkTables(found)
	m.checkDocuments(found)
	m.checkTotals(found)
	if len(*found) == 0 {
		return nil
	}
	return Refuse(RefusalManifest, "the manifest is not valid: "+strings.Join(*found, "; "))
}

func (m *Manifest) checkHeader(p *problems) {
	if m.FormatVersion != FormatVersion {
		p.addf("format version %d is not %d", m.FormatVersion, FormatVersion)
	}
	if !ValidCommandID(m.InstanceID) || !m.Purpose.Valid() {
		p.addf("the instance id and purpose are required")
	}
	if !boundedText(m.SchemaVersion, maxVersionLength) ||
		len(m.ReleaseVersion) > maxVersionLength ||
		!boundedText(m.OrganizationName, maxOrganizationName) ||
		m.TakenAt.IsZero() {
		p.addf("the schema version, organization name and when it was taken are required")
	}
	if len(m.Omitted) > maxManifestOmissions || (len(m.Omitted) > 0 && m.Purpose != PurposeExport) {
		p.addf("only an export lists what it left out")
	}
	for _, o := range m.Omitted {
		if len(o.Table) > maxOmissionIdentifier || len(o.Column) > maxOmissionIdentifier ||
			!boundedText(o.Reason, maxOmissionReason) {
			p.addf("an omission is malformed")
			return
		}
	}
}

func (m *Manifest) checkEncryption(p *problems) {
	if m.Purpose.Encrypted() != (m.Encryption != nil) {
		p.addf("every archive but an export is encrypted")
	}
	if m.Encryption != nil && (m.Encryption.Algorithm != EncryptionAlgorithm ||
		m.Encryption.SegmentBytes != SegmentBytes || !keyIDPattern.MatchString(m.Encryption.KeyID) ||
		m.Encryption.WrappedDataKey == "") {
		p.addf(
			"the encryption block must be %s with %d-byte segments, a key id and the wrapped data key",
			EncryptionAlgorithm,
			SegmentBytes,
		)
	}
}

func (m *Manifest) checkTables(p *problems) {
	if len(m.Tables) == 0 || len(m.Tables) > maxManifestTables ||
		len(m.Documents) > maxManifestDocuments {
		p.addf("the manifest lists too few or too many parts")
		return
	}
	keys := make(map[string]struct{}, len(m.Tables))
	for i := range m.Tables {
		t := &m.Tables[i]
		if !t.wellFormed(i + 1) {
			p.addf("table %d (%s) is malformed", i+1, t.Name)
			continue
		}
		if _, dup := keys[t.Key]; dup {
			p.addf("table %s is listed twice", t.Name)
		}
		keys[t.Key] = struct{}{}
	}
}

func (t *Table) wellFormed(position int) bool {
	bare, ok := BareTable(t.Name)
	return ok && namePattern.MatchString(t.Name) && len(t.Name) <= maxTableName && t.Rows >= 0 &&
		t.SizeBytes >= 0 && sha256Pattern.MatchString(t.SHA256) && t.Key == TablePart(position, bare) &&
		len(t.Key) <= maxTableKey && t.Encoding == TableEncoding
}

func (m *Manifest) checkDocuments(p *problems) {
	keys := make(map[string]struct{}, len(m.Documents))
	for i := range m.Documents {
		d := &m.Documents[i]
		if !d.wellFormed() {
			p.addf("document %d is malformed", i+1)
			return
		}
		if _, dup := keys[d.Key]; dup {
			p.addf("document %s is listed twice", d.Key)
			return
		}
		keys[d.Key] = struct{}{}
	}
}

func (d *Document) wellFormed() bool {
	object, ok := ObjectKey(d.Key)
	return ok && objectkey.Clean(object) && d.SizeBytes >= 0 && sha256Pattern.MatchString(d.SHA256) &&
		len(d.Key) <= maxDocumentKey && !strings.Contains(d.Key, "..") &&
		(d.ContentType == "" ||
			(len(d.ContentType) <= maxContentType && contentTypePattern.MatchString(d.ContentType))) &&
		!strings.HasPrefix(object, "/")
}

func (m *Manifest) checkTotals(p *problems) {
	if m.ContentSHA256 != m.ContentDigest() {
		p.addf("the content digest does not match the manifest's parts")
	}
	var total int64
	for i := range m.Tables {
		total += m.Tables[i].SizeBytes
	}
	for i := range m.Documents {
		total += m.Documents[i].SizeBytes
	}
	if total != m.TotalSizeBytes {
		p.addf("totalSizeBytes is not the parts' sizes together")
	}
}

func (m *Manifest) Table(name string) (*Table, bool) {
	for i := range m.Tables {
		if m.Tables[i].Name == name {
			return &m.Tables[i], true
		}
	}
	return nil, false
}

func ManifestSHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func ValidSHA256(s string) bool {
	return sha256Pattern.MatchString(s)
}
