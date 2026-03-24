package auditservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeEntry_CurrentAndPreviousState(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	err := sdm.RegisterSensitiveFields(permission.Resource("users"), []services.SensitiveField{
		{Name: "password", Action: services.SensitiveFieldOmit},
	})
	require.NoError(t, err)

	entry := &audit.Entry{
		Resource: permission.Resource("users"),
		CurrentState: map[string]any{
			"password": "secret",
			"name":     "John",
		},
		PreviousState: map[string]any{
			"password": "old-secret",
			"name":     "Jane",
		},
		Metadata: map[string]any{
			"password": "meta-secret",
		},
	}

	err = sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	assert.Nil(t, entry.CurrentState["password"])
	assert.Equal(t, "John", entry.CurrentState["name"])
	assert.Nil(t, entry.PreviousState["password"])
	assert.Equal(t, "Jane", entry.PreviousState["name"])
	assert.Nil(t, entry.Metadata["password"])
}

func TestSanitizeEntry_WithChanges(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	err := sdm.RegisterSensitiveFields(permission.Resource("users"), []services.SensitiveField{
		{Name: "password", Action: services.SensitiveFieldMask},
	})
	require.NoError(t, err)

	entry := &audit.Entry{
		Resource: permission.Resource("users"),
		Changes: map[string]any{
			"password": map[string]any{
				"from": "old-password",
				"to":   "new-password",
			},
			"name": map[string]any{
				"from": "John",
				"to":   "Jane",
			},
		},
	}

	err = sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	pwChange := entry.Changes["password"].(map[string]any)
	assert.NotEqual(t, "old-password", pwChange["from"])
	assert.NotEqual(t, "new-password", pwChange["to"])

	nameChange := entry.Changes["name"].(map[string]any)
	assert.Equal(t, "John", nameChange["from"])
	assert.Equal(t, "Jane", nameChange["to"])
}

func TestSanitizeEntry_AutoDetectFieldName(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	entry := &audit.Entry{
		Resource: permission.Resource("test"),
		CurrentState: map[string]any{
			"api_key":  "my-api-key-value",
			"username": "visible",
		},
	}

	err := sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	assert.Equal(t, "****", entry.CurrentState["api_key"])
	assert.Equal(t, "visible", entry.CurrentState["username"])
}

func TestSanitizeEntry_AutoDetectPattern(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	entry := &audit.Entry{
		Resource: permission.Resource("test"),
		CurrentState: map[string]any{
			"note": "My SSN is 123-45-6789",
		},
	}

	err := sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	assert.Equal(t, "****", entry.CurrentState["note"])
}

func TestSanitizeEntry_NestedMap(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	err := sdm.RegisterSensitiveFields(permission.Resource("test"), []services.SensitiveField{
		{Name: "ssn", Path: "profile", Action: services.SensitiveFieldOmit},
	})
	require.NoError(t, err)

	entry := &audit.Entry{
		Resource: permission.Resource("test"),
		CurrentState: map[string]any{
			"profile": map[string]any{
				"ssn":  "123-45-6789",
				"name": "John",
			},
		},
	}

	err = sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	profile := entry.CurrentState["profile"].(map[string]any)
	assert.Nil(t, profile["ssn"])
	assert.Equal(t, "John", profile["name"])
}

func TestSanitizeEntry_SliceItems(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	entry := &audit.Entry{
		Resource: permission.Resource("test"),
		CurrentState: map[string]any{
			"items": []any{
				map[string]any{
					"password": "secret",
					"name":     "item1",
				},
			},
		},
	}

	err := sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	items := entry.CurrentState["items"].([]any)
	item := items[0].(map[string]any)
	assert.Equal(t, "****", item["password"])
	assert.Equal(t, "item1", item["name"])
}

func TestSanitizeEntry_SliceWithSensitiveStrings(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	entry := &audit.Entry{
		Resource: permission.Resource("test"),
		CurrentState: map[string]any{
			"items": []any{
				"123-45-6789",
				"normal text",
			},
		},
	}

	err := sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	items := entry.CurrentState["items"].([]any)
	assert.Equal(t, "****", items[0])
	assert.Equal(t, "normal text", items[1])
}

func TestSanitizeEntry_UserObject(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyDefault)

	entry := &audit.Entry{
		Resource: permission.Resource("test"),
		User: &tenant.User{
			ID:                    pulid.ID("usr_01ABCDEFGH"),
			BusinessUnitID:        pulid.ID("bu_01ABCDEFGH"),
			CurrentOrganizationID: pulid.ID("org_01ABCDEFGH"),
			EmailAddress:          "user@example.com",
		},
	}

	err := sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	assert.Contains(t, entry.User.EmailAddress, "@example.com")
	assert.NotEqual(t, "user@example.com", entry.User.EmailAddress)
}

