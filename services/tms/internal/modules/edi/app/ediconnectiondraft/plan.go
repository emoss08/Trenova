package ediconnectiondraft

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionfield"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultNote          = "Trenova's default; the document does not say"
	configurationNote    = "Used on %d of your partners' Lading projects; the document does not say"
	gsFromISANote        = "The GS ID defaults to the ISA ID; the document does not give a separate one"
	versionFromISA       = "The interchange's X12 version; the document gives none for this set"
	agentCustomerNote    = "Chosen by the agent from your customers; the document does not name it"
	draftPartnerLabel    = "Partner"
	fieldPartnerCustomer = "partner.customer"
	transportLabel       = "Communication profile"
)

type controlWrite struct {
	set       edidocument.TransactionSet
	direction edidocument.DocumentDirection
	kind      edidocument.ControlNumberKind
	next      int64
}

type writePlan struct {
	partner         *edi.EDIPartner
	transport       *edisetup.UpsertEDICommunicationProfileRequest
	transportChange ediconnectiondraftports.ChangeKind
	project         *projectWrite
	mappings        []*edi.EDIMappingProfileItem
	controls        []controlWrite
}

type planner struct {
	s          *Service
	req        *ediconnectiondraftports.ConnectionDraftRequest
	pages      *ediconnectionfield.Pages
	evidence   *ediconnectionfield.Evidence
	samples    []*ediconnectionfield.Sample
	match      *partnerMatch
	current    *ediconnectionfield.Current
	identity   *ediconnectiondraftports.InterchangeIdentity
	values     map[string]ediconnectiondraftports.DraftValue
	method     edi.ConnectionMethod
	draft      *ediconnectiondraftports.ConnectionDraft
	writes     *writePlan
	encoded    *lading.Encoded
	manifest   *lading.Check
	validateID pulid.ID
	docTypes   map[string]pulid.ID
}

func (s *Service) PlanConnectionDraft(
	ctx context.Context,
	req *ediconnectiondraftports.ConnectionDraftRequest,
) (*ediconnectiondraftports.ConnectionDraft, error) {
	p, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	return p.draft, nil
}

