package audit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/atomic"
)

const (
	starPattern = "****"
)

type MaskStrategy int32

const (
	MaskStrategyStrict MaskStrategy = iota
	MaskStrategyDefault
	MaskStrategyPartial
)

type SensitiveDataManager struct {
	fields                sync.Map // map[permission.Resource]map[string]SensitiveFieldConfig
	patternCache          sync.Map // map[string]*regexp.Regexp
	autoDetect            atomic.Bool
	strategy              atomic.Int32
	compiledFieldPatterns []*regexp.Regexp
	patternMutex          sync.RWMutex
	encryptionKey         []byte // 32-byte key for AES-256
	encryptionKeyMutex    sync.RWMutex
}

type SensitiveFieldConfig struct {
	Path   string
	Name   string
	Action services.SensitiveFieldAction
}

var sensitivePatterns = map[string]string{
	"ssn":            `\b\d{3}-\d{2}-\d{4}\b|\b\d{9}\b`,
	"creditCard":     `\b(?:\d[ -]*?){13,19}\b`,
	"email":          `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
	"phone":          `\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`,
	"apiKey":         `\b(?i)(api[_-]?key|apikey|api[_-]?secret|access[_-]?token|auth[_-]?token|bearer)\s*[:=]\s*["']?[\w\-]{20,}["']?\b|^[A-Za-z0-9]{20,}$`,
	"googleApiKey":   `\bAIza[0-9A-Za-z\-_]{35}\b`,
	"awsKey":         `\b(?:AKIA|A3T|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[0-9A-Z]{16}\b`,
	"jwt":            `\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`,
	"dbConnection":   `\b(?i)(mongodb|postgres|postgresql|mysql|redis|mssql|oracle):\/\/[^\s]+\b`,
	"privateKey":     `-----BEGIN\s+(?:RSA\s+)?PRIVATE\s+KEY-----[\s\S]+?-----END\s+(?:RSA\s+)?PRIVATE\s+KEY-----`,
	"ipAddress":      `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
	"bankAccount":    `\b\d{9}\s*\d{1,17}\b`,
	"driversLicense": `\b(?i)(license|licence|dl|driver)\s*(?:number|no|#)?\s*[:=]?\s*[A-Z0-9]{5,20}\b`,
	"dateOfBirth":    `\b(?:0[1-9]|1[0-2])[-/](?:0[1-9]|[12][0-9]|3[01])[-/](?:19|20)\d{2}\b`,
	"taxId":          `\b\d{2}-\d{7}\b`,
	"passport":       `\b(?i)passport\s*(?:number|no|#)?\s*[:=]?\s*[A-Z0-9]{6,20}\b`,
}

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
	`(?i)configuration\.apiKey`,
}

func NewSensitiveDataManager(cfg config.EncryptionConfig) *SensitiveDataManager {
	sdm := &SensitiveDataManager{
		compiledFieldPatterns: make([]*regexp.Regexp, 0, len(sensitiveFieldPatterns)),
	}
	sdm.autoDetect.Store(true)
	sdm.strategy.Store(int32(MaskStrategyDefault))

	sdm.precompileFieldPatterns()

	if cfg.Key != "" {
		hash := sha256.Sum256([]byte(cfg.Key))
		sdm.encryptionKey = hash[:]
	}

	return sdm
}

func (s *SensitiveDataManager) precompileFieldPatterns() {
	s.patternMutex.Lock()
	defer s.patternMutex.Unlock()

	for _, pattern := range sensitiveFieldPatterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			s.compiledFieldPatterns = append(s.compiledFieldPatterns, regex)
		}
	}
}

func (s *SensitiveDataManager) RegisterSensitiveFields(
	resource permission.Resource,
	fields []services.SensitiveField,
) error {
	if len(fields) == 0 {
		return nil
	}

	existingFieldsInterface, _ := s.fields.LoadOrStore(
		resource,
		make(map[string]SensitiveFieldConfig),
	)
	existingFields, _ := existingFieldsInterface.(map[string]SensitiveFieldConfig)

	newFields := make(map[string]SensitiveFieldConfig, len(existingFields)+len(fields))
	for k, v := range existingFields {
		newFields[k] = v
	}

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

	s.fields.Store(resource, newFields)

	return nil
}

