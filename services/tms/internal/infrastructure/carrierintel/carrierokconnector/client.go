package carrierokconnector

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/shared/carrierok"
)

const (
	maxAutocompleteLimit = 15

	messageIdentifierRequired = "a carrier identifier is required"
	messageFMCSAIdentifier    = "an FMCSA-depth lookup requires a DOT or docket number"
	messageSearchCriteria     = "at least one search criterion is required"
	messageAutocompleteTerm   = "autocomplete requires at least two characters"
)

var (
	_ services.CarrierIntelClient          = (*Client)(nil)
	_ services.CarrierIntelSearcher        = (*Client)(nil)
	_ services.CarrierIntelNativeMonitor   = (*Client)(nil)
	_ services.CarrierIntelEquipmentLookup = (*Client)(nil)
)

type Client struct {
	sdk      *carrierok.Client
	recorder intelkit.Recorder
}

func isInvalidQuery(err error) bool {
	return errors.Is(err, carrierok.ErrInvalidQuery)
}

func (c *Client) Provider() integration.Type {
	return integration.TypeCarrierOK
}

func (c *Client) IsSandbox() bool {
	return c.sdk.Sandbox()
}

func (c *Client) Lookup(
	ctx context.Context,
	req *services.CarrierIntelLookupRequest,
) (*services.CarrierIntelLookupResult, error) {
	if req == nil || req.Identifier.IsZero() {
		return nil, errorMapper.InvalidRequest(messageIdentifierRequired)
	}

	depth := req.Depth
	if !depth.IsValid() {
		depth = carrierintel.LookupDepthFull
	}
	id := trimIdentifier(req.Identifier)

	var (
		endpoint carrierintel.Endpoint
		fetch    func(ctx context.Context) (*carrierok.Profile, error)
	)
	switch depth {
	case carrierintel.LookupDepthFMCSA:
		query, ok := fmcsaQuery(id)
		if !ok {
			return nil, errorMapper.InvalidRequest(messageFMCSAIdentifier)
		}
		endpoint = carrierintel.EndpointProfileFMCSA
		fetch = func(ctx context.Context) (*carrierok.Profile, error) {
			return c.sdk.ProfileFMCSA(ctx, query)
		}
	case carrierintel.LookupDepthLite:
		query := profileQuery(id)
		endpoint = carrierintel.EndpointProfileLite
		fetch = func(ctx context.Context) (*carrierok.Profile, error) {
			return c.sdk.ProfileLite(ctx, query)
		}
	default:
		query := profileQuery(id)
		endpoint = carrierintel.EndpointProfileFull
		fetch = func(ctx context.Context) (*carrierok.Profile, error) {
			return c.sdk.Profile(ctx, query)
		}
	}

	outcome := &intelkit.CallOutcome{
		Endpoint:  endpoint,
		DOTNumber: id.DOTNumber,
		Started:   time.Now(),
		Units:     1,
	}
	sdkProfile, err := fetch(ctx)
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		return nil, outcome.Mapped
	}

	profile := normalizeProfile(sdkProfile)
	if dot := profile.DOTNumber(); dot != "" {
		outcome.DOTNumber = dot
	}
	outcome.Found = true
	c.recorder.Record(ctx, outcome)

	return &services.CarrierIntelLookupResult{
		Profile:     profile,
		Raw:         sdkProfile.Raw,
		Endpoint:    endpoint,
		ProviderRef: providerRef(sdkProfile, profile),
		Depth:       depth,
		SourceAsOf:  intelkit.Unix(sdkProfile.Identity.SnapshotDate),
	}, nil
}