func TestSanitizeUserObject_Nil(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.sanitizeUserObject(nil)
}

func TestSanitizeUserObject_EmptyEmail(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyStrict)

	user := &tenant.User{EmailAddress: ""}
	sdm.sanitizeUserObject(user)
	assert.Equal(t, "", user.EmailAddress)
}

func TestSanitizeUserObject_Default(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyDefault)

	user := &tenant.User{
		ID:                    pulid.ID("usr_01ABCDEF"),
		BusinessUnitID:        pulid.ID("bu_01ABCDEF"),
		CurrentOrganizationID: pulid.ID("org_01ABCDEF"),
		EmailAddress:          "longuser@example.com",
	}
	sdm.sanitizeUserObject(user)

	assert.Contains(t, user.EmailAddress, "@example.com")
	assert.Equal(t, "l", user.EmailAddress[:1])
	assert.Contains(t, string(user.ID), "*")
}

func TestSanitizeUserObject_Strict_AutoDetect(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyStrict)
	sdm.SetAutoDetect(true)

	user := &tenant.User{
		EmailAddress:  "test@example.com",
		ProfilePicURL: "https://example.com/pic.jpg",
		ThumbnailURL:  "https://example.com/thumb.jpg",
	}
	sdm.sanitizeUserObject(user)

	assert.Equal(t, "****@example.com", user.EmailAddress)
	assert.Equal(t, "", user.ProfilePicURL)
	assert.Equal(t, "", user.ThumbnailURL)
}

func TestSanitizeUserObject_Partial_ShortUsername(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyPartial)

	user := &tenant.User{
		EmailAddress: "ab@example.com",
	}
	sdm.sanitizeUserObject(user)

	assert.Equal(t, "ab@example.com", user.EmailAddress)
}

func TestSanitizeUserObject_InvalidEmailFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strategy MaskStrategy
		email    string
	}{
		{"strict_no_at", MaskStrategyStrict, "noemailhere"},
		{"default_no_at", MaskStrategyDefault, "noemailhere"},
		{"partial_no_at", MaskStrategyPartial, "noemailhere"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
			sdm.SetMaskStrategy(tt.strategy)

			user := &tenant.User{EmailAddress: tt.email}
			sdm.sanitizeUserObject(user)

			assert.Equal(t, "****", user.EmailAddress)
		})
	}
}

func TestSanitizeUserObject_DefaultEmptyLocalPart(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyDefault)

	user := &tenant.User{EmailAddress: "@example.com"}
	sdm.sanitizeUserObject(user)

	assert.Equal(t, "****", user.EmailAddress)
}

func TestMaskPulID(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.maskPulID(pulid.ID("usr_01ABCDEFGH"))
	assert.Contains(t, string(result), "usr_")
	assert.Contains(t, string(result), "*")

	result = sdm.maskPulID(pulid.ID("ab"))
	assert.Equal(t, pulid.ID("****"), result)

	result = sdm.maskPulID(pulid.ID("longidwithoutunderscore"))
	assert.Equal(t, pulid.ID("****"), result)
}

func TestApplyMaskStrategy_EmptyString(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyMaskStrategy("")
	assert.Equal(t, "", result)
}

func TestApplyMaskStrategy_ShortString_Default(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyDefault)

	result := sdm.applyMaskStrategy("abcd")
	assert.Equal(t, "****", result)
}

func TestApplyMaskStrategy_Partial_MediumString(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyPartial)

	result := sdm.applyMaskStrategy("hello1")
	strResult := result.(string)
	assert.Equal(t, "he", strResult[:2])
	assert.Equal(t, "1", strResult[len(strResult)-1:])
}

func TestApplyMaskStrategy_Partial_ShortString(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyPartial)

	result := sdm.applyMaskStrategy("abc")
	assert.Equal(t, "****", result)
}

func TestApplyMaskStrategy_Bool(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyMaskStrategy(true)
	assert.Equal(t, true, result)

	result = sdm.applyMaskStrategy(false)
	assert.Equal(t, false, result)
}

func TestApplyMaskStrategy_Numeric_Default(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyDefault)

	result := sdm.applyMaskStrategy(12345)
	assert.Equal(t, "****", result)
}

