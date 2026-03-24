package auditservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeMapInArray_RegisteredGenericFieldPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	registeredFields := map[string]SensitiveFieldConfig{
		"items.secret": {Path: "items", Name: "secret", Action: services.SensitiveFieldOmit},
	}

	data := map[string]any{
		"secret": "hidden-value",
		"name":   "visible",
	}

	sdm.sanitizeMapInArray(data, registeredFields, "items", "items[0]")

	assert.Nil(t, data["secret"])
	assert.Equal(t, "visible", data["name"])
}

func TestSanitizeMapInArray_RegisteredSpecificFieldPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	registeredFields := map[string]SensitiveFieldConfig{
		"items[0].token": {Path: "items[0]", Name: "token", Action: services.SensitiveFieldMask},
	}

	data := map[string]any{
		"token": "my-secret-token",
		"id":    "123",
	}

	sdm.sanitizeMapInArray(data, registeredFields, "items", "items[0]")

	assert.Equal(t, "****", data["token"])
	assert.Equal(t, "123", data["id"])
}

func TestSanitizeMapInArray_RegisteredArrayNotationPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	registeredFields := map[string]SensitiveFieldConfig{
		"moves[].apiKey": {Path: "moves[]", Name: "apiKey", Action: services.SensitiveFieldOmit},
	}

	data := map[string]any{
		"apiKey": "secret-api-key",
		"status": "active",
	}

	sdm.sanitizeMapInArray(data, registeredFields, "moves", "moves[2]")

	assert.Nil(t, data["apiKey"])
	assert.Equal(t, "active", data["status"])
}

func TestSanitizeMapInArray_RegisteredKeyOnlyNoPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	registeredFields := map[string]SensitiveFieldConfig{
		"credential": {Path: "", Name: "credential", Action: services.SensitiveFieldOmit},
	}

	data := map[string]any{
		"credential": "secret-cred",
		"label":      "test",
	}

	sdm.sanitizeMapInArray(data, registeredFields, "entries", "entries[0]")

	assert.Nil(t, data["credential"])
	assert.Equal(t, "test", data["label"])
}

func TestSanitizeMapInArray_AutoDetectFieldName(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	data := map[string]any{
		"password":  "secret-pass",
		"firstName": "John",
	}

	sdm.sanitizeMapInArray(data, nil, "users", "users[0]")

	assert.Equal(t, "****", data["password"])
	assert.Equal(t, "John", data["firstName"])
}

func TestSanitizeMapInArray_AutoDetectSensitiveValue(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	data := map[string]any{
		"note": "My SSN is 123-45-6789",
		"city": "Springfield",
	}

	sdm.sanitizeMapInArray(data, nil, "records", "records[0]")

	assert.Equal(t, "****", data["note"])
	assert.Equal(t, "Springfield", data["city"])
}

func TestSanitizeMapInArray_NestedMap(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	registeredFields := map[string]SensitiveFieldConfig{
		"items[0].nested.secret": {
			Path:   "items[0].nested",
			Name:   "secret",
			Action: services.SensitiveFieldOmit,
		},
	}

	data := map[string]any{
		"nested": map[string]any{
			"secret": "hidden",
			"public": "visible",
		},
	}

	sdm.sanitizeMapInArray(data, registeredFields, "items", "items[0]")

	nestedMap := data["nested"].(map[string]any)
	assert.Nil(t, nestedMap["secret"])
	assert.Equal(t, "visible", nestedMap["public"])
}

func TestSanitizeMapInArray_NestedSlice(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	data := map[string]any{
		"entries": []any{
			"123-45-6789",
			"normal-text",
		},
	}

	sdm.sanitizeMapInArray(data, nil, "items", "items[0]")

	entries := data["entries"].([]any)
	assert.Equal(t, "****", entries[0])
	assert.Equal(t, "normal-text", entries[1])
}

func TestSanitizeMapInArray_AutoDetectDisabled(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	data := map[string]any{
		"password": "secret",
		"ssn":      "123-45-6789",
	}

	sdm.sanitizeMapInArray(data, nil, "records", "records[0]")

	assert.Equal(t, "secret", data["password"])
	assert.Equal(t, "123-45-6789", data["ssn"])
}

func TestSanitizeMapInArray_AutoDetectFieldPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	data := map[string]any{
		"apiKey": "my-key-value",
	}

	sdm.sanitizeMapInArray(data, nil, "configuration", "configuration[0]")

	assert.Equal(t, "****", data["apiKey"])
}

