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

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
)

// SensitiveDataManager is a manager for sensitive data.
type SensitiveDataManager struct {
	resourceFields map[permission.Resource][]services.SensitiveField
	encryptionKey  []byte
	regexCache     map[string]*regexp.Regexp
	mu             sync.RWMutex
}

func NewSensitiveDataManager() *SensitiveDataManager {
	return &SensitiveDataManager{
		resourceFields: make(map[permission.Resource][]services.SensitiveField),
		regexCache:     make(map[string]*regexp.Regexp),
	}
}

// SetEncryptionKey sets the encryption key for field-level encryption
// This should be called during initialization with a secure key
func (sdm *SensitiveDataManager) SetEncryptionKey(key []byte) error {
	if len(key) != 32 { // AES-256 requires 32 bytes
		return eris.New("encryption key must be 32 bytes for AES-256")
	}

	sdm.mu.Lock()
	defer sdm.mu.Unlock()

	sdm.encryptionKey = make([]byte, len(key))
	copy(sdm.encryptionKey, key)

	return nil
}

// RegisterSensitiveFields registers sensitive fields for a resource.
func (sdm *SensitiveDataManager) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	sdm.mu.Lock()
	defer sdm.mu.Unlock()

	// Precompile any regex patterns for efficiency
	for i, field := range fields {
		if field.Pattern != "" {
			if _, exists := sdm.regexCache[field.Pattern]; !exists {
				compiled, err := regexp.Compile(field.Pattern)
				if err != nil {
					return eris.Wrapf(err, "invalid regex pattern for field %s", field.Name)
				}
				sdm.regexCache[field.Pattern] = compiled
			}
		}

		// Validate that we have an encryption key if any field uses encryption
		if field.Action == services.SensitiveFieldEncrypt && len(sdm.encryptionKey) == 0 {
			return eris.New("encryption key not set for encrypted fields")
		}

		// Store the validated field
		fields[i] = field
	}

	sdm.resourceFields[resource] = fields
	return nil
}

// GetSensitiveFields returns the sensitive fields for a resource.
func (sdm *SensitiveDataManager) GetSensitiveFields(
	resource permission.Resource,
) []services.SensitiveField {
	sdm.mu.RLock()
	defer sdm.mu.RUnlock()

	fields, ok := sdm.resourceFields[resource]
	if !ok {
		return nil
	}

	// Return a copy to prevent modification of internal state
	result := make([]services.SensitiveField, len(fields))
	copy(result, fields)

	return result
}

// sanitizeData sanitizes the data in an audit entry.
func (sdm *SensitiveDataManager) sanitizeData(entry *audit.Entry) error {
	fields := sdm.GetSensitiveFields(entry.Resource)
	if len(fields) == 0 {
		return nil
	}

	// Sanitize Changes
	if entry.Changes != nil {
		if err := sdm.sanitizeJSONMap(entry.Changes, fields); err != nil {
			return err
		}
	}

	// Sanitize States
	if entry.PreviousState != nil {
		if err := sdm.sanitizeJSONMap(entry.PreviousState, fields); err != nil {
			return err
		}
	}
	if entry.CurrentState != nil {
		if err := sdm.sanitizeJSONMap(entry.CurrentState, fields); err != nil {
			return err
		}
	}

	// Sanitize Metadata
	if entry.Metadata != nil {
		if err := sdm.sanitizeJSONMap(entry.Metadata, fields); err != nil {
			return err
		}
	}

	entry.SensitiveData = true
	return nil
}

// sanitizeJSONMap sanitizes the data in a JSON map.
func (sdm *SensitiveDataManager) sanitizeJSONMap(
	data map[string]any,
	fields []services.SensitiveField,
) error {
	// Apply direct and path-based sanitization
	if err := sdm.applyNameAndPathRules(data, fields); err != nil {
		return err
	}

	// Apply pattern-based sanitization and handle recursion
	return sdm.applyPatternsAndRecurse(data, fields)
}

// applyNameAndPathRules handles the direct field matches and path-based rules (Phase 1)
func (sdm *SensitiveDataManager) applyNameAndPathRules(
	data map[string]any,
	fields []services.SensitiveField,
) error {
	for _, fieldRule := range fields {
		// Skip purely pattern-based rules
		if fieldRule.Pattern != "" && fieldRule.Name == "" && fieldRule.Path == "" {
			continue
		}

		targetMap := data
		keyName := fieldRule.Name

		// Navigate to target map if path is specified
		if fieldRule.Path != "" {
			found, pathMap := sdm.findMapAtPath(data, fieldRule.Path)
			if !found {
				continue // Path not found
			}
			targetMap = pathMap
		}

		// Apply sanitization if field exists
		if value, exists := targetMap[keyName]; exists {
			if err := sdm.applySanitizationAction(targetMap, keyName, value, fieldRule.Action); err != nil {
				return eris.Wrapf(
					err,
					"failed to apply sanitization to %s.%s",
					fieldRule.Path,
					keyName,
				)
			}
		}
	}
	return nil
}

// findMapAtPath navigates to the map at the specified dot-separated path
func (sdm *SensitiveDataManager) findMapAtPath(
	data map[string]any,
	path string,
) (found bool, nestedMap map[string]any) {
	currentMap := data

	for segment := range strings.SplitSeq(path, ".") {
		nested, ok := currentMap[segment].(map[string]any)
		if !ok {
			return false, nil
		}
		currentMap = nested
	}

	return true, currentMap
}

