// Package audit provides sensitive data management for audit logging.
//
// The SensitiveDataManagerV2 provides automatic detection and sanitization of sensitive data
// in audit logs. It supports multiple masking strategies and can be configured based on
// your environment and security requirements.
//
// # When to Use Auto-Detection (Default: ON)
//
// Auto-detection automatically identifies and masks common sensitive patterns like:
// - Social Security Numbers (SSN)
// - Credit card numbers
// - API keys and tokens
// - JWT tokens
// - Private keys
// - Common field names (password, secret, token, etc.)
//
// Enable auto-detection when:
// - Running in production (prevents accidental data exposure)
// - You want an extra layer of security
// - Working with third-party data where fields might not be known
// - During development to catch sensitive data early
//
// Disable auto-detection when:
// - Running tests that need predictable output
// - Performance is critical and all fields are explicitly configured
// - During data migration where original values must be preserved
//
// # Masking Strategies
//
// MaskStrategyStrict (Production):
// - Shows minimal information
// - Example: "test@example.com" → "****@example.com"
// - Example: "4111111111111111" → "************1111"
//
// MaskStrategyDefault (Staging):
// - Balanced between security and debugging
// - Example: "test@example.com" → "t***@example.com"
// - Example: "4111111111111111" → "************1111"
//
// MaskStrategyPartial (Development):
// - Shows more information for easier debugging
// - Example: "test@example.com" → "te***@example.com"
// - Example: "mysecrettoken123" → "my**********123"
//
// # Configuration Examples
//
// The service automatically configures based on environment, but you can override:
//
//	// For debugging a specific issue in production
//	auditService.SetSensitiveDataMaskStrategy(MaskStrategyPartial)
//	defer auditService.SetSensitiveDataMaskStrategy(MaskStrategyStrict)
//
//	// During data migration
//	auditService.SetSensitiveDataAutoDetect(false)
//	// ... perform migration ...
//	auditService.SetSensitiveDataAutoDetect(true)
package audit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
)

// SensitivePatterns defines common patterns for sensitive data detection
var SensitivePatterns = struct {
	SSN        *regexp.Regexp
	CreditCard *regexp.Regexp
	APIKey     *regexp.Regexp
	JWT        *regexp.Regexp
	PrivateKey *regexp.Regexp
}{
	SSN:        regexp.MustCompile(`^\d{3}-?\d{2}-?\d{4}$`),
	CreditCard: regexp.MustCompile(`^\d{13,19}$`),
	APIKey:     regexp.MustCompile(`^[A-Za-z0-9_\-]{20,}$`),
	JWT:        regexp.MustCompile(`^[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+$`),
	PrivateKey: regexp.MustCompile(`-{5}BEGIN.*PRIVATE KEY-{5}`),
}

// SensitiveDataManagerV2 is an improved sensitive data manager
type SensitiveDataManagerV2 struct {
	// Field configurations per resource
	resourceFields sync.Map // map[permission.Resource][]services.SensitiveField

	// Encryption key for field-level encryption
	encryptionKey atomic.Pointer[[]byte]

	// Regex cache for performance
	regexCache sync.Map // map[string]*regexp.Regexp

	// Performance metrics
	metrics struct {
		sanitizedFields atomic.Int64
		encryptedFields atomic.Int64
		hashedFields    atomic.Int64
		maskedFields    atomic.Int64
		errors          atomic.Int64
	}

	// Configuration
	config struct {
		autoDetectSensitive bool
		maskStrategy        MaskStrategy
	}
}

// MaskStrategy defines how values should be masked
type MaskStrategy int

const (
	MaskStrategyDefault MaskStrategy = iota
	MaskStrategyStrict               // * Show less information
	MaskStrategyPartial              // * Show more information for debugging
)

// NewSensitiveDataManagerV2 creates a new improved sensitive data manager
func NewSensitiveDataManagerV2() *SensitiveDataManagerV2 {
	return &SensitiveDataManagerV2{
		config: struct {
			autoDetectSensitive bool
			maskStrategy        MaskStrategy
		}{
			autoDetectSensitive: true,
			maskStrategy:        MaskStrategyDefault,
		},
	}
}

// SetEncryptionKey sets the encryption key for field-level encryption
func (m *SensitiveDataManagerV2) SetEncryptionKey(key []byte) error {
	if len(key) != 32 { // * AES-256 requires 32 bytes
		return eris.New("encryption key must be 32 bytes for AES-256")
	}

	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	m.encryptionKey.Store(&keyCopy)

	return nil
}