func (s *Service) plan(
	ctx context.Context,
	req *ediconnectiondraftports.ConnectionDraftRequest,
) (*planner, error) {
	if err := ediconnectionfield.ValidateRequest(req); err != nil {
		return nil, err
	}
	req = ediconnectionfield.WithRedactedQuotes(req)

	p := &planner{
		s:        s,
		req:      req,
		pages:    ediconnectionfield.NewPages(req.Document.Pages),
		evidence: ediconnectionfield.NewEvidence(),
		values:   map[string]ediconnectiondraftports.DraftValue{},
		writes:   &writePlan{},
		docTypes: map[string]pulid.ID{},
		draft: &ediconnectiondraftports.ConnectionDraft{
			DocumentID:     req.Document.ID,
			FileName:       req.Document.FileName,
			PageCount:      len(req.Document.Pages),
			DocumentDigest: ediconnectionfield.DocumentDigest(req.Document.Pages),
		},
	}
	p.gatherEvidence(ctx)

	var err error
	if p.match, err = s.matchPartner(ctx, req, p.evidence.Hints()); err != nil {
		return nil, err
	}
	p.validateID = p.match.partnerID()
	if p.validateID.IsNil() {
		p.validateID = pulid.MustNew("edip_")
	}
	p.current = ediconnectionfield.ReadCurrent(
		p.match.setup,
		edi.ConnectionMethod(p.evidence.Hint(ediconnectionfield.TransportMethod)),
	)
	if p.identity, err = s.repo.GetInterchangeIdentity(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	if p.identity == nil {
		p.identity = &ediconnectiondraftports.InterchangeIdentity{}
	}

	p.resolveValues()
	if err = p.buildPartner(ctx); err != nil {
		return nil, err
	}
	p.buildInterchange()
	p.buildTransport()
	exchanges, err := p.buildSets(ctx)
	if err != nil {
		return nil, err
	}
	if err = p.buildProject(ctx, exchanges); err != nil {
		return nil, err
	}
	if err = p.buildMappings(ctx); err != nil {
		return nil, err
	}
	p.buildGaps()
	p.buildSamples(ctx)
	p.buildConnection()
	p.settle()

	return p, nil
}

func (p *planner) gatherEvidence(ctx context.Context) {
	for inputIndex := range p.req.Values {
		input := &p.req.Values[inputIndex]
		spec := ediconnectionfield.Catalog[input.Field]
		p.evidence.Add(input.Field, ediconnectionfield.ReadValue(p.pages, spec, input))
	}
	for _, resolution := range p.req.Resolutions {
		p.evidence.Resolutions[resolution.Field] = resolution
	}

	partnerIDs := make([]string, 0, 2)
	for _, c := range p.evidence.ByField[ediconnectionfield.PartnerID] {
		if c.Status == ediconnectiondraftports.ValueStatusAccepted ||
			c.Status == ediconnectiondraftports.ValueStatusHeld {
			partnerIDs = append(partnerIDs, c.Value)
		}
	}
	p.samples = ediconnectionfield.FindSamples(p.pages)
	for _, sample := range p.samples {
		sample.Parse(ctx)
		sample.Orient(partnerIDs)
		sample.Contribute(p.evidence)
	}
}

func (p *planner) resolve(
	key string,
	alternative *ediconnectionfield.Fallback,
	required bool,
) ediconnectiondraftports.DraftValue {
	spec := ediconnectionfield.Catalog[key]
	if alternative == nil && spec.Fallback != "" {
		alternative = &ediconnectionfield.Fallback{
			Value:  spec.Fallback,
			Status: ediconnectiondraftports.ValueStatusDefaulted,
			Source: ediconnectiondraftports.DraftSource{
				Kind: ediconnectiondraftports.SourceKindDefault,
				Note: defaultNote,
			},
		}
	}
	value := ediconnectionfield.Resolve(
		spec,
		p.evidence,
		p.current.Values[key],
		alternative,
		required,
	)
	p.values[key] = value
	return value
}

func (p *planner) usable(key string) (string, bool) {
	value, ok := p.values[key]
	if !ok || !value.Status.Usable() || value.Value == "" {
		return "", false
	}
	return value.Value, true
}

func (p *planner) resolveValues() {
	method := p.resolve(ediconnectionfield.TransportMethod, nil, true)
	if method.Status.Usable() {
		p.method = edi.ConnectionMethod(method.Value)
	}

	for _, key := range sortedKeys(ediconnectionfield.ScopePartner, ediconnectionfield.ScopeInterchange) {
		spec := ediconnectionfield.Catalog[key]
		p.resolve(key, p.interchangeFallback(key), spec.Required)
	}
	for _, key := range []string{ediconnectionfield.PartnerGSID, ediconnectionfield.OurGSID} {
		p.resolve(key, p.interchangeFallback(key), false)
	}

	if p.method != "" {
		for _, key := range sortedKeys(ediconnectionfield.ScopeTransport) {
			spec := ediconnectionfield.Catalog[key]
			if key == ediconnectionfield.TransportMethod || !spec.UsedBy(p.method) {
				continue
			}
			p.resolve(key, nil, spec.Required)
		}
	}

	for _, set := range ediconnectionfield.SupportedSets {
		for _, direction := range ediconnectionfield.Directions {
			supported := ediconnectionfield.SetField(
				set,
				direction,
				ediconnectionfield.SetAttrSupported,
			)
			if len(p.evidence.ByField[supported]) == 0 &&
				p.current.Values[supported] == "" &&
				!p.hasResolution(supported) {
				continue
			}
			p.resolve(supported, nil, false)
		}
	}
	for _, kind := range ediconnectionfield.ControlNumberKinds {
		key := ediconnectionfield.ControlNumberField(kind)
		if len(p.evidence.ByField[key]) > 0 || p.hasResolution(key) {
			p.resolve(key, nil, false)
		}
	}
}

func (p *planner) hasResolution(key string) bool {
	_, ok := p.evidence.Resolutions[key]
	return ok
}

func (p *planner) interchangeFallback(key string) *ediconnectionfield.Fallback {
	identity := p.identity
	switch key {
	case ediconnectionfield.OurQualifier:
		return identityFallback(identity, identity.Qualifier)
	case ediconnectionfield.OurID:
		return identityFallback(identity, identity.ID)
	case ediconnectionfield.OurGSID:
		ours, ok := p.values[ediconnectionfield.OurID]
		if ok && ours.Source.Kind == ediconnectiondraftports.SourceKindConfiguration {
			if configured := identityFallback(identity, identity.ApplicationID); configured != nil {
				return configured
			}
		}
		return p.gsFallback(ediconnectionfield.OurID)
	case ediconnectionfield.PartnerGSID:
		return p.gsFallback(ediconnectionfield.PartnerID)
	default:
		return nil
	}
}

func identityFallback(
	identity *ediconnectiondraftports.InterchangeIdentity,
	value string,
) *ediconnectionfield.Fallback {
	if identity == nil || identity.Partners == 0 || value == "" {
		return nil
	}
	return configurationFallback(value, identity.Partners)
}

func (p *planner) gsFallback(isaKey string) *ediconnectionfield.Fallback {
	value, ok := p.usable(isaKey)
	if !ok {
		return nil
	}
	return &ediconnectionfield.Fallback{
		Value:  value,
		Status: ediconnectiondraftports.ValueStatusDefaulted,
		Source: ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindDefault,
			Note: gsFromISANote,
		},
	}
}

