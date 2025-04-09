package permission

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/samber/lo"
)

func evaluateFieldPermission(perm *permission.Permission, field string, ctx *services.PermissionContext) services.FieldPermissionCheck {
	if field == "" {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   eris.New("field name cannot be empty"),
		}
	}

	fp, found := findFieldPermission(perm.FieldPermissions, field)
	if !found {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   fmt.Errorf("no permissions defined for field: %s", field),
		}
	}

	if !hasAction(fp.Actions, permission.ActionModifyField) {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   fmt.Errorf("modify action not permitted for field: %s", field),
		}
	}

	if !evaluateConditions(fp.Conditions, ctx) {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   fmt.Errorf("condition check failed for field: %s", field),
		}
	}

	return services.FieldPermissionCheck{
		Allowed: true,
		Error:   nil,
	}
}

func evaluateFieldViewPermission(perm *permission.Permission, field string, ctx *services.PermissionContext) services.FieldPermissionCheck {
	fp, found := findFieldPermission(perm.FieldPermissions, field)
	if !found {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   fmt.Errorf("no permissions defined for field: %s", field),
		}
	}

	if !hasAction(fp.Actions, permission.ActionViewField) {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   fmt.Errorf("view action not permitted for field: %s", field),
		}
	}

	if !evaluateConditions(fp.Conditions, ctx) {
		return services.FieldPermissionCheck{
			Allowed: false,
			Error:   fmt.Errorf("condition check failed for field: %s", field),
		}
	}

	return services.FieldPermissionCheck{
		Allowed: true,
		Error:   nil,
	}
}

func getAllResourceActions(resource permission.Resource) []permission.Action {
	if actions, exists := permission.ResourceActionMap[resource]; exists {
		return actions
	}

	return permission.BaseActions
}