func (c *Client) Search(
	ctx context.Context,
	req *services.CarrierIntelSearchRequest,
) (*services.CarrierIntelSearchResult, error) {
	if req == nil {
		return nil, errorMapper.InvalidRequest(messageSearchCriteria)
	}
	params := carrierok.SearchParams{
		Query:       strings.TrimSpace(req.Query),
		CompanyName: strings.TrimSpace(req.CompanyName),
		EIN:         strings.TrimSpace(req.EIN),
		VIN:         strings.TrimSpace(req.VIN),
		State:       strings.TrimSpace(req.State),
		Limit:       max(req.Limit, 0),
		Offset:      max(req.Offset, 0),
	}
	if params.Query == "" && params.CompanyName == "" && params.EIN == "" && params.VIN == "" &&
		params.State == "" {
		return nil, errorMapper.InvalidRequest(messageSearchCriteria)
	}

	outcome := &intelkit.CallOutcome{
		Endpoint: carrierintel.EndpointSearch,
		Started:  time.Now(),
		Units:    1,
	}
	result, err := c.sdk.Search(ctx, params)
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		return nil, outcome.Mapped
	}
	outcome.Found = true
	c.recorder.Record(ctx, outcome)

	items := make([]services.CarrierIntelSearchHit, 0, len(result.Items))
	for idx := range result.Items {
		profile := normalizeProfile(&result.Items[idx])
		items = append(items, services.CarrierIntelSearchHit{
			Profile:     profile,
			ProviderRef: providerRef(&result.Items[idx], profile),
		})
	}
	return &services.CarrierIntelSearchResult{
		Items: items,
		Total: int(max(result.TotalCount, int64(len(items)))),
	}, nil
}

func (c *Client) Autocomplete(
	ctx context.Context,
	query string,
	limit int,
) ([]services.CarrierIntelSuggestion, error) {
	term := strings.TrimSpace(query)
	if len([]rune(term)) < 2 {
		return nil, errorMapper.InvalidRequest(messageAutocompleteTerm)
	}

	outcome := &intelkit.CallOutcome{
		Endpoint: carrierintel.EndpointAutocomplete,
		Started:  time.Now(),
		Units:    1,
	}
	suggestions, err := c.sdk.Autocomplete(ctx, term, min(max(limit, 0), maxAutocompleteLimit))
	if err != nil {
		outcome.RawErr = err
		outcome.Mapped = errorMapper.Map(err)
		c.recorder.Record(ctx, outcome)
		return nil, outcome.Mapped
	}
	outcome.Found = true
	c.recorder.Record(ctx, outcome)

	out := make([]services.CarrierIntelSuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		out = append(out, services.CarrierIntelSuggestion{
			DOTNumber: suggestion.DOTNumber,
			LegalName: suggestion.LegalName,
			DBAName:   suggestion.DBAName,
			City:      suggestion.City,
			State:     suggestion.State,
		})
	}
	return out, nil
}

func trimIdentifier(id services.CarrierIntelIdentifier) services.CarrierIntelIdentifier {
	return services.CarrierIntelIdentifier{
		DOTNumber:    strings.TrimSpace(id.DOTNumber),
		DocketNumber: strings.TrimSpace(id.DocketNumber),
		Company:      strings.TrimSpace(id.Company),
		EIN:          strings.TrimSpace(id.EIN),
		Email:        strings.TrimSpace(id.Email),
		Phone:        strings.TrimSpace(id.Phone),
	}
}

func profileQuery(id services.CarrierIntelIdentifier) carrierok.ProfileQuery {
	switch {
	case id.DOTNumber != "":
		return carrierok.ProfileQuery{DOTNumber: id.DOTNumber}
	case id.DocketNumber != "":
		return carrierok.ProfileQuery{DocketNumber: id.DocketNumber}
	case id.Company != "":
		return carrierok.ProfileQuery{Company: id.Company}
	case id.EIN != "":
		return carrierok.ProfileQuery{EIN: id.EIN}
	case id.Email != "":
		return carrierok.ProfileQuery{Email: id.Email}
	default:
		return carrierok.ProfileQuery{Phone: id.Phone}
	}
}

func fmcsaQuery(id services.CarrierIntelIdentifier) (carrierok.FMCSAQuery, bool) {
	switch {
	case id.DOTNumber != "":
		return carrierok.FMCSAQuery{DOTNumber: id.DOTNumber}, true
	case id.DocketNumber != "":
		return carrierok.FMCSAQuery{DocketNumber: id.DocketNumber}, true
	default:
		return carrierok.FMCSAQuery{}, false
	}
}

func providerRef(sdkProfile *carrierok.Profile, profile *carrierintel.Profile) string {
	if docID := intelkit.Text(sdkProfile.Identity.DocID); docID != "" {
		return docID
	}
	if ref := profile.Identity.ProviderRef(); ref != "" {
		return ref
	}
	return sdkProfile.ProfileID()
}
