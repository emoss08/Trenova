package locationcodegenerator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

var errLocationCodeRequestRequired = errors.New("location code request is required")

var fixedLocationCodePeriod = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

type Params struct {
	fx.In

	SequenceConfigRepo repositories.SequenceConfigRepository
	SequenceGenerator  seqgen.Generator
}

type Generator struct {
	sequenceConfigRepo repositories.SequenceConfigRepository
	sequenceGenerator  seqgen.Generator
}

func New(p Params) services.LocationCodeGenerator {
	return &Generator{
		sequenceConfigRepo: p.SequenceConfigRepo,
		sequenceGenerator:  p.SequenceGenerator,
	}
}

func (g *Generator) Generate(
	ctx context.Context,
	req services.LocationCodeGenerateRequest,
) (string, error) {
	if req.OrganizationID.IsNil() || req.BusinessUnitID.IsNil() {
		return "", errLocationCodeRequestRequired
	}

	strategy, err := g.resolveStrategy(ctx, req)
	if err != nil {
		return "", err
	}

	prefix, err := g.BuildPrefix(req.Input, strategy)
	if err != nil {
		return "", err
	}

	format := &tenant.SequenceFormat{
		Type:           tenant.SequenceTypeLocationCode,
		Prefix:         prefix,
		SequenceDigits: int(strategy.SequenceDigits),
		UseSeparators:  strategy.Separator != "",
		SeparatorChar:  strategy.Separator,
	}

	code, err := g.sequenceGenerator.Generate(ctx, &seqgen.GenerateRequest{
		Type:   tenant.SequenceTypeLocationCode,
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		Time:   fixedLocationCodePeriod,
		Format: format,
	})
	if err != nil {
		return "", fmt.Errorf("generate location code sequence: %w", err)
	}
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("generated location code cannot be empty")
	}
	if len([]rune(code)) > tenant.MaxLocationCodeLength {
		return "", fmt.Errorf(
			"generated location code cannot exceed %d characters",
			tenant.MaxLocationCodeLength,
		)
	}

	return code, nil
}

func (g *Generator) BuildPrefix(
	input services.LocationCodeInput,
	strategy *tenant.LocationCodeStrategy,
) (string, error) {
	strategy = tenant.EffectiveLocationCodeStrategy(strategy)
	if err := strategy.Validate(); err != nil {
		return "", err
	}

	fallback := normalizedToken(strategy.FallbackPrefix, strategy.Casing)
	if fallback == "" {
		return "", fmt.Errorf("fallback prefix must contain letters or digits")
	}

	parts := make([]string, 0, len(strategy.Components))
	lastWasFallback := false
	for _, component := range strategy.Components {
		token := componentValue(input, component)
		normalized := normalizedToken(token, strategy.Casing)
		usedFallback := false
		if normalized == "" {
			normalized = fallback
			usedFallback = true
		}
		if usedFallback && lastWasFallback {
			continue
		}

		parts = append(parts, stringutils.TruncateRunes(normalized, int(strategy.ComponentWidth)))
		lastWasFallback = usedFallback
	}
	if len(parts) == 0 {
		parts = append(parts, stringutils.TruncateRunes(fallback, int(strategy.ComponentWidth)))
	}

	return strings.Join(parts, strategy.Separator), nil
}

func (g *Generator) resolveStrategy(
	ctx context.Context,
	req services.LocationCodeGenerateRequest,
) (*tenant.LocationCodeStrategy, error) {
	doc, err := g.sequenceConfigRepo.GetByTenant(ctx, repositories.GetSequenceConfigRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get sequence config: %w", err)
	}

	for _, cfg := range doc.Configs {
		if cfg != nil && cfg.SequenceType == tenant.SequenceTypeLocationCode {
			strategy := tenant.EffectiveLocationCodeStrategy(cfg.LocationCodeStrategy)
			if err = strategy.Validate(); err != nil {
				return nil, fmt.Errorf("invalid location code strategy: %w", err)
			}

			return strategy, nil
		}
	}

	return tenant.DefaultLocationCodeStrategy(), nil
}

func componentValue(
	input services.LocationCodeInput,
	component tenant.LocationCodeComponent,
) string {
	switch component {
	case tenant.LocationCodeComponentName:
		return input.Name
	case tenant.LocationCodeComponentCity:
		return input.City
	case tenant.LocationCodeComponentState:
		return input.StateAbbreviation
	case tenant.LocationCodeComponentPostalCode:
		return input.PostalCode
	default:
		return ""
	}
}

func normalizedToken(value string, casing tenant.LocationCodeCasing) string {
	token := stringutils.NormalizeIdentifier(value)
	if casing == tenant.LocationCodeCasingLower {
		return strings.ToLower(token)
	}

	return strings.ToUpper(token)
}
