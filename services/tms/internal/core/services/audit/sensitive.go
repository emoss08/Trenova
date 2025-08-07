/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package audit

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"go.uber.org/atomic"
)

// MaskStrategy defines how sensitive data should be masked
type MaskStrategy int

const (
	// MaskStrategyStrict replaces entire value with mask (most secure)
	MaskStrategyStrict MaskStrategy = iota
	// MaskStrategyDefault shows partial information (balanced)
	MaskStrategyDefault
	// MaskStrategyPartial shows more information (least secure)
	MaskStrategyPartial
)

// SensitiveDataManagerV2 is an improved version of the sensitive data manager
// with better performance, more features, and auto-detection capabilities
type SensitiveDataManagerV2 struct {
	// Using sync.Map for better concurrent performance
	fields       sync.Map // map[permission.Resource]map[string]SensitiveFieldConfig
	patternCache sync.Map // map[string]*regexp.Regexp

	// Configuration
	autoDetect atomic.Bool
	strategy   atomic.Int32

	// Pre-compiled patterns for better performance
	compiledFieldPatterns []*regexp.Regexp
	patternMutex          sync.RWMutex
}

// SensitiveFieldConfig holds the configuration for a sensitive field
type SensitiveFieldConfig struct {
	Path   string
	Name   string
	Action services.SensitiveFieldAction
}

// Pre-defined sensitive data patterns for auto-detection
var sensitivePatterns = map[string]string{
	// US Social Security Number
	"ssn": `\b\d{3}-\d{2}-\d{4}\b|\b\d{9}\b`,

	// Credit Card Numbers (basic patterns)
	"creditCard": `\b(?:\d[ -]*?){13,19}\b`,

	// Email addresses
	"email": `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,

	// Phone numbers (US format)
	"phone": `\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`,

	// API Keys (generic patterns - improved)
	"apiKey": `\b(?i)(api[_-]?key|apikey|api[_-]?secret|access[_-]?token|auth[_-]?token|bearer)\s*[:=]\s*["']?[\w\-]{20,}["']?\b|^[A-Za-z0-9]{20,}$`,

	// Google API Keys specifically
	"googleApiKey": `\bAIza[0-9A-Za-z\-_]{35}\b`,

	// AWS Keys
	"awsKey": `\b(?:AKIA|A3T|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[0-9A-Z]{16}\b`,

	// JWT Tokens
	"jwt": `\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`,

	// Database Connection Strings
	"dbConnection": `\b(?i)(mongodb|postgres|postgresql|mysql|redis|mssql|oracle):\/\/[^\s]+\b`,

	// Private Keys (generic pattern)
	"privateKey": `-----BEGIN\s+(?:RSA\s+)?PRIVATE\s+KEY-----[\s\S]+?-----END\s+(?:RSA\s+)?PRIVATE\s+KEY-----`,

	// IP Addresses (IPv4)
	"ipAddress": `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,

	// Bank Account Numbers (US routing + account)
	"bankAccount": `\b\d{9}\s*\d{1,17}\b`,

	// Driver's License (generic pattern - numbers and letters)
	"driversLicense": `\b(?i)(license|licence|dl|driver)\s*(?:number|no|#)?\s*[:=]?\s*[A-Z0-9]{5,20}\b`,

	// Date of Birth (various formats)
	"dateOfBirth": `\b(?:0[1-9]|1[0-2])[-/](?:0[1-9]|[12][0-9]|3[01])[-/](?:19|20)\d{2}\b`,

	// Tax ID / EIN
	"taxId": `\b\d{2}-\d{7}\b`,

	// Passport Number (alphanumeric)
	"passport": `\b(?i)passport\s*(?:number|no|#)?\s*[:=]?\s*[A-Z0-9]{6,20}\b`,
}

// Field name patterns that might contain sensitive data
var sensitiveFieldPatterns = []string{
	`(?i)password`,
	`(?i)secret`,
	`(?i)token`,
	`(?i)api[_-]?key`,
	`(?i)apikey`,
	`(?i)private[_-]?key`,
	`(?i)ssn|social[_-]?security`,
	`(?i)credit[_-]?card`,
	`(?i)bank[_-]?account`,
	`(?i)routing[_-]?number`,
	`(?i)license[_-]?number`,
	`(?i)passport`,
	`(?i)tax[_-]?id`,
	`(?i)date[_-]?of[_-]?birth|dob`,
	`(?i)salary|wage|income`,
	`(?i)medical|health`,
	`(?i)configuration\.apiKey`, // Specific pattern for nested API keys
}

// NewSensitiveDataManagerV2 creates a new improved sensitive data manager
func NewSensitiveDataManagerV2() *SensitiveDataManagerV2 {
	sdm := &SensitiveDataManagerV2{
		compiledFieldPatterns: make([]*regexp.Regexp, 0, len(sensitiveFieldPatterns)),
	}
	sdm.autoDetect.Store(true)
	sdm.strategy.Store(int32(MaskStrategyDefault))

	// Pre-compile field patterns for better performance
	sdm.precompileFieldPatterns()

	return sdm
}

// precompileFieldPatterns compiles all field patterns at initialization
func (s *SensitiveDataManagerV2) precompileFieldPatterns() {
	s.patternMutex.Lock()
	defer s.patternMutex.Unlock()

	for _, pattern := range sensitiveFieldPatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			s.compiledFieldPatterns = append(s.compiledFieldPatterns, regex)
		}
	}
}

