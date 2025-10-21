package variable

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/variable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ResolverParams struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.VariableRepository
}

type ResolverService struct {
	l    *zap.Logger
	repo repositories.VariableRepository
}

func NewResolverService(p ResolverParams) services.VariableResolverService {
	return &ResolverService{
		l:    p.Logger.Named("service.variable-resolver"),
		repo: p.Repo,
	}
}

var variablePattern = regexp.MustCompile(`\{\{(\w+)\}\}`)

func (s *ResolverService) ProcessTemplate( //nolint:funlen // this is 1 over the 50 line limit
	ctx context.Context,
	template string,
	vCtx variable.Context,
	contextID pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
) (string, error) {
	log := s.l.With(
		zap.String("operation", "ProcessTemplate"),
		zap.String("context", vCtx.String()),
		zap.String("contextID", contextID.String()),
		zap.String("orgID", orgID.String()),
	)

	if template == "" {
		return "", nil
	}

	matches := variablePattern.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return template, nil
	}

	keyMap := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			keyMap[match[1]] = true
		}
	}

	keys := make([]string, 0, len(keyMap))
	for key := range keyMap {
		keys = append(keys, key)
	}

	log.Debug("extracted variable keys from template",
		zap.Strings("keys", keys),
		zap.Int("count", len(keys)),
	)

	variables, err := s.repo.GetActiveVariablesByKeys(ctx, repositories.GetVariablesByKeysRequest{
		OrgID: orgID,
		Keys:  keys,
	})
	if err != nil {
		log.Error("failed to fetch variables", zap.Error(err))
		return template, nil
	}

	params := map[string]any{
		"contextId": contextID.String(),
		"orgId":     orgID.String(),
		"buId":      buID.String(),
	}

	switch vCtx {
	case variable.ContextCustomer:
		params["customerId"] = contextID.String()
	case variable.ContextInvoice:
		params["invoiceId"] = contextID.String()
	case variable.ContextShipment:
		params["shipmentId"] = contextID.String()
	case variable.ContextOrganization:
		params["organizationId"] = contextID.String()
	case variable.ContextSystem:
		// System variables don't need a specific context ID
		// They typically use orgId for organization-wide settings
	}

	result := template
	for _, v := range variables {
		if v.AppliesTo != vCtx && v.AppliesTo != variable.ContextSystem {
			log.Debug("skipping variable - wrong context",
				zap.String("variableKey", v.Key),
				zap.String("variableContext", v.AppliesTo.String()),
				zap.String("requestedContext", vCtx.String()),
			)
			continue
		}

		// Resolve the variable value
		value, vErr := s.repo.ResolveVariable(ctx, repositories.ResolveVariableRequest{
			Variable: v,
			Params:   params,
		})
		if vErr != nil {
			log.Error("failed to resolve variable",
				zap.String("variableKey", v.Key),
				zap.Error(vErr),
			)
			value = v.DefaultValue
		}

		if v.Format != nil {
			value = s.applyFormat(ctx, value, v.Format)
		}

		placeholder := fmt.Sprintf("{{%s}}", v.Key)
		result = strings.ReplaceAll(result, placeholder, value)

		log.Debug("replaced variable in template",
			zap.String("variableKey", v.Key),
			zap.String("value", value),
		)
	}

	remainingMatches := variablePattern.FindAllStringSubmatch(result, -1)
	if len(remainingMatches) > 0 {
		unresolvedKeys := make([]string, 0, len(remainingMatches))
		for _, match := range remainingMatches {
			if len(match) > 1 {
				unresolvedKeys = append(unresolvedKeys, match[1])
			}
		}
		log.Warn("unresolved variables remain in template",
			zap.Strings("unresolvedKeys", unresolvedKeys),
		)
	}

	return result, nil
}

func (s *ResolverService) ResolveVariable(
	ctx context.Context,
	v *variable.Variable,
	params map[string]any,
) (string, error) {
	log := s.l.With(
		zap.String("operation", "ResolveVariable"),
		zap.String("variableKey", v.Key),
	)

	result, err := s.repo.ResolveVariable(ctx, repositories.ResolveVariableRequest{
		Variable: v,
		Params:   params,
	})
	if err != nil {
		log.Error("failed to resolve variable", zap.Error(err))
		return v.DefaultValue, err
	}

	if v.Format != nil {
		result = s.applyFormat(ctx, result, v.Format)
	}

	return result, nil
}

func (s *ResolverService) ResolveVariables(
	ctx context.Context,
	variables []*variable.Variable,
	params map[string]any,
) (map[string]string, error) {
	log := s.l.With(
		zap.String("operation", "ResolveVariables"),
		zap.Int("variableCount", len(variables)),
	)

	results := make(map[string]string, len(variables))

	for _, v := range variables {
		value, err := s.ResolveVariable(ctx, v, params)
		if err != nil {
			log.Error("failed to resolve variable",
				zap.String("variableKey", v.Key),
				zap.Error(err),
			)
			value = v.DefaultValue
		}
		results[v.Key] = value
	}

	return results, nil
}

func (s *ResolverService) applyFormat(
	ctx context.Context,
	value string,
	format *variable.VariableFormat,
) string {
	if value == "" || format == nil {
		return value
	}

	formattedValue, err := s.repo.ExecuteFormatSQL(ctx, repositories.ExecuteFormatSQLRequest{
		FormatSQL: format.FormatSQL,
		Value:     value,
	})
	if err != nil {
		s.l.Error("failed to apply format",
			zap.String("formatName", format.Name),
			zap.String("value", value),
			zap.Error(err),
		)
		return value
	}

	return formattedValue
}
