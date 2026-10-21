package secretconfig

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type Scope struct {
	Purpose      encryptionservice.Purpose
	Tenant       pagination.TenantInfo
	ResourceKind string
	Subject      string
	Bind         bool
}

func (sc Scope) aad(fieldKey string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        sc.Purpose,
		OrganizationID: sc.Tenant.OrgID,
		BusinessUnitID: sc.Tenant.BuID,
		ResourceID:     sc.ResourceKind + ":" + sc.Subject + ":" + fieldKey,
	}
}

type Codec struct {
	encryption *encryptionservice.Service
	l          *zap.Logger
}

func NewCodec(encryption *encryptionservice.Service, logger *zap.Logger) Codec {
	return Codec{encryption: encryption, l: logger}
}

func (c Codec) Encrypt(value, fieldKey string, scope Scope) (string, error) {
	if !scope.Bind {
		return c.encryption.EncryptString(value)
	}

	return c.encryption.EncryptStringWithAAD(value, scope.aad(fieldKey))
}

func (c Codec) Decrypt(value, fieldKey string, scope Scope) (string, error) {
	if !scope.Bind {
		return c.encryption.DecryptString(value)
	}

	decrypted, _, err := c.encryption.DecryptStringWithLegacy(value, scope.aad(fieldKey))
	return decrypted, err
}

func (c Codec) Rebind(stored, fieldKey string, scope Scope) string {
	if !scope.Bind || stored == "" {
		return stored
	}

	_, rebound, err := c.encryption.DecryptAndRebind(stored, scope.aad(fieldKey))
	if err != nil {
		c.l.Warn("failed to bind secret to tenant", zap.Error(err),
			zap.String("resource", scope.ResourceKind), zap.String("subject", scope.Subject))
		return stored
	}
	if rebound == "" {
		return stored
	}

	return rebound
}

func (c Codec) Merge(
	fields []configspec.Field,
	incoming map[string]string,
	existing map[string]any,
	scope Scope,
) (map[string]any, error) {
	merged := make(map[string]any, len(fields))
	for idx := range fields {
		field := &fields[idx]
		value := strings.TrimSpace(incoming[field.Key])

		if !field.Sensitive {
			if value == "" && field.Default != "" {
				value = field.Default
			}
			merged[field.Key] = value
			continue
		}

		if value == "" {
			merged[field.Key] = c.Rebind(
				configspec.ReadString(existing, field.Key),
				field.Key,
				scope,
			)
			continue
		}

		encrypted, err := c.Encrypt(value, field.Key, scope)
		if err != nil {
			return nil, errortypes.NewBusinessError(
				"failed to encrypt configuration value",
			).WithInternal(err)
		}
		merged[field.Key] = encrypted
	}

	return merged, nil
}

func (c Codec) ReadField(
	configuration map[string]any,
	field *configspec.Field,
	scope Scope,
) (string, error) {
	if field == nil {
		return "", nil
	}

	value := configspec.ReadString(configuration, field.Key)
	if value == "" || !field.Sensitive {
		return value, nil
	}

	return c.Decrypt(value, field.Key, scope)
}

func (c Codec) ReadAll(
	configuration map[string]any,
	fields []configspec.Field,
	scope Scope,
) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for idx := range fields {
		value, err := c.ReadField(configuration, &fields[idx], scope)
		if err != nil {
			return nil, err
		}
		values[fields[idx].Key] = value
	}

	return values, nil
}
