package auditservice

import (
	"strings"
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

func TestNewSensitiveDataManager(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	require.NotNil(t, sdm)
	assert.True(t, sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyDefault), sdm.strategy.Load())
}

func TestRegisterSensitiveFields(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	fields := []services.SensitiveField{
		{Name: "ssn", Path: "user", Action: services.SensitiveFieldOmit},
		{Name: "email", Path: "user", Action: services.SensitiveFieldMask},
	}

	err := sdm.RegisterSensitiveFields(permission.Resource("test_resource"), fields)
	require.NoError(t, err)

	stored, ok := sdm.fields.Load(permission.Resource("test_resource"))
	require.True(t, ok)

	storedMap, ok := stored.(map[string]SensitiveFieldConfig)
	require.True(t, ok)
	assert.Len(t, storedMap, 2)
	assert.Contains(t, storedMap, "user.ssn")
	assert.Contains(t, storedMap, "user.email")
}

func TestApplyMaskStrategy_Strict(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})
	sdm.SetMaskStrategy(MaskStrategyStrict)

	stringResult := sdm.applyMaskStrategy("sensitive-data")
	assert.Equal(t, "****", stringResult)

	intResult := sdm.applyMaskStrategy(12345)
	assert.Equal(t, 0, intResult)
}

func TestApplyMaskStrategy_Default(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})
	sdm.SetMaskStrategy(MaskStrategyDefault)

	result := sdm.applyMaskStrategy("hello world")
	strResult, ok := result.(string)
	require.True(t, ok)
	assert.Equal(t, "h", strResult[:1])
	assert.Contains(t, strResult, "****")
	assert.Equal(t, "d", strResult[len(strResult)-1:])
}

func TestApplyMaskStrategy_Partial(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})
	sdm.SetMaskStrategy(MaskStrategyPartial)

	result := sdm.applyMaskStrategy("sensitive-data")
	strResult, ok := result.(string)
	require.True(t, ok)
	assert.Equal(t, "sen", strResult[:3])
	assert.Equal(t, "ata", strResult[len(strResult)-3:])
}

func TestSanitizeEntry_WithRegisteredFields(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})
	sdm.SetAutoDetect(false)

	fields := []services.SensitiveField{
		{Name: "secret", Path: "", Action: services.SensitiveFieldOmit},
	}

	err := sdm.RegisterSensitiveFields(permission.Resource("test_resource"), fields)
	require.NoError(t, err)

	entry := &audit.Entry{
		Resource: permission.Resource("test_resource"),
		CurrentState: map[string]any{
			"secret": "my-secret-value",
			"name":   "visible",
		},
	}

	err = sdm.SanitizeEntry(entry)
	require.NoError(t, err)
	assert.Nil(t, entry.CurrentState["secret"])
	assert.Equal(t, "visible", entry.CurrentState["name"])
}

func TestSanitizeEntry_NilEntry(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	err := sdm.SanitizeEntry(nil)
	assert.NoError(t, err)
}

func TestContainsSensitivePattern_SSN(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.True(t, sdm.containsSensitivePattern("123-45-6789"))
}

func TestContainsSensitivePattern_Email(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.True(t, sdm.containsSensitivePattern("test@example.com"))
}

func TestContainsSensitivePattern_CreditCard(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.True(t, sdm.containsSensitivePattern("4111111111111111"))
}

func TestIsSensitiveFieldName_Password(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.True(t, sdm.isSensitiveFieldName("password"))
}

func TestIsSensitiveFieldName_ApiKey(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.True(t, sdm.isSensitiveFieldName("api_key"))
}

func TestIsSensitiveFieldName_Regular(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.False(t, sdm.isSensitiveFieldName("name"))
}

func TestSanitizeUserObject_Strict(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})
	sdm.SetMaskStrategy(MaskStrategyStrict)

	user := &tenant.User{
		ID:                    pulid.ID("usr_01ABCDEFGH"),
		BusinessUnitID:        pulid.ID("bu_01ABCDEFGH"),
		CurrentOrganizationID: pulid.ID("org_01ABCDEFGH"),
		EmailAddress:          "user@example.com",
	}

	sdm.sanitizeUserObject(user)

	assert.Equal(t, "****@example.com", user.EmailAddress)
	assert.Contains(t, string(user.ID), "*")
	assert.Contains(t, string(user.BusinessUnitID), "*")
	assert.Contains(t, string(user.CurrentOrganizationID), "*")
}

func TestSanitizeUserObject_Partial(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})
	sdm.SetMaskStrategy(MaskStrategyPartial)

	user := &tenant.User{
		EmailAddress: "user@example.com",
	}

	sdm.sanitizeUserObject(user)

	assert.Contains(t, user.EmailAddress, "us")
	assert.Contains(t, user.EmailAddress, "@example.com")
}

func TestApplyHashStrategy(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	result := sdm.applyHashStrategy("some-value")
	strResult, ok := result.(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(strResult, "SHA256:"))
}

func TestApplyEncryptStrategy(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	result := sdm.applyEncryptStrategy("some-value")
	strResult, ok := result.(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(strResult, "ENC:"))
}

func TestSetAutoDetect(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.True(t, sdm.autoDetect.Load())

	sdm.SetAutoDetect(false)
	assert.False(t, sdm.autoDetect.Load())

	sdm.SetAutoDetect(true)
	assert.True(t, sdm.autoDetect.Load())
}

func TestSetMaskStrategy(t *testing.T) {
	t.Parallel()

	sdm := NewSensitiveDataManager(config.EncryptionConfig{Key: "test-encryption-key-for-testing"})

	assert.Equal(t, int32(MaskStrategyDefault), sdm.strategy.Load())

	sdm.SetMaskStrategy(MaskStrategyStrict)
	assert.Equal(t, int32(MaskStrategyStrict), sdm.strategy.Load())

	sdm.SetMaskStrategy(MaskStrategyPartial)
	assert.Equal(t, int32(MaskStrategyPartial), sdm.strategy.Load())
}