func configurationFallback(value string, partners int) *ediconnectionfield.Fallback {
	return &ediconnectionfield.Fallback{
		Value:  value,
		Status: ediconnectiondraftports.ValueStatusDefaulted,
		Source: ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindConfiguration,
			Note: fmt.Sprintf(configurationNote, partners),
		},
	}
}

func sortedKeys(scopes ...ediconnectionfield.Scope) []string {
	keys := make([]string, 0, len(ediconnectionfield.Catalog))
	for key, spec := range ediconnectionfield.Catalog {
		if slices.Contains(scopes, spec.Scope) {
			keys = append(keys, key)
		}
	}
	slices.SortFunc(keys, func(a, b string) int {
		return catalogOrder(a) - catalogOrder(b)
	})
	return keys
}

var catalogPositions = func() map[string]int {
	order := []string{
		ediconnectionfield.PartnerName, ediconnectionfield.PartnerCode, ediconnectionfield.PartnerContactName, ediconnectionfield.PartnerContactEmail,
		ediconnectionfield.PartnerContactPhone, ediconnectionfield.PartnerTimezone, ediconnectionfield.PartnerCountry,
		ediconnectionfield.TransportMethod, ediconnectionfield.TransportEnvironment,
		"as2.localAs2Id", "as2.partnerAs2Id", "as2.endpointUrl", "as2.mdnMode", "as2.mdnUrl",
		"as2.signingAlgorithm", "as2.encryptionAlgorithm", "as2.compressionAlgorithm",
		"as2.basicAuthUsername", "transport.host", "transport.port", "transport.username",
		"transport.authMode", "transport.inboundDirectory", "transport.outboundDirectory",
		"transport.archiveDirectory", "transport.fileNamingPattern", "van.providerName",
		"van.mailboxId",
		ediconnectionfield.PartnerQualifier, ediconnectionfield.PartnerID, ediconnectionfield.PartnerGSID, ediconnectionfield.OurQualifier, ediconnectionfield.OurID,
		ediconnectionfield.OurGSID, ediconnectionfield.X12Version, ediconnectionfield.UsageIndicator, ediconnectionfield.ElementSep, ediconnectionfield.SegmentSep,
		ediconnectionfield.ComponentSep, ediconnectionfield.RepetitionSep,
	}
	positions := make(map[string]int, len(order))
	for index, key := range order {
		positions[key] = index
	}
	return positions
}()