func TestApplyMaskStrategy_Numeric_Partial(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetMaskStrategy(MaskStrategyPartial)

	result := sdm.applyMaskStrategy(float64(99.99))
	assert.Equal(t, "****", result)
}

func TestApplyMaskStrategy_Map(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyMaskStrategy(map[string]any{"key": "val"})
	assert.Equal(t, "[REDACTED]", result)
}

func TestApplyMaskStrategy_OtherTypes(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyMaskStrategy([]string{"a", "b"})
	assert.Equal(t, "****", result)
}

func TestMaskValue_Omit(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.maskValue("secret", services.SensitiveFieldOmit)
	assert.Nil(t, result)
}

func TestMaskValue_Hash(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.maskValue("secret", services.SensitiveFieldHash)
	str := result.(string)
	assert.Contains(t, str, "SHA256:")
}

func TestMaskValue_Encrypt(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.maskValue("secret", services.SensitiveFieldEncrypt)
	str := result.(string)
	assert.Contains(t, str, "ENC:")
}

func TestMaskValue_UnknownAction(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.maskValue("secret", services.SensitiveFieldAction(99))
	assert.Equal(t, "secret", result)
}

func TestApplyHashStrategy_EmptyString(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyHashStrategy("")
	assert.Equal(t, "", result)
}

func TestApplyHashStrategy_Int(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyHashStrategy(42)
	str := result.(string)
	assert.Contains(t, str, "SHA256:")
}

func TestApplyHashStrategy_Float(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyHashStrategy(3.14)
	str := result.(string)
	assert.Contains(t, str, "SHA256:")
}

func TestApplyHashStrategy_Bool(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyHashStrategy(true)
	str := result.(string)
	assert.Contains(t, str, "SHA256:")
}

func TestApplyHashStrategy_Map(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyHashStrategy(map[string]any{"key": "val"})
	str := result.(string)
	assert.Contains(t, str, "SHA256:")
}

func TestApplyHashStrategy_Slice(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyHashStrategy([]any{1, 2, 3})
	str := result.(string)
	assert.Contains(t, str, "SHA256:")
}

func TestApplyHashStrategy_OtherType(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	type custom struct{ X int }
	result := sdm.applyHashStrategy(custom{X: 5})
	str := result.(string)
	assert.Contains(t, str, "SHA256:")
}

func TestApplyEncryptStrategy_EmptyString(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy("")
	assert.Equal(t, "", result)
}

func TestApplyEncryptStrategy_NoKey_FallbackToMask(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: ""})
	sdm.SetMaskStrategy(MaskStrategyStrict)

	result := sdm.applyEncryptStrategy("secret")
	assert.Equal(t, "****", result)
}

func TestApplyEncryptStrategy_Int(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy(42)
	str := result.(string)
	assert.Contains(t, str, "ENC:")
}

func TestApplyEncryptStrategy_Bool(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy(true)
	str := result.(string)
	assert.Contains(t, str, "ENC:")
}

func TestApplyEncryptStrategy_Map(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy(map[string]any{"k": "v"})
	str := result.(string)
	assert.Contains(t, str, "ENC:")
}

func TestApplyEncryptStrategy_Slice(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	result := sdm.applyEncryptStrategy([]any{1, 2})
	str := result.(string)
	assert.Contains(t, str, "ENC:")
}

func TestApplyEncryptStrategy_OtherType(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	type custom struct{ X int }
	result := sdm.applyEncryptStrategy(custom{X: 5})
	str := result.(string)
	assert.Contains(t, str, "ENC:")
}

func TestContainsSensitivePattern_Phone(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	assert.True(t, sdm.containsSensitivePattern("(555) 123-4567"))
}

func TestContainsSensitivePattern_LongToken(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	assert.True(t, sdm.containsSensitivePattern("abcdefghijklmnopqrstuvwxyz"))
}

func TestContainsSensitivePattern_SafeValue(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	assert.False(t, sdm.containsSensitivePattern("hello world"))
}

func TestIsSensitiveFieldPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	assert.True(t, sdm.isSensitiveFieldPath("user.password"))
	assert.True(t, sdm.isSensitiveFieldPath("configuration.apiKey"))
	assert.False(t, sdm.isSensitiveFieldPath("user.name"))
}

func TestShouldMaskValue_String(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	assert.True(t, sdm.shouldMaskValue("123-45-6789"))
	assert.False(t, sdm.shouldMaskValue("hello"))
}

func TestShouldMaskValue_Map(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	assert.True(t, sdm.shouldMaskValue(map[string]any{"password": "secret"}))
	assert.True(t, sdm.shouldMaskValue(map[string]any{"data": "123-45-6789"}))
	assert.False(t, sdm.shouldMaskValue(map[string]any{"name": "John"}))
}