func supportsAction(resource permission.Resource, action permission.Action) bool {
	actions := getAllResourceActions(resource)
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func findFieldPermission(permissions []*permission.FieldPermission, field string) (*permission.FieldPermission, bool) {
	for i := range permissions {
		if permissions[i].Field == field {
			return permissions[i], true
		}
	}
	return nil, false
}

func hasAction(actions []permission.Action, target permission.Action) bool {
	for _, action := range actions {
		if action == target {
			return true
		}
	}
	return false
}

func evaluateConditions(conditions []*permission.Condition, ctx *services.PermissionContext) bool {
	if len(conditions) == 0 {
		return true
	}

	sorted := sortConditionsByPriority(conditions)

	for _, condition := range sorted {
		if !evaluateCondition(condition, ctx) {
			return false
		}
	}

	return true
}

func checkScope(scope permission.Scope, check *services.PermissionCheck) bool {
	switch scope {
	case permission.ScopeGlobal:
		return true
	case permission.ScopeBU:
		return check.BusinessUnitID != pulid.Nil
	case permission.ScopeOrg:
		return check.OrganizationID != pulid.Nil
	case permission.ScopePersonal:
		return check.ResourceID != pulid.Nil && check.ResourceID == check.UserID
	default:
		return false
	}
}

func sortConditionsByPriority(conditions []*permission.Condition) []*permission.Condition {
	if len(conditions) == 0 {
		return conditions
	}

	sorted := make([]*permission.Condition, len(conditions))
	copy(sorted, conditions)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	return sorted
}

func canModifyField(field string, ctx *services.PermissionContext, fieldPerms []*permission.FieldPermission) bool {
	fp, found := findFieldPermission(fieldPerms, field)
	if !found {
		return false
	}

	if !hasModifyAction(fp.Actions) {
		return false
	}

	return evaluateFieldConditions(fp.Conditions, ctx)
}

func hasModifyAction(actions []permission.Action) bool {
	return hasAction(actions, permission.ActionModifyField)
}

func evaluateFieldConditions(conditions []*permission.Condition, ctx *services.PermissionContext) bool {
	if len(conditions) == 0 {
		return true
	}

	sorted := sortConditionsByPriority(conditions)
	for _, condition := range sorted {
		if !evaluateCondition(condition, ctx) {
			return false
		}
	}

	return true
}

func evaluateCondition(c *permission.Condition, ctx *services.PermissionContext) bool {
	switch c.Type {
	case permission.ConditionTypeField:
		return evaluateFieldCondition(c, ctx)
	case permission.ConditionTypeTime:
		return evaluateTimeCondition(c, ctx)
	case permission.ConditionTypeRole:
		return evaluateRoleCondition(c, ctx)
	case permission.ConditionTypeOwnership:
		return evaluateOwnershipCondition(c, ctx)
	case permission.ConditionTypeCustom:
		// TODO(Wolfred): Implement custom condition evaluation
		return false
	default:
		return false
	}
}

func evaluateFieldCondition(c *permission.Condition, ctx *services.PermissionContext) bool {
	fieldValue, exists := ctx.CustomData[c.Field]
	if !exists {
		return false
	}

	op := permission.Operator(c.Operator)
	switch op {
	case permission.OpEquals:
		return fieldValue == c.Value
	case permission.OpNotEquals:
		return fieldValue != c.Value
	case permission.OpIn:
		for _, v := range c.Values {
			if v == fieldValue {
				return true
			}
		}
		return false
	case permission.OpNotIn:
		for _, v := range c.Values {
			if v == fieldValue {
				return false
			}
		}
		return true
	case permission.OpContains:
		strField, ok := fieldValue.(string)
		if !ok {
			return false
		}
		strValue, ok := c.Value.(string)
		if !ok {
			return false
		}
		return strings.Contains(strField, strValue)
	case permission.OpGreaterThan, permission.OpLessThan, permission.OpNotContains:
		// ! Do not support these operators for fields
		return false
	default:
		return false
	}
}

func evaluateTimeCondition(c *permission.Condition, ctx *services.PermissionContext) bool {
	op := permission.Operator(c.Operator)
	now := ctx.Time

	targetTime, err := timeutils.ParseTimeValue(c.Value)
	if err != nil {
		return false
	}

	switch op {
	case permission.OpEquals:
		return now.Equal(targetTime)
	case permission.OpGreaterThan:
		return now.After(targetTime)
	case permission.OpLessThan:
		return now.Before(targetTime)
	case permission.OpIn:
		// For time ranges
		if len(c.Values) != 2 {
			return false
		}
		start, startErr := timeutils.ParseTimeValue(c.Values[0])
		if startErr != nil {
			return false
		}
		end, endErr := timeutils.ParseTimeValue(c.Values[1])
		if endErr != nil {
			return false
		}
		return now.After(start) && now.Before(end)
	// ! Do not support these operators for time
	case permission.OpNotEquals, permission.OpNotIn, permission.OpContains, permission.OpNotContains:
		return false
	default:
		return false
	}
}

func evaluateRoleCondition(c *permission.Condition, ctx *services.PermissionContext) bool {
	op := permission.Operator(c.Operator)

	switch op {
	case permission.OpEquals:
		role, ok := c.Value.(string)
		if !ok {
			return false
		}
		return lo.Contains(ctx.Roles, &role)
	case permission.OpIn:
		for _, reqRole := range c.Values {
			role, ok := reqRole.(string)
			if !ok {
				continue
			}
			if lo.Contains(ctx.Roles, &role) {
				return true
			}
		}
		return false
	case permission.OpNotIn:
		for _, reqRole := range c.Values {
			role, ok := reqRole.(string)
			if !ok {
				continue
			}
			if lo.Contains(ctx.Roles, &role) {
				return false
			}
		}
		return true
	// ! Do not support these operators for roles
	case permission.OpNotEquals, permission.OpGreaterThan, permission.OpLessThan, permission.OpContains, permission.OpNotContains:
		return false
	default:
		return false
	}
}

func evaluateOwnershipCondition(c *permission.Condition, ctx *services.PermissionContext) bool {
	op := permission.Operator(c.Operator)

	switch op {
	case permission.OpEquals:
		resourceOwner, ok := c.Value.(string)
		if !ok {
			return false
		}
		return resourceOwner == ctx.UserID.String()

	case permission.OpIn:
		switch c.Field {
		case "business_unit_id":
			strValue, ok := c.Value.(string)
			if !ok {
				return false
			}
			return ctx.BuID.String() == strValue
		case "organization_id":
			strValue, ok := c.Value.(string)
			if !ok {
				return false
			}
			return ctx.OrgID.String() == strValue
		default:
			return false
		}
	case permission.OpContains, permission.OpNotEquals, permission.OpNotContains, permission.OpGreaterThan, permission.OpLessThan, permission.OpNotIn:
		return false

	default:
		return false
	}
}