func catalogOrder(key string) int {
	if position, ok := catalogPositions[key]; ok {
		return position
	}
	return len(catalogPositions)
}

func (p *planner) fields(keys ...string) []ediconnectiondraftports.DraftValue {
	out := make([]ediconnectiondraftports.DraftValue, 0, len(keys))
	for _, key := range keys {
		if value, ok := p.values[key]; ok {
			out = append(out, value)
		}
	}
	return out
}

func (p *planner) buildPartner(ctx context.Context) error {
	draft := p.match.draft
	draft.Fields = p.fields(sortedKeys(ediconnectionfield.ScopePartner)...)

	if p.req.CustomerID.IsNotNil() {
		customer, err := p.customerValue(ctx)
		if err != nil {
			return err
		}
		if customer.Status == ediconnectiondraftports.ValueStatusAccepted {
			draft.CustomerID = p.req.CustomerID
		}
		p.values[customer.Field] = *customer
		draft.Fields = append(draft.Fields, *customer)
	}

	partner := p.partnerEntity()
	for valueIndex := range draft.Fields {
		value := &draft.Fields[valueIndex]
		if value.Change.Writes() {
			applyPartnerField(partner, value, draft.CustomerID)
		}
	}

	draft.Change = ediconnectiondraftports.ChangeKindAdd
	if p.match.setup != nil {
		draft.Change = ediconnectiondraftports.ChangeKindSame
		if slices.ContainsFunc(draft.Fields, func(value ediconnectiondraftports.DraftValue) bool {
			return value.Change.Writes()
		}) {
			draft.Change = ediconnectiondraftports.ChangeKindChange
		}
	}
	if multiErr := p.s.validator.ValidatePartner(partner); multiErr != nil {
		p.addFailures(draftPartnerLabel, multiErr.Errors)
	}
	draft.Status = partner.Status
	p.draft.Partner = draft
	p.writes.partner = partner
	return nil
}

func (p *planner) customerValue(ctx context.Context) (*ediconnectiondraftports.DraftValue, error) {
	labels, err := p.s.repo.ResolveMappingTargets(
		ctx,
		&ediconnectiondraftports.ResolveEDIMappingTargetsRequest{
			TenantInfo: p.req.TenantInfo,
			Targets:    customerTarget(p.req.CustomerID),
		},
	)
	if err != nil {
		return nil, err
	}
	customer := &ediconnectiondraftports.DraftValue{
		Field:  fieldPartnerCustomer,
		Label:  "Customer",
		Status: ediconnectiondraftports.ValueStatusAccepted,
		Source: ediconnectiondraftports.DraftSource{
			Kind: ediconnectiondraftports.SourceKindAgent,
			Note: agentCustomerNote,
		},
	}
	name, found := labels[edi.MappingEntityTypeCustomer][p.req.CustomerID]
	if !found {
		customer.Status = ediconnectiondraftports.ValueStatusInvalid
		customer.Reason = "No customer with that ID exists in this organization"
		customer.Change = ediconnectiondraftports.ChangeKindNone
		return customer, nil
	}
	customer.Value = name
	if p.match.setup != nil && p.match.setup.Partner.CustomerID.IsNotNil() {
		customer.Current = p.match.setup.Partner.CustomerID.String()
		if p.match.setup.Partner.CustomerID == p.req.CustomerID {
			customer.Current = name
		}
	}
	ediconnectionfield.WithChange(customer)
	return customer, nil
}