// RegisterSensitiveFields registers sensitive fields for a resource
func (m *SensitiveDataManagerV2) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	// Precompile regex patterns
	for _, field := range fields {
		if field.Pattern != "" {
			if _, exists := m.regexCache.Load(field.Pattern); !exists {
				compiled, err := regexp.Compile(field.Pattern)
				if err != nil {
					return eris.Wrapf(err, "invalid regex pattern for field %s", field.Name)
				}
				m.regexCache.Store(field.Pattern, compiled)
			}
		}

		// Validate encryption key requirement
		if field.Action == services.SensitiveFieldEncrypt {
			if m.encryptionKey.Load() == nil {
				return eris.New("encryption key not set for encrypted fields")
			}
		}
	}

	m.resourceFields.Store(resource, fields)
	return nil
}

// SanitizeEntry sanitizes sensitive data in an audit entry
func (m *SensitiveDataManagerV2) SanitizeEntry(entry *audit.Entry) error {
	fieldsVal, exists := m.resourceFields.Load(entry.Resource)
	if !exists {
		// * If auto-detection is enabled, scan for common patterns
		if m.config.autoDetectSensitive {
			return m.autoDetectAndSanitize(entry)
		}
		return nil
	}

	fields, _ := fieldsVal.([]services.SensitiveField)

	// * Process all data fields
	for _, data := range []map[string]any{
		entry.Changes,
		entry.PreviousState,
		entry.CurrentState,
		entry.Metadata,
	} {
		if data != nil {
			if err := m.sanitizeMap(data, fields, ""); err != nil {
				m.metrics.errors.Add(1)
				return err
			}
		}
	}

	entry.SensitiveData = true
	return nil
}

// sanitizeMap recursively sanitizes a map
func (m *SensitiveDataManagerV2) sanitizeMap(
	data map[string]any,
	fields []services.SensitiveField,
	currentPath string,
) error {
	for key, value := range data {
		path := key
		if currentPath != "" {
			path = currentPath + "." + key
		}

		// * Check field rules
		for _, field := range fields {
			if m.shouldSanitizeField(key, path, value, field) {
				if err := m.applySanitization(data, key, value, field.Action); err != nil {
					return err
				}
				break // ! Field matched, no need to check other rules
			}
		}

		// Recurse into nested structures
		switch v := value.(type) {
		case map[string]any:
			if err := m.sanitizeMap(v, fields, path); err != nil {
				return err
			}
		case []any:
			if err := m.sanitizeArray(v, fields, path); err != nil {
				return err
			}
		}
	}

	return nil
}