// RegisterSensitiveFields registers sensitive fields for a resource
func (s *SensitiveDataManagerV2) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	if len(fields) == 0 {
		return nil
	}

	// Load existing fields for this resource
	existingFieldsInterface, _ := s.fields.LoadOrStore(
		resource,
		make(map[string]SensitiveFieldConfig),
	)
	existingFields := existingFieldsInterface.(map[string]SensitiveFieldConfig)

	// Create a new map to avoid concurrent modification
	newFields := make(map[string]SensitiveFieldConfig, len(existingFields)+len(fields))
	for k, v := range existingFields {
		newFields[k] = v
	}

	// Add new fields
	for _, field := range fields {
		key := field.Path + "." + field.Name
		if field.Path == "" {
			key = field.Name
		}

		newFields[key] = SensitiveFieldConfig{
			Path:   field.Path,
			Name:   field.Name,
			Action: field.Action,
		}
	}

	// Store the updated map atomically
	s.fields.Store(resource, newFields)

	return nil
}

// SanitizeEntry sanitizes sensitive data in an audit entry
func (s *SensitiveDataManagerV2) SanitizeEntry(entry *audit.Entry) error {
	if entry == nil {
		return nil
	}

	// Get registered fields for this resource
	var registeredFields map[string]SensitiveFieldConfig
	if fieldsInterface, ok := s.fields.Load(entry.Resource); ok {
		registeredFields = fieldsInterface.(map[string]SensitiveFieldConfig)
	}

	// Sanitize all data fields
	if entry.CurrentState != nil {
		s.sanitizeMap(entry.CurrentState, registeredFields, "")
	}
	if entry.PreviousState != nil {
		s.sanitizeMap(entry.PreviousState, registeredFields, "")
	}
	if entry.Metadata != nil {
		s.sanitizeMap(entry.Metadata, registeredFields, "")
	}

	// IMPORTANT: Also sanitize the Changes field which contains diff data
	if entry.Changes != nil {
		s.sanitizeChangesMap(entry.Changes, registeredFields)
	}

	// IMPORTANT: Sanitize the User object if present
	// The User field is a relationship that contains sensitive data like email addresses
	if entry.User != nil {
		s.sanitizeUserObject(entry.User)
	}

	return nil
}

// sanitizeChangesMap specifically handles the Changes field which has a special structure
func (s *SensitiveDataManagerV2) sanitizeChangesMap(
	changes map[string]any,
	registeredFields map[string]SensitiveFieldConfig,
) {
	for changePath, changeData := range changes {
		// Check if this change path is for a sensitive field
		isSensitive := false

		// Check registered fields
		if _, ok := registeredFields[changePath]; ok {
			isSensitive = true
		}

		// Check auto-detection patterns
		if !isSensitive && s.autoDetect.Load() && s.isSensitiveFieldPath(changePath) {
			isSensitive = true
		}

		if isSensitive {
			// Sanitize the change data
			if changeMap, ok := changeData.(map[string]any); ok {
				// Mask the "from" and "to" values
				if from, exists := changeMap["from"]; exists && from != nil {
					changeMap["from"] = s.maskValue(from, services.SensitiveFieldMask)
				}
				if to, exists := changeMap["to"]; exists && to != nil {
					changeMap["to"] = s.maskValue(to, services.SensitiveFieldMask)
				}
			}
		} else if changeMap, ok := changeData.(map[string]any); ok {
			// Even if the path itself isn't sensitive, check the actual values
			if s.autoDetect.Load() {
				// Check if the values contain sensitive patterns
				if from, exists := changeMap["from"]; exists && from != nil {
					if s.shouldMaskValue(from) {
						changeMap["from"] = s.maskValue(from, services.SensitiveFieldMask)
					}
				}
				if to, exists := changeMap["to"]; exists && to != nil {
					if s.shouldMaskValue(to) {
						changeMap["to"] = s.maskValue(to, services.SensitiveFieldMask)
					}
				}
			}
		}
	}
}