// SanitizeEntry sanitizes sensitive data in an audit entry
func (s *SensitiveDataManager) SanitizeEntry(entry *audit.Entry) error {
	if entry == nil {
		return nil
	}

	var registeredFields map[string]SensitiveFieldConfig
	if fieldsInterface, ok := s.fields.Load(entry.Resource); ok {
		registeredFields, _ = fieldsInterface.(map[string]SensitiveFieldConfig)
	}

	if entry.CurrentState != nil {
		s.sanitizeMap(entry.CurrentState, registeredFields, "")
	}
	if entry.PreviousState != nil {
		s.sanitizeMap(entry.PreviousState, registeredFields, "")
	}
	if entry.Metadata != nil {
		s.sanitizeMap(entry.Metadata, registeredFields, "")
	}

	// ! IMPORTANT: Also sanitize the Changes field which contains diff data
	if entry.Changes != nil {
		s.sanitizeChangesMap(entry.Changes, registeredFields)
	}

	// ! IMPORTANT: Sanitize the User object if present
	// ! The User field is a relationship that contains sensitive data like email addresses
	if entry.User != nil {
		s.sanitizeUserObject(entry.User)
	}

	return nil
}

func (s *SensitiveDataManager) sanitizeChangesMap( //nolint:gocognit // TODO: refactor this
	changes map[string]any,
	registeredFields map[string]SensitiveFieldConfig,
) {
	for changePath, changeData := range changes {
		isSensitive := false

		if _, ok := registeredFields[changePath]; ok {
			isSensitive = true
		}

		if !isSensitive && s.autoDetect.Load() && s.isSensitiveFieldPath(changePath) {
			isSensitive = true
		}

		if isSensitive { //nolint:nestif // TODO: refactor this
			if changeMap, ok := changeData.(map[string]any); ok {
				if from, exists := changeMap["from"]; exists && from != nil {
					changeMap["from"] = s.maskValue(from, services.SensitiveFieldMask)
				}
				if to, exists := changeMap["to"]; exists && to != nil {
					changeMap["to"] = s.maskValue(to, services.SensitiveFieldMask)
				}
			}
		} else if changeMap, ok := changeData.(map[string]any); ok {
			if s.autoDetect.Load() {
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

func (s *SensitiveDataManager) sanitizeUserObject(u *tenant.User) {
	if u == nil {
		return
	}

	strategy := MaskStrategy(s.strategy.Load())

	if u.EmailAddress != "" { //nolint:nestif // TODO: refactor this
		switch strategy {
		case MaskStrategyStrict:
			parts := strings.Split(u.EmailAddress, "@")
			if len(parts) == 2 {
				u.EmailAddress = "****@" + parts[1]
			} else {
				u.EmailAddress = starPattern
			}
		case MaskStrategyDefault:
			parts := strings.Split(u.EmailAddress, "@")
			if len(parts) == 2 && parts[0] != "" {
				u.EmailAddress = parts[0][:1] + strings.Repeat(
					"*",
					len(parts[0])-1,
				) + "@" + parts[1]
			} else {
				u.EmailAddress = starPattern
			}
		case MaskStrategyPartial:
			parts := strings.Split(u.EmailAddress, "@")
			switch len(parts) {
			case 2:
				if len(parts[0]) > 2 {
					u.EmailAddress = parts[0][:2] + strings.Repeat(
						"*",
						len(parts[0])-2,
					) + "@" + parts[1]
				} else {
					u.EmailAddress = parts[0] + "@" + parts[1]
				}
			default:
				u.EmailAddress = starPattern
			}
		}
	}

	if strategy != MaskStrategyPartial {
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

	if s.autoDetect.Load() && strategy == MaskStrategyStrict {
		u.ProfilePicURL = ""
		u.ThumbnailURL = ""
	}
}

func (s *SensitiveDataManager) maskPulID(id pulid.ID) pulid.ID {
	idStr := string(id)
	if len(idStr) > 4 {
		parts := strings.Split(idStr, "_")
		if len(parts) == 2 {
			return pulid.ID(parts[0] + "_" + strings.Repeat("*", len(parts[1])))
		}
	}
	return pulid.ID(starPattern)
}

func (s *SensitiveDataManager) shouldMaskValue(value any) bool {
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

func (s *SensitiveDataManager) sanitizeMap(
	data map[string]any,
	registeredFields map[string]SensitiveFieldConfig,
	currentPath string,
) {
	for key, value := range data {
		fullPath := key
		if currentPath != "" {
			fullPath = currentPath + "." + key
		}

		if cfg, ok := registeredFields[fullPath]; ok {
			data[key] = s.maskValue(value, cfg.Action)
			continue
		}

		if cfg, ok := registeredFields[key]; ok && cfg.Path == "" {
			data[key] = s.maskValue(value, cfg.Action)
			continue
		}

		if s.autoDetect.Load() &&
			(s.isSensitiveFieldName(key) || s.isSensitiveFieldPath(fullPath)) {
			data[key] = s.maskValue(value, services.SensitiveFieldMask)
			continue
		}

		if s.autoDetect.Load() {
			if strValue, ok := value.(string); ok && s.containsSensitivePattern(strValue) {
				data[key] = s.maskValue(value, services.SensitiveFieldMask)
				continue
			}
		}

		if nestedMap, ok := value.(map[string]any); ok {
			s.sanitizeMap(nestedMap, registeredFields, fullPath)
		}

		if slice, ok := value.([]any); ok {
			s.sanitizeSlice(slice, registeredFields, fullPath)
		}
	}
}

func (s *SensitiveDataManager) sanitizeSlice(
	slice []any,
	registeredFields map[string]SensitiveFieldConfig,
	parentPath string,
) {
	for i, item := range slice {
		arrayPath := fmt.Sprintf("%s[%d]", parentPath, i)

		switch v := item.(type) {
		case map[string]any:
			s.sanitizeMapInArray(v, registeredFields, parentPath, arrayPath)
		case string:
			if s.autoDetect.Load() && s.containsSensitivePattern(v) {
				slice[i] = s.maskValue(v, services.SensitiveFieldMask)
			}
		case []any:
			s.sanitizeSlice(v, registeredFields, arrayPath)
		}
	}
}

func (s *SensitiveDataManager) sanitizeMapInArray(
	data map[string]any,
	registeredFields map[string]SensitiveFieldConfig,
	genericPath string, // e.g., "shipmentMoves"
	specificPath string, // e.g., "shipmentMoves[0]"
) {
	for key, value := range data {
		genericFieldPath := genericPath + "." + key
		specificFieldPath := specificPath + "." + key

		arrayNotationPath := genericPath + "[]." + key

		shouldMask := false
		var cfg SensitiveFieldConfig

		if c, ok := registeredFields[genericFieldPath]; ok {
			shouldMask = true
			cfg = c
		} else if sc, sok := registeredFields[specificFieldPath]; sok {
			shouldMask = true
			cfg = sc
		} else if ac, aok := registeredFields[arrayNotationPath]; aok {
			shouldMask = true
			cfg = ac
		} else if bc, bok := registeredFields[key]; bok && c.Path == "" {
			shouldMask = true
			cfg = bc
		}

		if shouldMask {
			data[key] = s.maskValue(value, cfg.Action)
			continue
		}

		if s.autoDetect.Load() &&
			(s.isSensitiveFieldName(key) || s.isSensitiveFieldPath(genericFieldPath)) {
			data[key] = s.maskValue(value, services.SensitiveFieldMask)
			continue
		}

		if s.autoDetect.Load() {
			if strValue, ok := value.(string); ok && s.containsSensitivePattern(strValue) {
				data[key] = s.maskValue(value, services.SensitiveFieldMask)
				continue
			}
		}

		switch v := value.(type) {
		case map[string]any:
			s.sanitizeMap(v, registeredFields, specificFieldPath)
		case []any:
			s.sanitizeSlice(v, registeredFields, specificFieldPath)
		}
	}
}

func (s *SensitiveDataManager) isSensitiveFieldName(fieldName string) bool {
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

func (s *SensitiveDataManager) isSensitiveFieldPath(fieldPath string) bool {
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

func (s *SensitiveDataManager) containsSensitivePattern(value string) bool {
	if len(value) >= 20 && regexp.MustCompile(`^[A-Za-z0-9_\-]+$`).MatchString(value) {
		return true
	}

	for _, pattern := range sensitivePatterns {
		regex, err := s.getCompiledPattern(pattern)
		if err != nil {
			return false
		}

		if regex.MatchString(value) {
			return true
		}
	}

	return false
}

func (s *SensitiveDataManager) getCompiledPattern(pattern string) (*regexp.Regexp, error) {
	if compiled, ok := s.patternCache.Load(pattern); ok {
		compiled, cOk := compiled.(*regexp.Regexp)
		if !cOk {
			return nil, ErrCompiledPatternNotRegexp
		}
		return compiled, nil
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	s.patternCache.Store(pattern, regex)
	return regex, nil
}

func (s *SensitiveDataManager) maskValue(value any, action services.SensitiveFieldAction) any {
	switch action {
	case services.SensitiveFieldOmit:
		return nil

	case services.SensitiveFieldMask:
		return s.applyMaskStrategy(value)

	case services.SensitiveFieldHash:
		return s.applyHashStrategy(value)

	case services.SensitiveFieldEncrypt:
		return s.applyEncryptStrategy(value)

	default:
		return value
	}
}

func (s *SensitiveDataManager) applyMaskStrategy(value any) any {
	strategy := MaskStrategy(s.strategy.Load())

	switch v := value.(type) {
	case string:
		if v == "" {
			return ""
		}

		switch strategy {
		case MaskStrategyStrict:
			return starPattern

		case MaskStrategyDefault:
			if len(v) > 4 {
				return v[:1] + strings.Repeat("*", len(v)-2) + v[len(v)-1:]
			}
			return starPattern

		case MaskStrategyPartial:
			if len(v) > 8 {
				return v[:3] + strings.Repeat("*", len(v)-6) + v[len(v)-3:]
			} else if len(v) > 4 {
				return v[:2] + strings.Repeat("*", len(v)-3) + v[len(v)-1:]
			}
			return starPattern
		}

	case int, int32, int64, float32, float64:
		switch strategy {
		case MaskStrategyStrict:
			return 0
		case MaskStrategyDefault, MaskStrategyPartial:
			return starPattern
		}

	case bool:
		return v

	case map[string]any:
		return "[REDACTED]"

	default:
		return starPattern
	}

	return starPattern
}

func (s *SensitiveDataManager) SetAutoDetect(enabled bool) {
	s.autoDetect.Store(enabled)
}

func (s *SensitiveDataManager) SetMaskStrategy(strategy MaskStrategy) {
	s.strategy.Store(int32(strategy))
}

func (s *SensitiveDataManager) ClearCache() {
	s.patternCache.Range(func(key, _ any) bool {
		s.patternCache.Delete(key)
		return true
	})
}

func (s *SensitiveDataManager) AddCustomPattern(name, pattern string) error {
	_, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	sensitivePatterns[name] = pattern
	return nil
}

func (s *SensitiveDataManager) RemoveCustomPattern(name string) {
	delete(sensitivePatterns, name)
	s.patternCache.Delete(sensitivePatterns[name])
}

func (s *SensitiveDataManager) applyHashStrategy(value any) any {
	switch v := value.(type) {
	case string:
		if v == "" {
			return ""
		}
		hash := sha256.Sum256([]byte(v))
		return "SHA256:" + hex.EncodeToString(hash[:])

	case int, int32, int64, float32, float64:
		strVal := fmt.Sprintf("%v", v)
		hash := sha256.Sum256([]byte(strVal))
		return "SHA256:" + hex.EncodeToString(hash[:])

	case bool:
		strVal := strconv.FormatBool(v)
		hash := sha256.Sum256([]byte(strVal))
		return "SHA256:" + hex.EncodeToString(hash[:])

	case map[string]any:
		strVal := fmt.Sprintf("%v", v)
		hash := sha256.Sum256([]byte(strVal))
		return "SHA256:" + hex.EncodeToString(hash[:])

	case []any:
		strVal := fmt.Sprintf("%v", v)
		hash := sha256.Sum256([]byte(strVal))
		return "SHA256:" + hex.EncodeToString(hash[:])

	default:
		strVal := fmt.Sprintf("%v", v)
		hash := sha256.Sum256([]byte(strVal))
		return "SHA256:" + hex.EncodeToString(hash[:])
	}
}

func (s *SensitiveDataManager) applyEncryptStrategy(value any) any {
	s.encryptionKeyMutex.RLock()
	key := s.encryptionKey
	s.encryptionKeyMutex.RUnlock()

	if len(key) == 0 {
		return s.applyMaskStrategy(value)
	}

	var plaintext []byte
	switch v := value.(type) {
	case string:
		if v == "" {
			return ""
		}
		plaintext = []byte(v)

	case int, int32, int64, float32, float64, bool:
		plaintext = []byte(fmt.Sprintf("%v", v))

	case map[string]any, []any:
		plaintext = []byte(fmt.Sprintf("%v", v))

	default:
		plaintext = []byte(fmt.Sprintf("%v", v))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return s.applyMaskStrategy(value)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return s.applyMaskStrategy(value)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return s.applyMaskStrategy(value)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return "ENC:" + base64.StdEncoding.EncodeToString(ciphertext)
}
