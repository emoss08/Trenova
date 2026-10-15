package ediconnectiondraft

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fakeRepo struct {
	partners  []*ediports.EDIPartnerSetup
	templates []*edi.EDITemplateVersion
	identity  *ediports.InterchangeIdentity
	targets   map[edi.MappingEntityType]map[pulid.ID]string
}

func (f *fakeRepo) FindPartnerMatches(
	_ context.Context,
	req *ediports.FindEDIPartnerMatchesRequest,
) ([]*ediports.EDIPartnerMatch, error) {
	out := make([]*ediports.EDIPartnerMatch, 0)
	for _, setup := range f.partners {
		partner := setup.Partner
		match := &ediports.EDIPartnerMatch{
			Partner:    partner,
			ByCode:     req.Code != "" && strings.EqualFold(partner.Code, req.Code),
			ByName:     req.Name != "" && strings.EqualFold(partner.Name, req.Name),
			ByCustomer: req.CustomerID.IsNotNil() && partner.CustomerID == req.CustomerID,
		}
		for _, profile := range setup.DocumentProfiles {
			if req.ISAID != "" && profile.Direction == edi.DocumentDirectionOutbound &&
				profile.Envelope.InterchangeReceiverID == req.ISAID {
				match.ByInterchange = true
			}
		}
		for _, profile := range setup.CommunicationProfiles {
			if req.AS2ID != "" && profile.Config["partnerAS2Id"] == req.AS2ID {
				match.ByAS2 = true
			}
		}
		if match.ByCode || match.ByName || match.ByCustomer || match.ByInterchange || match.ByAS2 {
			out = append(out, match)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetPartnerSetup(
	_ context.Context,
	req *ediports.GetEDIPartnerSetupRequest,
) (*ediports.EDIPartnerSetup, error) {
	for _, setup := range f.partners {
		if setup.Partner.ID == req.PartnerID {
			return setup, nil
		}
	}
	return nil, errortypes.NewNotFoundError("EDIPartner not found")
}

func (f *fakeRepo) ListTemplateCandidates(
	_ context.Context,
	req *ediports.ListEDITemplateCandidatesRequest,
) ([]*edi.EDITemplateVersion, error) {
	out := make([]*edi.EDITemplateVersion, 0)
	for _, version := range f.templates {
		if version.Template.TransactionSet == req.TransactionSet &&
			version.Template.Direction == req.Direction {
			out = append(out, version)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetInterchangeIdentity(
	context.Context,
	pagination.TenantInfo,
) (*ediports.InterchangeIdentity, error) {
	if f.identity == nil {
		return &ediports.InterchangeIdentity{}, nil
	}
	return f.identity, nil
}

func (f *fakeRepo) ResolveMappingTargets(
	_ context.Context,
	req *ediports.ResolveEDIMappingTargetsRequest,
) (map[edi.MappingEntityType]map[pulid.ID]string, error) {
	out := make(map[edi.MappingEntityType]map[pulid.ID]string)
	for entityType, ids := range req.Targets {
		out[entityType] = map[pulid.ID]string{}
		for _, id := range ids {
			if label, ok := f.targets[entityType][id]; ok {
				out[entityType][id] = label
			}
		}
	}
	return out, nil
}

type fakeDocumentTypes struct{}

func (fakeDocumentTypes) ListDocumentTypes(
	_ context.Context,
	req ediports.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	return []*edi.EDIDocumentType{{
		ID:             documentTypeID(req.TransactionSet, req.Direction),
		TransactionSet: req.TransactionSet,
		Direction:      req.Direction,
	}}, nil
}

func (fakeDocumentTypes) SelectDocumentTypeOptions(
	context.Context,
	*ediports.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIDocumentType], error) {
	return &pagination.ListResult[*edi.EDIDocumentType]{}, nil
}

func documentTypeID(set edi.TransactionSet, direction edi.DocumentDirection) pulid.ID {
	return pulid.ID("edidt_x12_" + string(set) + "_" + strings.ToLower(string(direction)))
}

type fakeSetup struct {
	settingsErrors []edix12.Diagnostic
	partners       []*edi.EDIPartner
	transports     []*ediservice.UpsertEDICommunicationProfileRequest
	profiles       []*ediservice.UpsertEDIPartnerDocumentProfileRequest
	mappings       []*ediports.SaveMappingItemsRequest
	controls       []*ediports.ResetEDIControlNumberRequest
	failOn         string
}

func (f *fakeSetup) fail(step string) error {
	if f.failOn == step {
		return errortypes.NewBusinessError(step + " failed")
	}
	return nil
}

func (f *fakeSetup) CreatePartner(
	_ context.Context,
	entity *edi.EDIPartner,
	_ *permissionports.RequestActor,
) (*edi.EDIPartner, error) {
	if err := f.fail("partner"); err != nil {
		return nil, err
	}
	created := *entity
	created.ID = pulid.MustNew("edip_")
	f.partners = append(f.partners, &created)
	return &created, nil
}

func (f *fakeSetup) UpdatePartner(
	_ context.Context,
	entity *edi.EDIPartner,
	_ *permissionports.RequestActor,
) (*edi.EDIPartner, error) {
	if err := f.fail("partner"); err != nil {
		return nil, err
	}
	f.partners = append(f.partners, entity)
	return entity, nil
}

func (f *fakeSetup) CreateCommunicationProfile(
	_ context.Context,
	req *ediservice.UpsertEDICommunicationProfileRequest,
	_ *permissionports.RequestActor,
) (*edi.EDICommunicationProfile, error) {
	if err := f.fail("transport"); err != nil {
		return nil, err
	}
	f.transports = append(f.transports, req)
	return &edi.EDICommunicationProfile{ID: pulid.MustNew("edicp_"), Name: req.Name}, nil
}

func (f *fakeSetup) UpdateCommunicationProfile(
	_ context.Context,
	req *ediservice.UpsertEDICommunicationProfileRequest,
	_ *permissionports.RequestActor,
) (*edi.EDICommunicationProfile, error) {
	f.transports = append(f.transports, req)
	return &edi.EDICommunicationProfile{ID: req.ProfileID, Name: req.Name}, nil
}

func (f *fakeSetup) UpsertPartnerDocumentProfile(
	_ context.Context,
	req *ediservice.UpsertEDIPartnerDocumentProfileRequest,
	_ *permissionports.RequestActor,
) (*edi.EDIPartnerDocumentProfile, error) {
	f.profiles = append(f.profiles, req)
	id := req.ProfileID
	if id.IsNil() {
		id = pulid.MustNew("edidp_")
	}
	return &edi.EDIPartnerDocumentProfile{ID: id, Name: req.Name}, nil
}

func (f *fakeSetup) SaveMappingProfile(
	_ context.Context,
	req *ediports.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	f.mappings = append(f.mappings, req)
	return slices.Clone(req.Items), nil
}

func (f *fakeSetup) StartControlNumber(
	_ context.Context,
	req *ediports.ResetEDIControlNumberRequest,
	_ *permissionports.RequestActor,
) (*edi.EDIControlNumberSequence, error) {
	f.controls = append(f.controls, req)
	return &edi.EDIControlNumberSequence{ID: pulid.MustNew("edicn_"), Kind: req.Kind}, nil
}

func (f *fakeSetup) ValidatePartnerSettings(
	context.Context,
	*ediservice.ValidatePartnerSettingsRequest,
) ([]edix12.Diagnostic, error) {
	return f.settingsErrors, nil
}

type fakeTemplates struct {
	ediports.EDITemplateRepository
	ensured []edi.DocumentDirection
}

func (f *fakeTemplates) EnsureStarterTemplate(
	_ context.Context,
	_ pagination.TenantInfo,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	f.ensured = append(f.ensured, direction)
	template := &edi.EDITemplate{ID: pulid.MustNew("editpl_"), TransactionSet: set, Direction: direction}
	return template, &edi.EDITemplateVersion{ID: pulid.MustNew("editv_"), TemplateID: template.ID}, nil
}

type harness struct {
	service   *Service
	repo      *fakeRepo
	setup     *fakeSetup
	templates *fakeTemplates
}

func newHarness() *harness {
	h := &harness{
		repo:      &fakeRepo{targets: map[edi.MappingEntityType]map[pulid.ID]string{}},
		setup:     &fakeSetup{},
		templates: &fakeTemplates{},
	}
	h.service = &Service{
		repo:          h.repo,
		templates:     h.templates,
		documentTypes: fakeDocumentTypes{},
		setup:         h.setup,
		validator:     ediservice.NewValidator(),
		db:            passthroughDB{},
		l:             zap.NewNop(),
	}
	return h
}

type passthroughDB struct{}

func (passthroughDB) DB() *bun.DB { return nil }

func (passthroughDB) DBForContext(context.Context) bun.IDB { return nil }

func (passthroughDB) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

func (passthroughDB) HealthCheck(context.Context) error { return nil }

func (passthroughDB) IsHealthy(context.Context) bool { return true }

func (passthroughDB) Close() error { return nil }