// sanitizeUserObject specifically handles the User field in audit entries
func (s *SensitiveDataManagerV2) sanitizeUserObject(u *user.User) {
	if u == nil {
		return
	}

	strategy := MaskStrategy(s.strategy.Load())

	// Always mask the email address
	if u.EmailAddress != "" {
		switch strategy {
		case MaskStrategyStrict:
			// Show only domain
			parts := strings.Split(u.EmailAddress, "@")
			if len(parts) == 2 {
				u.EmailAddress = "****@" + parts[1]
			} else {
				u.EmailAddress = "****"
			}
		case MaskStrategyDefault:
			// Show first character and domain
			parts := strings.Split(u.EmailAddress, "@")
			if len(parts) == 2 && len(parts[0]) > 0 {
				u.EmailAddress = parts[0][:1] + strings.Repeat(
					"*",
					len(parts[0])-1,
				) + "@" + parts[1]
			} else {
				u.EmailAddress = "****"
			}
		case MaskStrategyPartial:
			// Show first 2 characters and domain
			parts := strings.Split(u.EmailAddress, "@")
			if len(parts) == 2 && len(parts[0]) > 2 {
				u.EmailAddress = parts[0][:2] + strings.Repeat(
					"*",
					len(parts[0])-2,
				) + "@" + parts[1]
			} else if len(parts) == 2 {
				u.EmailAddress = parts[0] + "@" + parts[1]
			} else {
				u.EmailAddress = "****"
			}
		}
	}

	// Mask IDs based on strategy
	if strategy != MaskStrategyPartial {
		// For strict and default strategies, mask IDs
		if u.ID != "" {
			u.ID = s.maskPulID(u.ID)
		}
		if u.BusinessUnitID != "" {
			u.BusinessUnitID = s.maskPulID(u.BusinessUnitID)
		}
		if u.CurrentOrganizationID != "" {
			u.CurrentOrganizationID = s.maskPulID(u.CurrentOrganizationID)
		}
	}

	// Clear profile picture URLs for privacy
	if s.autoDetect.Load() && strategy == MaskStrategyStrict {
		u.ProfilePicURL = ""
		u.ThumbnailURL = ""
	}
}

// maskPulID masks a PULID keeping the prefix and masking the rest
func (s *SensitiveDataManagerV2) maskPulID(id pulid.ID) pulid.ID {
	idStr := string(id)
	if len(idStr) > 4 {
		// Keep the prefix (e.g., "usr_") and mask the rest
		parts := strings.Split(idStr, "_")
		if len(parts) == 2 {
			return pulid.ID(parts[0] + "_" + strings.Repeat("*", len(parts[1])))
		}
	}
	return pulid.ID("****")
}

// shouldMaskValue checks if a value contains sensitive patterns
func (s *SensitiveDataManagerV2) shouldMaskValue(value any) bool {
	switch v := value.(type) {
	case string:
		return s.containsSensitivePattern(v)
	case map[string]any:
		// For nested objects, check if any field contains sensitive data
		for key, val := range v {
			if s.isSensitiveFieldName(key) {
				return true
			}
			if strVal, ok := val.(string); ok && s.containsSensitivePattern(strVal) {
				return true
			}
		}
	}
	return false
}