func TestGetCompiledPattern_CachesResult(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	pattern := `\btest\b`

	regex1, err := sdm.getCompiledPattern(pattern)
	require.NoError(t, err)
	require.NotNil(t, regex1)

	regex2, err := sdm.getCompiledPattern(pattern)
	require.NoError(t, err)
	require.NotNil(t, regex2)

	assert.Equal(t, regex1, regex2)
}

func TestGetCompiledPattern_InvalidPattern(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	regex, err := sdm.getCompiledPattern(`[invalid`)
	assert.Error(t, err)
	assert.Nil(t, regex)
}

func TestGetCompiledPattern_CacheWithInvalidType(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	sdm.patternCache.Store("bad-pattern", "not-a-regexp")

	regex, err := sdm.getCompiledPattern("bad-pattern")
	assert.Error(t, err)
	assert.Nil(t, regex)
	assert.Equal(t, ErrCompiledPatternNotRegexp, err)
}

func TestApplyEncryptStrategy_Float(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy(float64(3.14))
	str, ok := result.(string)
	require.True(t, ok)
	assert.Contains(t, str, "ENC:")
}

func TestApplyEncryptStrategy_Int32(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy(int32(100))
	str, ok := result.(string)
	require.True(t, ok)
	assert.Contains(t, str, "ENC:")
}

func TestApplyEncryptStrategy_Int64(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy(int64(999))
	str, ok := result.(string)
	require.True(t, ok)
	assert.Contains(t, str, "ENC:")
}

func TestApplyEncryptStrategy_Float32(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy(float32(2.5))
	str, ok := result.(string)
	require.True(t, ok)
	assert.Contains(t, str, "ENC:")
}

func TestSanitizeEntry_WithRegisteredFieldsInArray(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	err := sdm.RegisterSensitiveFields(permission.Resource("shipment"), []services.SensitiveField{
		{Name: "driverSSN", Path: "shipmentMoves", Action: services.SensitiveFieldOmit},
	})
	require.NoError(t, err)

	entry := &audit.Entry{
		Resource: permission.Resource("shipment"),
		CurrentState: map[string]any{
			"shipmentMoves": []any{
				map[string]any{
					"driverSSN": "123-45-6789",
					"moveID":    "move_01",
				},
				map[string]any{
					"driverSSN": "987-65-4321",
					"moveID":    "move_02",
				},
			},
		},
	}

	err = sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	moves := entry.CurrentState["shipmentMoves"].([]any)
	move0 := moves[0].(map[string]any)
	move1 := moves[1].(map[string]any)
	assert.Nil(t, move0["driverSSN"])
	assert.Equal(t, "move_01", move0["moveID"])
	assert.Nil(t, move1["driverSSN"])
	assert.Equal(t, "move_02", move1["moveID"])
}

func TestSanitizeChangesMap_NonMapChangeData(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	changes := map[string]any{
		"field1": "just a string, not a map",
		"field2": 42,
	}

	sdm.sanitizeChangesMap(changes, nil)

	assert.Equal(t, "just a string, not a map", changes["field1"])
	assert.Equal(t, 42, changes["field2"])
}

func TestSanitizeChangesMap_NilFromTo(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	changes := map[string]any{
		"password": map[string]any{
			"from": nil,
			"to":   "new-value",
		},
	}

	sdm.sanitizeChangesMap(changes, nil)

	pwChange := changes["password"].(map[string]any)
	assert.Nil(t, pwChange["from"])
	assert.Equal(t, "****", pwChange["to"])
}

func TestSanitizeChangesMap_AutoDetectValueInNonSensitiveField(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	changes := map[string]any{
		"notes": map[string]any{
			"from": "safe value",
			"to":   "123-45-6789",
		},
	}

	sdm.sanitizeChangesMap(changes, nil)

	notesChange := changes["notes"].(map[string]any)
	assert.Equal(t, "safe value", notesChange["from"])
	assert.Equal(t, "****", notesChange["to"])
}

func TestSanitizeChangesMap_AutoDetectDisabledNoMasking(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	changes := map[string]any{
		"notes": map[string]any{
			"from": "123-45-6789",
			"to":   "987-65-4321",
		},
	}

	sdm.sanitizeChangesMap(changes, nil)

	notesChange := changes["notes"].(map[string]any)
	assert.Equal(t, "123-45-6789", notesChange["from"])
	assert.Equal(t, "987-65-4321", notesChange["to"])
}

func TestSanitizeMapInArray_NonStringValueAutoDetect(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)

	data := map[string]any{
		"count":  42,
		"active": true,
	}

	sdm.sanitizeMapInArray(data, nil, "items", "items[0]")

	assert.Equal(t, 42, data["count"])
	assert.Equal(t, true, data["active"])
}
