package compiler

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type authzResult struct {
	scopes map[string]permission.DataScope
}

func (c *Compiler) authorize(
	ctx context.Context,
	req *services.ReportCompileRequest,
	v *validatedDef,
	requireExport bool,
) (*authzResult, error) {
	result := &authzResult{
		scopes: make(map[string]permission.DataScope, len(v.entityKeys)),
	}

	var denied []string

	for _, entityKey := range v.entityKeys {
		entity, ok := c.catalog.Entity(entityKey)
		if !ok {
			return nil, fmt.Errorf("catalog entity %q vanished during compilation", entityKey)
		}

		detail, err := c.permissionEngine.GetResourcePermissions(
			ctx, req.Tenant.UserID, req.Tenant.OrgID, entity.Resource.String(),
		)
		if err != nil {
			return nil, fmt.Errorf("resolve permissions for %q: %w", entity.Resource, err)
		}

		if !hasOperation(detail, permission.OpRead) {
			denied = append(denied, fmt.Sprintf(
				"you do not have read access to %s", entity.PluralLabel,
			))
			continue
		}
		if requireExport && !hasOperation(detail, permission.OpExport) {
			denied = append(denied, fmt.Sprintf(
				"you do not have export access to %s", entity.PluralLabel,
			))
			continue
		}

		result.scopes[entityKey] = detail.DataScope

		for _, ref := range v.refs {
			if ref.entity.Key != entityKey {
				continue
			}
			if fieldErr := c.checkFieldAccess(detail, ref); fieldErr != "" {
				denied = append(denied, fieldErr)
			}
		}
	}

	if len(denied) > 0 {
		return nil, errortypes.NewAuthorizationError(
			"This report references data you do not have access to: " +
				strings.Join(dedupeStrings(denied), "; "),
		)
	}

	return result, nil
}

func (c *Compiler) checkFieldAccess(
	detail *services.ResourcePermissionDetail,
	ref *resolvedRef,
) string {
	sensitivity := c.permissionRegistry.GetFieldSensitivity(
		ref.entity.Resource.String(), ref.field.Key,
	)
	if !detail.MaxSensitivity.CanAccess(sensitivity) {
		return fmt.Sprintf(
			"field %q on %s requires %s access",
			ref.field.Label, ref.entity.Label, sensitivity,
		)
	}

	if len(detail.AccessibleFields) > 0 && !containsField(detail.AccessibleFields, ref.field.Key) {
		return fmt.Sprintf(
			"field %q on %s is not in your accessible field set",
			ref.field.Label, ref.entity.Label,
		)
	}

	return ""
}

func hasOperation(detail *services.ResourcePermissionDetail, op permission.Operation) bool {
	for _, allowed := range detail.Operations {
		if allowed == op {
			return true
		}
	}
	return false
}

func containsField(fields []string, field string) bool {
	for _, f := range fields {
		if f == field {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
