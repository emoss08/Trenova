package fmcsaconnector

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/shared/fmcsa"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	messageLookupIdentifier = "FMCSA lookups require a DOT or docket number"
	messageSearchName       = "FMCSA search requires a carrier name"
	messageAutocompleteTerm = "autocomplete requires at least two characters"
	minAutocompleteLength   = 2
)

var (
	_ services.CarrierIntelClient   = (*Client)(nil)
	_ services.CarrierIntelSearcher = (*Client)(nil)
)

type Client struct {
	sdk      *fmcsa.Client
	recorder intelkit.Recorder
}

func (c *Client) Provider() integration.Type {
	return integration.TypeFMCSAQCMobile
}

func (c *Client) IsSandbox() bool {
	return false
}

func (c *Client) Lookup(
	ctx context.Context,
	req *services.CarrierIntelLookupRequest,
) (*services.CarrierIntelLookupResult, error) {
	if req == nil {
		return nil, errorMapper.InvalidRequest(messageLookupIdentifier)
	}
	dot := strings.TrimSpace(req.Identifier.DOTNumber)
	docket := strings.TrimSpace(req.Identifier.DocketNumber)
	if dot == "" && stringutils.DigitsOnly(docket) == "" {
		if req.Identifier.IsZero() {
			return nil, errorMapper.InvalidRequest(messageLookupIdentifier)
		}
		return nil, errorMapper.Unsupported(messageLookupIdentifier)
	}

	outcome := &intelkit.CallOutcome{
		Endpoint:  carrierintel.EndpointComposite,
		DOTNumber: dot,
		Started:   time.Now(),
		Units:     1,
	}
	composite, err := c.composite(ctx, dot, docket)
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		return nil, outcome.Mapped
	}

	profile := normalizeComposite(composite)
	if resolved := profile.DOTNumber(); resolved != "" {
		outcome.DOTNumber = resolved
	}
	outcome.Found = true
	c.recorder.Record(ctx, outcome)

	ref := profile.Identity.ProviderRef()
	if ref == "" {
		ref = outcome.DOTNumber
	}
	return &services.CarrierIntelLookupResult{
		Profile:     profile,
		Raw:         composite.Raw,
		Endpoint:    carrierintel.EndpointComposite,
		ProviderRef: ref,
		Depth:       carrierintel.LookupDepthFMCSA,
		SourceAsOf:  intelkit.Unix(composite.Carrier.SnapshotDate),
	}, nil
}

func (c *Client) composite(
	ctx context.Context,
	dot, docket string,
) (*fmcsa.CompositeCarrier, error) {
	if dot == "" {
		carriers, err := c.sdk.CarriersByDocket(ctx, docket)
		if err != nil {
			return nil, err
		}
		for idx := range carriers {
			if resolved := intelkit.Text(carriers[idx].DOTNumber); resolved != "" {
				dot = resolved
				break
			}
		}
		if dot == "" {
			return nil, fmcsa.ErrNotFound
		}
	}
	return c.sdk.Composite(ctx, dot)
}

func (c *Client) Search(
	ctx context.Context,
	req *services.CarrierIntelSearchRequest,
) (*services.CarrierIntelSearchResult, error) {
	if req == nil {
		return nil, errorMapper.InvalidRequest(messageSearchName)
	}
	name := strings.TrimSpace(req.Query)
	if name == "" {
		name = strings.TrimSpace(req.CompanyName)
	}
	if name == "" {
		if req.EIN != "" || req.VIN != "" || req.State != "" {
			return nil, errorMapper.Unsupported(messageSearchName)
		}
		return nil, errorMapper.InvalidRequest(messageSearchName)
	}

	offset := max(req.Offset, 0)
	carriers, err := c.searchByName(
		ctx,
		carrierintel.EndpointSearch,
		name,
		offset,
		max(req.Limit, 0),
	)
	if err != nil {
		return nil, err
	}

	state := strings.ToUpper(strings.TrimSpace(req.State))
	items := make([]services.CarrierIntelSearchHit, 0, len(carriers))
	for idx := range carriers {
		profile := normalizeCarrier(&carriers[idx])
		if state != "" && !inState(profile, state) {
			continue
		}
		ref := profile.Identity.ProviderRef()
		items = append(items, services.CarrierIntelSearchHit{Profile: profile, ProviderRef: ref})
	}
	return &services.CarrierIntelSearchResult{
		Items: items,
		Total: offset + len(items),
	}, nil
}

func (c *Client) Autocomplete(
	ctx context.Context,
	query string,
	limit int,
) ([]services.CarrierIntelSuggestion, error) {
	term := strings.TrimSpace(query)
	if len([]rune(term)) < minAutocompleteLength {
		return nil, errorMapper.InvalidRequest(messageAutocompleteTerm)
	}

	carriers, err := c.searchByName(ctx, carrierintel.EndpointAutocomplete, term, 0, max(limit, 0))
	if err != nil {
		return nil, err
	}

	suggestions := make([]services.CarrierIntelSuggestion, 0, len(carriers))
	for idx := range carriers {
		carrier := &carriers[idx]
		suggestions = append(suggestions, services.CarrierIntelSuggestion{
			DOTNumber: intelkit.Text(carrier.DOTNumber),
			LegalName: intelkit.Text(carrier.LegalName),
			DBAName:   intelkit.Text(carrier.DBAName),
			City:      intelkit.Text(carrier.PhysicalCity),
			State:     strings.ToUpper(intelkit.Text(carrier.PhysicalState)),
		})
	}
	return suggestions, nil
}

func (c *Client) searchByName(
	ctx context.Context,
	endpoint carrierintel.Endpoint,
	name string,
	offset, limit int,
) ([]fmcsa.Carrier, error) {
	outcome := &intelkit.CallOutcome{
		Endpoint: endpoint,
		Started:  time.Now(),
		Units:    1,
	}
	carriers, err := c.sdk.CarriersByName(ctx, name, offset, limit)
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		if fmcsa.IsNotFound(err) {
			return []fmcsa.Carrier{}, nil
		}
		return nil, outcome.Mapped
	}
	outcome.Found = true
	c.recorder.Record(ctx, outcome)
	return carriers, nil
}

func inState(profile *carrierintel.Profile, state string) bool {
	if profile.Identity == nil || profile.Identity.PhysicalAddress == nil ||
		profile.Identity.PhysicalAddress.State == "" {
		return true
	}
	return profile.Identity.PhysicalAddress.State == state
}