func (p *planner) partnerEntity() *edi.EDIPartner {
	if p.match.setup == nil {
		return &edi.EDIPartner{
			BusinessUnitID: p.req.TenantInfo.BuID,
			OrganizationID: p.req.TenantInfo.OrgID,
			Kind:           edi.PartnerKindExternal,
			Status:         domaintypes.StatusInactive,
			Settings:       map[string]any{},
		}
	}
	existing := *p.match.setup.Partner
	existing.Settings = maps.Clone(existing.Settings)
	return &existing
}

func applyPartnerField(
	partner *edi.EDIPartner,
	value *ediconnectiondraftports.DraftValue,
	customerID pulid.ID,
) {
	switch value.Field {
	case ediconnectionfield.PartnerCode:
		partner.Code = value.Value
	case ediconnectionfield.PartnerName:
		partner.Name = value.Value
	case ediconnectionfield.PartnerContactName:
		partner.ContactName = value.Value
	case ediconnectionfield.PartnerContactEmail:
		partner.ContactEmail = value.Value
	case ediconnectionfield.PartnerContactPhone:
		partner.ContactPhone = value.Value
	case ediconnectionfield.PartnerTimezone:
		partner.Timezone = value.Value
	case ediconnectionfield.PartnerCountry:
		partner.Country = value.Value
	case fieldPartnerCustomer:
		partner.CustomerID = customerID
	}
}

func (p *planner) buildInterchange() {
	keys := sortedKeys(ediconnectionfield.ScopeInterchange)
	p.draft.Interchange = p.fields(keys...)
}

func (p *planner) buildTransport() {
	if p.method == "" {
		return
	}
	existing := p.current.Transport
	if existing != nil && existing.Method != p.method {
		existing = nil
	}

	fields := p.transportFields(existing == nil)
	request, secrets := p.transportRequest(existing)
	changed := p.fillTransportConfig(request.Config, fields)

	entity := &edi.EDICommunicationProfile{
		BusinessUnitID:   p.req.TenantInfo.BuID,
		OrganizationID:   p.req.TenantInfo.OrgID,
		EDIPartnerID:     p.validateID,
		Method:           p.method,
		Status:           domaintypes.Status(request.Status),
		Name:             request.Name,
		Config:           request.Config,
		EncryptedSecrets: secrets,
	}

	change := ediconnectiondraftports.ChangeKindAdd
	if existing != nil {
		change = ediconnectiondraftports.ChangeKindSame
		if changed {
			change = ediconnectiondraftports.ChangeKindChange
		}
	}
	p.draft.Transport = &ediconnectiondraftports.DraftTransport{
		ProfileID:   request.ProfileID,
		Version:     request.Version,
		Name:        request.Name,
		Method:      p.method,
		Status:      entity.Status,
		Change:      change,
		Fields:      fields,
		Credentials: edivalidation.CredentialRequirements(entity),
	}
	if multiErr := p.s.validator.ValidateCommunicationProfile(entity); multiErr != nil {
		p.addFailures(transportLabel, multiErr.Errors)
	}
	p.writes.transport = request
	p.writes.transportChange = change
}

func (p *planner) transportFields(isNew bool) []ediconnectiondraftports.DraftValue {
	keys := []string{ediconnectionfield.TransportMethod}
	for _, key := range sortedKeys(ediconnectionfield.ScopeTransport) {
		if key != ediconnectionfield.TransportMethod &&
			ediconnectionfield.Catalog[key].UsedBy(p.method) {
			keys = append(keys, key)
		}
	}
	fields := p.fields(keys...)
	if !isNew {
		return fields
	}
	for index := range fields {
		fields[index].Current = ""
		if fields[index].Value != "" && fields[index].Status.Usable() {
			fields[index].Change = ediconnectiondraftports.ChangeKindAdd
		}
	}
	return fields
}