// applyPatternsAndRecurse handles pattern matching and recursion (Phase 2)
func (sdm *SensitiveDataManager) applyPatternsAndRecurse(
	data map[string]any,
	fields []services.SensitiveField,
) error {
	for key, val := range data {
		// Apply pattern-based rules
		if err := sdm.applyPatternRules(data, key, val, fields); err != nil {
			return err
		}

		// Recurse into nested structures
		if err := sdm.recurseIntoNestedStructures(val, fields); err != nil {
			return err
		}
	}
	return nil
}

// applyPatternRules applies pattern-based rules to a specific value
func (sdm *SensitiveDataManager) applyPatternRules(
	data map[string]any,
	key string,
	val any,
	fields []services.SensitiveField,
) error {
	strVal, ok := val.(string)
	if !ok {
		return nil // Only strings can match patterns
	}

	for _, fieldRule := range fields {
		if fieldRule.Pattern == "" {
			continue // Skip non-pattern rules
		}

		sdm.mu.RLock()
		pattern, exists := sdm.regexCache[fieldRule.Pattern]
		sdm.mu.RUnlock()

		if exists && pattern.MatchString(strVal) {
			if err := sdm.applySanitizationAction(data, key, val, fieldRule.Action); err != nil {
				return eris.Wrapf(err, "failed to apply pattern sanitization to key %s", key)
			}
		}
	}
	return nil
}

// recurseIntoNestedStructures recursively processes nested maps and arrays
func (sdm *SensitiveDataManager) recurseIntoNestedStructures(
	val any,
	fields []services.SensitiveField,
) error {
	switch v := val.(type) {
	case map[string]any:
		if err := sdm.sanitizeJSONMap(v, fields); err != nil {
			return err
		}
	case []any:
		if err := sdm.sanitizeJSONArray(v, fields); err != nil {
			return err
		}
	}
	return nil
}

// applySanitizationAction applies the specified sanitization action to a field
func (sdm *SensitiveDataManager) applySanitizationAction(
	data map[string]any,
	key string,
	value any,
	action services.SensitiveFieldAction,
) error {
	switch action {
	case services.SensitiveFieldOmit:
		delete(data, key)
	case services.SensitiveFieldMask:
		data[key] = sdm.maskValue(value)
	case services.SensitiveFieldHash:
		hashed, err := sdm.hashValue(value)
		if err != nil {
			return eris.Wrapf(err, "failed to hash field %s", key)
		}
		data[key] = hashed
	case services.SensitiveFieldEncrypt:
		encrypted, err := sdm.encryptValue(value)
		if err != nil {
			return eris.Wrapf(err, "failed to encrypt field %s", key)
		}
		data[key] = encrypted
	}
	return nil
}

// sanitizeJSONArray sanitizes elements in a JSON array
func (sdm *SensitiveDataManager) sanitizeJSONArray(
	arr []any,
	fields []services.SensitiveField,
) error {
	for _, item := range arr {
		if mapItem, ok := item.(map[string]any); ok {
			if err := sdm.sanitizeJSONMap(mapItem, fields); err != nil {
				return err
			}
		}
	}
	return nil
}

// maskValue masks the value of a sensitive field.
func (sdm *SensitiveDataManager) maskValue(value any) any {
	switch v := value.(type) {
	case string:
		// * Handle special cases first
		if maskedValue, handled := sdm.handleSpecialCases(v); handled {
			return maskedValue
		}

		// Default string masking for non-special cases
		if len(v) <= 4 {
			return DefaultMaskValue
		}
		visible := len(v) / 4
		return v[:visible] + strings.Repeat("*", len(v)-visible)

	case int, int32, int64, float32, float64:
		return DefaultMaskValue

	default:
		return DefaultMaskValue
	}
}

// hashValue hashes the value of a sensitive field.
func (sdm *SensitiveDataManager) hashValue(value any) (string, error) {
	hash := sha256.New()
	_, err := fmt.Fprintf(hash, "%v", value)
	if err != nil {
		return "", eris.Wrap(err, "failed to compute hash")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// encryptValue encrypts a value using AES-GCM
func (sdm *SensitiveDataManager) encryptValue(value any) (string, error) {
	sdm.mu.RLock()
	key := sdm.encryptionKey
	sdm.mu.RUnlock()

	if len(key) == 0 {
		return "", eris.New("encryption key not set")
	}

	// Convert value to string
	plaintext := fmt.Sprintf("%v", value)

	// Create a new AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", eris.Wrap(err, "failed to create AES cipher")
	}

	// Create a GCM mode cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", eris.Wrap(err, "failed to create GCM cipher")
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", eris.Wrap(err, "failed to create nonce")
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64 encoded ciphertext
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// handleSpecialCases handles special cases for sensitive fields.
func (sdm *SensitiveDataManager) handleSpecialCases(value string) (string, bool) {
	// Handle URLs
	if is.URL.Validate(value) == nil {
		parts := strings.Split(value, "://")
		if len(parts) >= 2 {
			return parts[0] + "://" + DefaultMaskValue, true
		}
	}

	// Handle emails
	if is.Email.Validate(value) == nil {
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			domain := parts[1]
			return "****@" + domain, true
		}
	}

	// Handle SSN (assuming US format XXX-XX-XXXX)
	ssnPattern := regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)
	if ssnPattern.MatchString(value) {
		return "XXX-XX-" + value[7:], true
	}

	// Handle credit card numbers
	ccPattern := regexp.MustCompile(`^\d{13,19}$`)
	if ccPattern.MatchString(value) {
		last4 := value[len(value)-4:]
		return strings.Repeat("*", len(value)-4) + last4, true
	}

	return "", false
}
