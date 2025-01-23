package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type SensitiveFieldAction int

const (
	SensitiveFieldOmit SensitiveFieldAction = iota
	SensitiveFieldMask
	SensitiveFieldHash
)

// SensitiveField is a field that is considered sensitive and should be masked.
type SensitiveField struct {
	Name    string
	Action  SensitiveFieldAction
	Pattern string // Optional regex pattern for more precise masking
}

// SensitiveDataManager is a manager for sensitive data.
type SensitiveDataManager struct {
	resourceFields map[permission.Resource][]SensitiveField
	mu             sync.RWMutex
}

func NewSensitiveDataManager() *SensitiveDataManager {
	return &SensitiveDataManager{
		resourceFields: make(map[permission.Resource][]SensitiveField),
	}
}

// RegisterSensitiveFields registers sensitive fields for a resource.
func (sdm *SensitiveDataManager) RegisterSensitiveFields(resource permission.Resource, fields []SensitiveField) {
	sdm.mu.Lock()
	defer sdm.mu.Unlock()
	sdm.resourceFields[resource] = fields
}

// GetSensitiveFields returns the sensitive fields for a resource.
func (sdm *SensitiveDataManager) GetSensitiveFields(resource permission.Resource) []SensitiveField {
	sdm.mu.RLock()
	defer sdm.mu.RUnlock()
	return sdm.resourceFields[resource]
}

// sanitizeData sanitizes the data in an audit entry.
func sanitizeData(entry *audit.Entry, fields []SensitiveField) {
	// Sanitize Changes
	if entry.Changes != nil {
		sanitizeJSONMap(entry.Changes, fields)
	}

	// Sanitize States
	if entry.PreviousState != nil {
		sanitizeJSONMap(entry.PreviousState, fields)
	}
	if entry.CurrentState != nil {
		sanitizeJSONMap(entry.CurrentState, fields)
	}

	// Sanitize Metadata
	if entry.Metadata != nil {
		sanitizeJSONMap(entry.Metadata, fields)
	}
}

// sanitizeJSONMap sanitizes the data in a JSON map.
func sanitizeJSONMap(data map[string]any, fields []SensitiveField) {
	for _, field := range fields {
		if value, exists := data[field.Name]; exists {
			switch field.Action {
			case SensitiveFieldOmit:
				delete(data, field.Name)
			case SensitiveFieldMask:
				data[field.Name] = maskValue(value)
			case SensitiveFieldHash:
				hashed, err := hashValue(value)
				if err != nil {
					continue
				}
				data[field.Name] = hashed
			}
		}

		// Handle nested objects
		for _, val := range data {
			switch v := val.(type) {
			case map[string]any:
				sanitizeJSONMap(v, fields)
			case []any:
				for _, item := range v {
					if nm, nmOK := item.(map[string]any); nmOK {
						sanitizeJSONMap(nm, fields)
					}
				}
			}
		}
	}
}

// maskValue masks the value of a sensitive field.
func maskValue(value any) any {
	switch v := value.(type) {
	case string:
		// * Handle special cases first
		if maskedValue, handled := handleSpecialCases(v); handled {
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
func hashValue(value any) (string, error) {
	hash := sha256.New()
	_, err := fmt.Fprintf(hash, "%v", value)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// handleSpecialCases handles special cases for sensitive fields.
func handleSpecialCases(value string) (string, bool) {
	if is.URL.Validate(value) == nil {
		return strings.Split(value, "://")[0] + "://" + DefaultMaskValue, true
	}

	if is.Email.Validate(value) == nil {
		parts := strings.Split(value, "@")
		domain := parts[1]
		return "****@" + domain, true
	}

	return "", false
}