// sanitizeArray sanitizes elements in an array
func (m *SensitiveDataManagerV2) sanitizeArray(
	arr []any,
	fields []services.SensitiveField,
	path string,
) error {
	for i, item := range arr {
		itemPath := fmt.Sprintf("%s[%d]", path, i)

		switch v := item.(type) {
		case map[string]any:
			if err := m.sanitizeMap(v, fields, itemPath); err != nil {
				return err
			}
		case []any:
			if err := m.sanitizeArray(v, fields, itemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// shouldSanitizeField determines if a field should be sanitized
func (m *SensitiveDataManagerV2) shouldSanitizeField(
	key, path string,
	value any,
	field services.SensitiveField,
) bool {
	// Direct name match
	if field.Name == key && field.Path == "" {
		return true
	}

	// Path match
	if field.Path != "" && field.Name != "" {
		expectedPath := field.Path + "." + field.Name
		if path == expectedPath {
			return true
		}
	}

	// Pattern match for string values
	if field.Pattern != "" { //nolint:nestif // this is fine.
		if strVal, ok := value.(string); ok {
			if regex, exists := m.regexCache.Load(field.Pattern); exists {
				if regex.(*regexp.Regexp).MatchString(strVal) { //nolint:errcheck // this is fine.
					return true
				}
			}
		}
	}

	return false
}

// applySanitization applies the specified sanitization action
func (m *SensitiveDataManagerV2) applySanitization(
	data map[string]any,
	key string,
	value any,
	action services.SensitiveFieldAction,
) error {
	switch action {
	case services.SensitiveFieldOmit:
		delete(data, key)
		m.metrics.sanitizedFields.Add(1)

	case services.SensitiveFieldMask:
		data[key] = m.maskValue(value)
		m.metrics.maskedFields.Add(1)

	case services.SensitiveFieldHash:
		hashed, err := m.hashValue(value)
		if err != nil {
			return err
		}
		data[key] = hashed
		m.metrics.hashedFields.Add(1)

	case services.SensitiveFieldEncrypt:
		encrypted, err := m.encryptValue(value)
		if err != nil {
			return err
		}
		data[key] = encrypted
		m.metrics.encryptedFields.Add(1)
	}

	m.metrics.sanitizedFields.Add(1)
	return nil
}

// maskValue masks a value based on the configured strategy
func (m *SensitiveDataManagerV2) maskValue(value any) string {
	switch v := value.(type) {
	case string:
		return m.maskString(v)
	case int, int32, int64, float32, float64:
		return m.maskNumber(v)
	default:
		return DefaultMaskValue
	}
}

// maskString masks a string value with intelligent detection
func (m *SensitiveDataManagerV2) maskString(value string) string {
	if value == "" {
		return ""
	}

	if is.Email.Validate(value) == nil {
		return m.maskEmail(value)
	}

	if is.URL.Validate(value) == nil {
		return m.maskURL(value)
	}

	if masked := m.maskByPattern(value); masked != "" {
		return masked
	}

	return m.maskDefault(value)
}

// maskEmail masks email addresses
func (m *SensitiveDataManagerV2) maskEmail(value string) string {
	parts := strings.Split(value, "@")
	if len(parts) == 2 {
		switch m.config.maskStrategy { //nolint:exhaustive // this is fine.
		case MaskStrategyStrict:
			return "****@" + parts[1]
		case MaskStrategyPartial:
			if len(parts[0]) > 2 {
				return parts[0][:2] + "***@" + parts[1]
			}
			return "***@" + parts[1]
		default:
			if parts[0] != "" {
				return parts[0][:1] + "***@" + parts[1]
			}
			return "****@" + parts[1]
		}
	}
	return value
}

// maskByPattern masks values based on sensitive patterns
func (m *SensitiveDataManagerV2) maskByPattern(value string) string {
	// SSN
	if SensitivePatterns.SSN.MatchString(value) {
		switch m.config.maskStrategy { //nolint:exhaustive // this is fine.
		case MaskStrategyStrict:
			return "XXX-XX-XXXX"
		case MaskStrategyPartial:
			if len(value) >= 4 {
				return "XXX-XX-" + value[len(value)-4:]
			}
		default:
			if len(value) >= 4 {
				return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
			}
		}
	}

	if SensitivePatterns.CreditCard.MatchString(value) {
		if len(value) >= 4 {
			return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
		}
		return strings.Repeat("*", len(value))
	}

	if SensitivePatterns.APIKey.MatchString(value) || SensitivePatterns.JWT.MatchString(value) {
		if m.config.maskStrategy == MaskStrategyPartial && len(value) > 8 {
			return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
		}
		return strings.Repeat("*", len(value))
	}

	if SensitivePatterns.PrivateKey.MatchString(value) {
		return "-----BEGIN PRIVATE KEY----- [REDACTED]"
	}

	return ""
}

// maskDefault applies default masking strategy
func (m *SensitiveDataManagerV2) maskDefault(value string) string {
	length := len(value)
	switch m.config.maskStrategy { //nolint:exhaustive // this is fine.
	case MaskStrategyStrict:
		return strings.Repeat("*", length)
	case MaskStrategyPartial:
		if length <= 4 {
			return strings.Repeat("*", length)
		}
		if length <= 8 {
			return value[:1] + strings.Repeat("*", length-2) + value[length-1:]
		}
		// * Show first 2 and last 2 chars
		return value[:2] + strings.Repeat("*", length-4) + value[length-2:]
	default:
		if length <= 4 {
			return DefaultMaskValue
		}
		// * Show first and last character
		return value[:1] + strings.Repeat("*", length-2) + value[length-1:]
	}
}

// maskURL masks URL values
func (m *SensitiveDataManagerV2) maskURL(value string) string {
	if idx := strings.Index(value, "://"); idx != -1 { //nolint:nestif // this is fine.
		protocol := value[:idx+3]
		rest := value[idx+3:]

		// * Check for credentials in URL
		if atIdx := strings.Index(rest, "@"); atIdx != -1 {
			// * Has credentials - mask them completely
			afterAt := rest[atIdx:]
			return protocol + "****:****" + afterAt
		}

		// * No credentials, mask the domain partially
		if m.config.maskStrategy == MaskStrategyStrict {
			return protocol + "****"
		}

		// * Show partial domain
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			domain := rest[:slashIdx]
			path := rest[slashIdx:]
			if len(domain) > 4 {
				return protocol + domain[:4] + "****" + path
			}
		}
		return protocol + "****"
	}
	return value
}

// maskNumber masks numeric values
func (m *SensitiveDataManagerV2) maskNumber(value any) string {
	str := fmt.Sprintf("%v", value)

	switch m.config.maskStrategy { //nolint:exhaustive // this is fine.
	case MaskStrategyStrict:
		return strings.Repeat("*", len(str))
	case MaskStrategyPartial:
		if len(str) > 4 {
			return strings.Repeat("*", len(str)-2) + str[len(str)-2:]
		}
		return strings.Repeat("*", len(str))
	default:
		return DefaultMaskValue
	}
}

// hashValue creates a secure hash of the value
func (m *SensitiveDataManagerV2) hashValue(value any) (string, error) {
	// * Use SHA-256 for consistent hashing
	hash := sha256.New()
	_, err := fmt.Fprintf(hash, "%v", value)
	if err != nil {
		return "", eris.Wrap(err, "failed to compute hash")
	}

	// * Return hex-encoded hash with prefix for identification
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))[:16], nil
}

// encryptValue encrypts a value using AES-GCM
func (m *SensitiveDataManagerV2) encryptValue(value any) (string, error) {
	keyPtr := m.encryptionKey.Load()
	if keyPtr == nil || len(*keyPtr) == 0 {
		return "", eris.New("encryption key not set")
	}
	key := *keyPtr

	// * Convert value to bytes
	plaintext := []byte(fmt.Sprintf("%v", value))

	// * Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", eris.Wrap(err, "failed to create cipher")
	}

	// * Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", eris.Wrap(err, "failed to create GCM")
	}

	// * Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", eris.Wrap(err, "failed to generate nonce")
	}

	// * Encrypt
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// * Return base64-encoded with prefix
	return "enc:gcm:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// autoDetectAndSanitize automatically detects and sanitizes common sensitive patterns