// sanitizeMap recursively sanitizes a map
func (s *SensitiveDataManagerV2) sanitizeMap(
	data map[string]any,
	registeredFields map[string]SensitiveFieldConfig,
	currentPath string,
) {
	for key, value := range data {
		fullPath := key
		if currentPath != "" {
			fullPath = currentPath + "." + key
		}

		// Check if this field is registered as sensitive
		if config, ok := registeredFields[fullPath]; ok {
			data[key] = s.maskValue(value, config.Action)
			continue
		}

		// Check if this field is registered without path
		if config, ok := registeredFields[key]; ok && config.Path == "" {
			data[key] = s.maskValue(value, config.Action)
			continue
		}

		// Auto-detect sensitive fields if enabled
		if s.autoDetect.Load() &&
			(s.isSensitiveFieldName(key) || s.isSensitiveFieldPath(fullPath)) {
			data[key] = s.maskValue(value, services.SensitiveFieldMask)
			continue
		}

		// Check for sensitive patterns in string values if auto-detect is enabled
		if s.autoDetect.Load() {
			if strValue, ok := value.(string); ok && s.containsSensitivePattern(strValue) {
				data[key] = s.maskValue(value, services.SensitiveFieldMask)
				continue
			}
		}

		// Recursively process nested maps
		if nestedMap, ok := value.(map[string]any); ok {
			s.sanitizeMap(nestedMap, registeredFields, fullPath)
		}

		// Process arrays and slices with improved path handling
		if slice, ok := value.([]any); ok {
			s.sanitizeSlice(slice, registeredFields, fullPath)
		}
	}
}

// sanitizeSlice handles array/slice sanitization with better path tracking
func (s *SensitiveDataManagerV2) sanitizeSlice(
	slice []any,
	registeredFields map[string]SensitiveFieldConfig,
	parentPath string,
) {
	for i, item := range slice {
		// Build array-aware path (e.g., "shipmentMoves[0]")
		arrayPath := fmt.Sprintf("%s[%d]", parentPath, i)

		switch v := item.(type) {
		case map[string]any:
			// For maps in arrays, sanitize with array-aware path
			s.sanitizeMapInArray(v, registeredFields, parentPath, arrayPath)
		case string:
			// Check string items in arrays
			if s.autoDetect.Load() && s.containsSensitivePattern(v) {
				slice[i] = s.maskValue(v, services.SensitiveFieldMask)
			}
		case []any:
			// Handle nested arrays
			s.sanitizeSlice(v, registeredFields, arrayPath)
		}
	}
}

// sanitizeMapInArray handles maps within arrays with special path considerations
func (s *SensitiveDataManagerV2) sanitizeMapInArray(
	data map[string]any,
	registeredFields map[string]SensitiveFieldConfig,
	genericPath string, // e.g., "shipmentMoves"
	specificPath string, // e.g., "shipmentMoves[0]"
) {
	for key, value := range data {
		// Check both generic and specific paths
		genericFieldPath := genericPath + "." + key
		specificFieldPath := specificPath + "." + key

		// Check if field is registered with array notation (e.g., "shipmentMoves[].tractorId")
		arrayNotationPath := genericPath + "[]." + key

		// Check all possible path variations
		shouldMask := false
		var config SensitiveFieldConfig

		if c, ok := registeredFields[genericFieldPath]; ok {
			shouldMask = true
			config = c
		} else if c, ok := registeredFields[specificFieldPath]; ok {
			shouldMask = true
			config = c
		} else if c, ok := registeredFields[arrayNotationPath]; ok {
			shouldMask = true
			config = c
		} else if c, ok := registeredFields[key]; ok && c.Path == "" {
			shouldMask = true
			config = c
		}

		if shouldMask {
			data[key] = s.maskValue(value, config.Action)
			continue
		}

		// Auto-detect sensitive fields
		if s.autoDetect.Load() &&
			(s.isSensitiveFieldName(key) || s.isSensitiveFieldPath(genericFieldPath)) {
			data[key] = s.maskValue(value, services.SensitiveFieldMask)
			continue
		}

		// Check for sensitive patterns in string values
		if s.autoDetect.Load() {
			if strValue, ok := value.(string); ok && s.containsSensitivePattern(strValue) {
				data[key] = s.maskValue(value, services.SensitiveFieldMask)
				continue
			}
		}

		// Recursively process nested structures
		switch v := value.(type) {
		case map[string]any:
			s.sanitizeMap(v, registeredFields, specificFieldPath)
		case []any:
			s.sanitizeSlice(v, registeredFields, specificFieldPath)
		}
	}
}

// isSensitiveFieldName checks if a field name matches sensitive patterns
func (s *SensitiveDataManagerV2) isSensitiveFieldName(fieldName string) bool {
	fieldNameLower := strings.ToLower(fieldName)

	s.patternMutex.RLock()
	defer s.patternMutex.RUnlock()

	for _, regex := range s.compiledFieldPatterns {
		if regex.MatchString(fieldNameLower) {
			return true
		}
	}

	return false
}