func (p *planner) transportRequest(
	existing *edi.EDICommunicationProfile,
) (request *edisetup.UpsertEDICommunicationProfileRequest, secrets map[string]string) {
	request = &edisetup.UpsertEDICommunicationProfileRequest{
		TenantInfo:   p.req.TenantInfo,
		EDIPartnerID: p.match.partnerID(),
		Method:       p.method,
		Status:       string(domaintypes.StatusInactive),
		Name:         transportName(p.partnerName(), p.method),
		Description:  "Drafted from the partner's implementation guide " + p.req.Document.FileName,
		Config:       map[string]any{},
	}
	if existing == nil {
		return request, map[string]string{}
	}
	request.Config = maps.Clone(existing.Config)
	if request.Config == nil {
		request.Config = map[string]any{}
	}
	request.ProfileID = existing.ID
	request.Status = string(existing.Status)
	request.Name = existing.Name
	request.Description = existing.Description
	request.Version = existing.Version
	return request, existing.EncryptedSecrets
}

func (p *planner) fillTransportConfig(
	config map[string]any,
	fields []ediconnectiondraftports.DraftValue,
) bool {
	changed := false
	set := func(key, value string) {
		if value == "" {
			return
		}
		if current, _ := config[key].(string); current != value {
			changed = true
		}
		config[key] = value
	}
	for index := range fields {
		field := &fields[index]
		spec := ediconnectionfield.Catalog[field.Field]
		if spec.ConfigKey != "" && field.Status.Usable() {
			set(spec.ConfigKey, field.Value)
		}
	}
	return changed
}

func transportName(partnerName string, method edi.ConnectionMethod) string {
	if partnerName == "" {
		return string(method)
	}
	return fmt.Sprintf("%s %s", partnerName, method)
}

func (p *planner) partnerName() string {
	if p.writes.partner != nil && p.writes.partner.Name != "" {
		return p.writes.partner.Name
	}
	value, _ := p.usable(ediconnectionfield.PartnerName)
	return value
}

func (p *planner) buildConnection() {
	connection := ediconnectiondraftports.DraftConnection{Method: p.method}
	for setIndex := range p.draft.Sets {
		set := &p.draft.Sets[setIndex]
		connection.Sets = append(connection.Sets, ediconnectiondraftports.DraftSetDirection{
			TransactionSet: set.TransactionSet,
			Direction:      set.Direction,
		})
		switch set.Direction {
		case edidocument.DocumentDirectionInbound:
			connection.Inbound = true
		case edidocument.DocumentDirectionOutbound:
			connection.Outbound = true
		}
	}
	p.draft.Connection = connection

	partner := p.writes.partner
	if connection.Inbound && !partner.EnabledForInbound {
		partner.EnabledForInbound = true
		p.markPartnerChange()
	}
	if connection.Outbound && !partner.EnabledForOutbound {
		partner.EnabledForOutbound = true
		p.markPartnerChange()
	}
}

func (p *planner) markPartnerChange() {
	if p.draft.Partner.Change == ediconnectiondraftports.ChangeKindAdd {
		return
	}
	p.draft.Partner.Change = ediconnectiondraftports.ChangeKindChange
}

func (p *planner) addFailures(record string, errs []*errortypes.Error) {
	for _, err := range errs {
		p.draft.Failures = append(p.draft.Failures, ediconnectiondraftports.DraftFailure{
			Record:  record,
			Field:   err.Field,
			Message: err.Message,
		})
	}
}

func titleSet(set edidocument.TransactionSet, direction edidocument.DocumentDirection) string {
	return fmt.Sprintf("%s %s", set, strings.ToLower(string(direction)))
}

func customerTarget(customerID pulid.ID) map[edi.MappingEntityType][]pulid.ID {
	targets := make(map[edi.MappingEntityType][]pulid.ID, 1)
	targets[edi.MappingEntityTypeCustomer] = []pulid.ID{customerID}
	return targets
}