func (m *SensitiveDataManagerV2) autoDetectAndSanitize(entry *audit.Entry) error {
	autoRules := []services.SensitiveField{
		{Pattern: SensitivePatterns.SSN.String(), Action: services.SensitiveFieldMask},
		{Pattern: SensitivePatterns.CreditCard.String(), Action: services.SensitiveFieldMask},
		{Pattern: SensitivePatterns.APIKey.String(), Action: services.SensitiveFieldMask},
		{Pattern: SensitivePatterns.JWT.String(), Action: services.SensitiveFieldMask},
		{Pattern: SensitivePatterns.PrivateKey.String(), Action: services.SensitiveFieldOmit},
		{Name: "password", Action: services.SensitiveFieldOmit},
		{Name: "secret", Action: services.SensitiveFieldOmit},
		{Name: "token", Action: services.SensitiveFieldMask},
		{Name: "apiKey", Action: services.SensitiveFieldMask},
		{Name: "api_key", Action: services.SensitiveFieldMask},
		{Name: "privateKey", Action: services.SensitiveFieldOmit},
		{Name: "private_key", Action: services.SensitiveFieldOmit},
	}

	anySanitized := false
	for _, data := range []map[string]any{
		entry.Changes,
		entry.PreviousState,
		entry.CurrentState,
		entry.Metadata,
	} {
		if data != nil {
			if err := m.sanitizeMap(data, autoRules, ""); err != nil {
				return err
			}
			anySanitized = true
		}
	}

	if anySanitized {
		entry.SensitiveData = true
	}

	return nil
}

// GetMetrics returns sanitization metrics
func (m *SensitiveDataManagerV2) GetMetrics() SanitizationMetrics {
	return SanitizationMetrics{
		SanitizedFields: m.metrics.sanitizedFields.Load(),
		EncryptedFields: m.metrics.encryptedFields.Load(),
		HashedFields:    m.metrics.hashedFields.Load(),
		MaskedFields:    m.metrics.maskedFields.Load(),
		Errors:          m.metrics.errors.Load(),
	}
}

// SetMaskStrategy sets the masking strategy
func (m *SensitiveDataManagerV2) SetMaskStrategy(strategy MaskStrategy) {
	m.config.maskStrategy = strategy
}

// SetAutoDetect enables or disables automatic sensitive data detection
func (m *SensitiveDataManagerV2) SetAutoDetect(enabled bool) {
	m.config.autoDetectSensitive = enabled
}

// ClearCache clears the regex cache
func (m *SensitiveDataManagerV2) ClearCache() {
	m.regexCache.Range(func(key, value any) bool {
		m.regexCache.Delete(key)
		return true
	})
}

// SanitizationMetrics holds metrics about sanitization operations
type SanitizationMetrics struct {
	SanitizedFields int64
	EncryptedFields int64
	HashedFields    int64
	MaskedFields    int64
	Errors          int64
}