func TestShouldMaskValue_OtherType(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	assert.False(t, sdm.shouldMaskValue(42))
	assert.False(t, sdm.shouldMaskValue(true))
}

func TestAddCustomPattern(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	err := sdm.AddCustomPattern("custom", `\bCUSTOM-\d+\b`)
	require.NoError(t, err)
}

func TestAddCustomPattern_InvalidRegex(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	err := sdm.AddCustomPattern("bad", `[invalid`)
	require.Error(t, err)
}

func TestRemoveCustomPattern(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	_ = sdm.AddCustomPattern("custom", `\bCUSTOM-\d+\b`)
	sdm.RemoveCustomPattern("custom")
}

func TestClearCache(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.containsSensitivePattern("123-45-6789")

	sdm.ClearCache()
}

func TestRegisterSensitiveFields_EmptyFields(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	err := sdm.RegisterSensitiveFields(permission.Resource("test"), nil)
	require.NoError(t, err)
}

func TestRegisterSensitiveFields_NoPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	err := sdm.RegisterSensitiveFields(permission.Resource("test"), []services.SensitiveField{
		{Name: "token", Path: "", Action: services.SensitiveFieldOmit},
	})
	require.NoError(t, err)

	stored, ok := sdm.fields.Load(permission.Resource("test"))
	require.True(t, ok)
	storedMap := stored.(map[string]SensitiveFieldConfig)
	assert.Contains(t, storedMap, "token")
}

func TestRegisterSensitiveFields_MergesExisting(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})

	err := sdm.RegisterSensitiveFields(permission.Resource("test"), []services.SensitiveField{
		{Name: "field1", Action: services.SensitiveFieldOmit},
	})
	require.NoError(t, err)

	err = sdm.RegisterSensitiveFields(permission.Resource("test"), []services.SensitiveField{
		{Name: "field2", Action: services.SensitiveFieldMask},
	})
	require.NoError(t, err)

	stored, ok := sdm.fields.Load(permission.Resource("test"))
	require.True(t, ok)
	storedMap := stored.(map[string]SensitiveFieldConfig)
	assert.Contains(t, storedMap, "field1")
	assert.Contains(t, storedMap, "field2")
}

func TestSanitizeChangesMap_AutoDetectSensitiveValue(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	changes := map[string]any{
		"notes": map[string]any{
			"from": "My SSN is 123-45-6789",
			"to":   "Updated",
		},
	}

	sdm.sanitizeChangesMap(changes, nil)

	notesChange := changes["notes"].(map[string]any)
	assert.Equal(t, "****", notesChange["from"])
	assert.Equal(t, "Updated", notesChange["to"])
}

func TestSanitizeChangesMap_AutoDetectFieldPath(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	changes := map[string]any{
		"password": map[string]any{
			"from": "old-pass",
			"to":   "new-pass",
		},
	}

	sdm.sanitizeChangesMap(changes, nil)

	pwChange := changes["password"].(map[string]any)
	assert.Equal(t, "****", pwChange["from"])
	assert.Equal(t, "****", pwChange["to"])
}

func TestSanitizeChangesMap_NilValues(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(false)

	registered := map[string]SensitiveFieldConfig{
		"secret": {Name: "secret", Action: services.SensitiveFieldMask},
	}

	changes := map[string]any{
		"secret": map[string]any{
			"from": nil,
			"to":   nil,
		},
	}

	sdm.sanitizeChangesMap(changes, registered)

	secretChange := changes["secret"].(map[string]any)
	assert.Nil(t, secretChange["from"])
	assert.Nil(t, secretChange["to"])
}

func TestNewSensitiveDataManager_NoEncryptionKey(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: ""})

	require.NotNil(t, sdm)
	assert.Nil(t, sdm.encryptionKey)
}

func TestSanitizeEntry_NestedSlice(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-key"})
	sdm.SetAutoDetect(true)
	sdm.SetMaskStrategy(MaskStrategyStrict)

	entry := &audit.Entry{
		Resource: permission.Resource("test"),
		CurrentState: map[string]any{
			"nested": []any{
				[]any{"123-45-6789", "safe"},
			},
		},
	}

	err := sdm.SanitizeEntry(entry)
	require.NoError(t, err)

	nested := entry.CurrentState["nested"].([]any)
	inner := nested[0].([]any)
	assert.Equal(t, "****", inner[0])
	assert.Equal(t, "safe", inner[1])
}
