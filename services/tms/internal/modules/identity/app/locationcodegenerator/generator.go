//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package locationcodegenerator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

var errLocationCodeRequestRequired = errors.New("location code request is required")

var fixedLocationCodePeriod = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

type Params struct {
	fx.In

	SequenceConfigRepo tenantports.SequenceConfigRepository
	SequenceGenerator  seqgen.Generator
}

type Generator struct {
	sequenceConfigRepo tenantports.SequenceConfigRepository
	sequenceGenerator  seqgen.Generator
}

func New(p Params) tenantports.LocationCodeGenerator {
	return &Generator{
		sequenceConfigRepo: p.SequenceConfigRepo,
		sequenceGenerator:  p.SequenceGenerator,
	}
}

func (g *Generator) Generate(
	ctx context.Context,
	req tenantports.LocationCodeGenerateRequest,
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

	format := &sequenceconfig.SequenceFormat{
		Type:           sequenceconfig.SequenceTypeLocationCode,
		Prefix:         prefix,
		SequenceDigits: int(strategy.SequenceDigits),
		UseSeparators:  strategy.Separator != "",
		SeparatorChar:  strategy.Separator,
	}

	code, err := g.sequenceGenerator.Generate(ctx, &seqgen.GenerateRequest{
		Type:   sequenceconfig.SequenceTypeLocationCode,
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
		Time:   fixedLocationCodePeriod,
		Format: format,
	})
	if err != nil {
		return "", fmt.Errorf("generate location code sequence: %w", err)
	}
	if strings.TrimSpace(code) == "" {
		return "", errors.New("generated location code cannot be empty")
	}
	if len([]rune(code)) > sequenceconfig.MaxLocationCodeLength {
		return "", fmt.Errorf(
			"generated location code cannot exceed %d characters",
			sequenceconfig.MaxLocationCodeLength,
		)
	}

	return code, nil
}

func (g *Generator) BuildPrefix(
	input tenantports.LocationCodeInput,
	strategy *sequenceconfig.LocationCodeStrategy,
) (string, error) {
	strategy = sequenceconfig.EffectiveLocationCodeStrategy(strategy)
	if err := strategy.Validate(); err != nil {
		return "", err
	}

	fallback := normalizedToken(strategy.FallbackPrefix, strategy.Casing)
	if fallback == "" {
		return "", errors.New("fallback prefix must contain letters or digits")
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
	req tenantports.LocationCodeGenerateRequest,
) (*sequenceconfig.LocationCodeStrategy, error) {
	doc, err := g.sequenceConfigRepo.GetByTenant(ctx, tenantports.GetSequenceConfigRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: req.OrganizationID,
			BuID:  req.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get sequence config: %w", err)
	}

	for _, cfg := range doc.Configs {
		if cfg != nil && cfg.SequenceType == sequenceconfig.SequenceTypeLocationCode {
			strategy := sequenceconfig.EffectiveLocationCodeStrategy(cfg.LocationCodeStrategy)
			if err = strategy.Validate(); err != nil {
				return nil, fmt.Errorf("invalid location code strategy: %w", err)
			}

			return strategy, nil
		}
	}

	return sequenceconfig.DefaultLocationCodeStrategy(), nil
}

func componentValue(
	input tenantports.LocationCodeInput,
	component sequenceconfig.LocationCodeComponent,
) string {
	switch component {
	case sequenceconfig.LocationCodeComponentName:
		return input.Name
	case sequenceconfig.LocationCodeComponentCity:
		return input.City
	case sequenceconfig.LocationCodeComponentState:
		return input.StateAbbreviation
	case sequenceconfig.LocationCodeComponentPostalCode:
		return input.PostalCode
	default:
		return ""
	}
}

func normalizedToken(value string, casing sequenceconfig.LocationCodeCasing) string {
	token := stringutils.NormalizeIdentifier(value)
	if casing == sequenceconfig.LocationCodeCasingLower {
		return strings.ToLower(token)
	}

	return strings.ToUpper(token)
}
