package aiproviderservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/optional"
)

func (s *Service) Patch(
	ctx context.Context,
	req *services.PatchAIProviderRequest,
	actor *services.RequestActor,
) (*aiprovider.Provider, error) {
	if req.Version <= 0 {
		return nil, errortypes.NewValidationError(
			"version", errortypes.ErrRequired, "Version is required",
		)
	}

	edits, err := s.patchEdits(req, actor)
	if err != nil {
		return nil, err
	}

	saved, previous, err := s.modify(ctx, &modifyRequest{
		tenantInfo: req.TenantInfo,
		providerID: req.ID,
		version:    req.Version,
		actor:      actor,
		edit: func(provider *aiprovider.Provider) {
			for _, edit := range edits {
				edit(provider)
			}
		},
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(&auditParams{
		provider:  saved,
		previous:  previous,
		operation: permission.OpUpdate,
		actor:     actor,
		comment:   "AI provider updated",
	})

	return saved.Redacted(), nil
}

type providerEdit func(*aiprovider.Provider)

func (s *Service) patchEdits(
	req *services.PatchAIProviderRequest,
	actor *services.RequestActor,
) ([]providerEdit, error) {
	multiErr := errortypes.NewMultiError()
	edits := make([]providerEdit, 0, 7)

	edits = appendBool(edits, multiErr, &boolPatch{
		field: "enabled", label: "Enabled", value: req.Enabled,
		set: func(p *aiprovider.Provider, v bool) { p.Enabled = v },
	})
	edits = appendBool(edits, multiErr, &boolPatch{
		field: "trusted", label: "Trusted", value: req.Trusted,
		set: func(p *aiprovider.Provider, v bool) { p.Trusted = v },
	})
	edits = appendBool(edits, multiErr, &boolPatch{
		field: "allowPrivateNetwork", label: "Allow private network",
		value: req.AllowPrivateNetwork,
		set:   func(p *aiprovider.Provider, v bool) { p.AllowPrivateNetwork = v },
	})
	if allow := req.AllowPrivateNetwork; allow.Set && allow.Value != nil && *allow.Value {
		if err := s.checkPrivateNetwork(true); err != nil {
			return nil, err
		}
	}

	if req.Tasks.Set {
		tasks := slices.Clone(req.Tasks.Value)
		edits = append(edits, func(p *aiprovider.Provider) { p.Tasks = tasks })
	}
	if req.InputCostPerMillion.Set {
		price := req.InputCostPerMillion.Value
		edits = append(edits, func(p *aiprovider.Provider) { p.InputCostPerMillion = price })
	}
	if req.OutputCostPerMillion.Set {
		price := req.OutputCostPerMillion.Value
		edits = append(edits, func(p *aiprovider.Provider) { p.OutputCostPerMillion = price })
	}

	if req.APIKey.Set {
		incoming := ""
		if req.APIKey.Value != nil {
			incoming = *req.APIKey.Value
		}
		replacement, err := s.keyReplacement(&keyChange{incoming: incoming, actor: actor})
		if err != nil {
			return nil, err
		}
		edits = append(edits, func(p *aiprovider.Provider) { p.ReplaceAPIKey(replacement) })
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return edits, nil
}

type boolPatch struct {
	field string
	label string
	value optional.Value[*bool]
	set   func(*aiprovider.Provider, bool)
}

func appendBool(
	edits []providerEdit,
	multiErr *errortypes.MultiError,
	patch *boolPatch,
) []providerEdit {
	if !patch.value.Set {
		return edits
	}
	if patch.value.Value == nil {
		multiErr.Add(patch.field, errortypes.ErrRequired, "{0} cannot be cleared", patch.label)
		return edits
	}
	value := *patch.value.Value
	set := patch.set

	return append(edits, func(p *aiprovider.Provider) { set(p, value) })
}