// isSensitiveFieldPath checks if a full field path matches sensitive patterns
func (s *SensitiveDataManagerV2) isSensitiveFieldPath(fieldPath string) bool {
	fieldPathLower := strings.ToLower(fieldPath)

	s.patternMutex.RLock()
	defer s.patternMutex.RUnlock()

	for _, regex := range s.compiledFieldPatterns {
		if regex.MatchString(fieldPathLower) {
			return true
		}
	}

	return false
}

// containsSensitivePattern checks if a string contains sensitive data patterns
func (s *SensitiveDataManagerV2) containsSensitivePattern(value string) bool {
	// Quick check for common API key patterns
	if len(value) >= 20 && regexp.MustCompile(`^[A-Za-z0-9_\-]+$`).MatchString(value) {
		// Likely an API key or token
		return true
	}

	for _, pattern := range sensitivePatterns {
		regex := s.getCompiledPattern(pattern)
		if regex != nil && regex.MatchString(value) {
			return true
		}
	}

	return false
}

// getCompiledPattern retrieves or compiles a regex pattern with caching
func (s *SensitiveDataManagerV2) getCompiledPattern(pattern string) *regexp.Regexp {
	// Check cache first
	if compiled, ok := s.patternCache.Load(pattern); ok {
		return compiled.(*regexp.Regexp)
	}

	// Compile and cache the pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Log error but don't fail - pattern might be invalid
		return nil
	}

	s.patternCache.Store(pattern, regex)
	return regex
}

// maskValue masks a value based on the action and current strategy
func (s *SensitiveDataManagerV2) maskValue(value any, action services.SensitiveFieldAction) any {
	switch action {
	case services.SensitiveFieldOmit:
		return nil

	case services.SensitiveFieldMask:
		return s.applyMaskStrategy(value)

	default:
		return value
	}
}

// applyMaskStrategy applies the current masking strategy to a value
func (s *SensitiveDataManagerV2) applyMaskStrategy(value any) any {
	strategy := MaskStrategy(s.strategy.Load())

	switch v := value.(type) {
	case string:
		if v == "" {
			return ""
		}

		switch strategy {
		case MaskStrategyStrict:
			// Show nothing
			return "****"

		case MaskStrategyDefault:
			// Show first and last character for strings longer than 4
			if len(v) > 4 {
				return v[:1] + strings.Repeat("*", len(v)-2) + v[len(v)-1:]
			}
			return "****"

		case MaskStrategyPartial:
			// Show more information
			if len(v) > 8 {
				return v[:3] + strings.Repeat("*", len(v)-6) + v[len(v)-3:]
			} else if len(v) > 4 {
				return v[:2] + strings.Repeat("*", len(v)-3) + v[len(v)-1:]
			}
			return "****"
		}

	case int, int32, int64, float32, float64:
		switch strategy {
		case MaskStrategyStrict:
			return 0
		case MaskStrategyDefault, MaskStrategyPartial:
			// For numbers, return a masked string representation
			return "****"
		}

	case bool:
		// Booleans are typically not sensitive
		return v

	case map[string]any:
		// For nested objects in Changes, mask the entire object
		return "[REDACTED]"

	default:
		// For other types, return mask
		return "****"
	}

	return "****"
}

// SetAutoDetect enables or disables automatic sensitive data detection
func (s *SensitiveDataManagerV2) SetAutoDetect(enabled bool) {
	s.autoDetect.Store(enabled)
}

// SetMaskStrategy sets the masking strategy
func (s *SensitiveDataManagerV2) SetMaskStrategy(strategy MaskStrategy) {
	s.strategy.Store(int32(strategy))
}

// ClearCache clears the pattern cache (useful for memory management)
func (s *SensitiveDataManagerV2) ClearCache() {
	s.patternCache.Range(func(key, value any) bool {
		s.patternCache.Delete(key)
		return true
	})
}

// AddCustomPattern adds a custom sensitive data pattern
func (s *SensitiveDataManagerV2) AddCustomPattern(name, pattern string) error {
	// Validate the pattern
	_, err := regexp.Compile(pattern)
	if err != nil {
		return eris.Wrap(err, "invalid regex pattern")
	}

	// Add to patterns
	sensitivePatterns[name] = pattern
	return nil
}

// RemoveCustomPattern removes a custom sensitive data pattern
func (s *SensitiveDataManagerV2) RemoveCustomPattern(name string) {
	delete(sensitivePatterns, name)
	// Also remove from cache if present
	s.patternCache.Delete(sensitivePatterns[name])
}
