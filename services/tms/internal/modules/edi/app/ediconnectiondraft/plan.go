package ediconnectiondraft

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultNote          = "Trenova's default; the document does not say"
	configurationNote    = "Used on %d of your partners' outbound profiles; the document does not say"
	gsFromISANote        = "The GS ID defaults to the ISA ID; the document does not give a separate one"
	groupDefaultNote     = "The standard functional group for this transaction set"
	versionFromISA       = "The interchange's X12 version; the document gives none for this set"
	agentCustomerNote    = "Chosen by the agent from your customers; the document does not name it"
	draftPartnerLabel    = "Partner"
	fieldPartnerCustomer = "partner.customer"
	transportLabel       = "Communication profile"
)

type profileWrite struct {
	set       edi.TransactionSet
	direction edi.DocumentDirection
	request   *ediservice.UpsertEDIPartnerDocumentProfileRequest
	starter   bool
	change    ediports.ChangeKind
}

type controlWrite struct {
	set       edi.TransactionSet
	direction edi.DocumentDirection
	kind      edi.ControlNumberKind
	next      int64
}

type writePlan struct {
	partner         *edi.EDIPartner
	transport       *ediservice.UpsertEDICommunicationProfileRequest
	transportChange ediports.ChangeKind
	profiles        []*profileWrite
	mappings        []*edi.EDIMappingProfileItem
	controls        []controlWrite
}

type planner struct {
	s          *Service
	req        *ediports.ConnectionDraftRequest
	pages      *pageIndex
	evidence   *evidence
	samples    []*specSample
	match      *partnerMatch
	current    *currentSetup
	identity   *ediports.InterchangeIdentity
	values     map[string]ediports.DraftValue
	method     edi.ConnectionMethod
	draft      *ediports.ConnectionDraft
	writes     *writePlan
	validateID pulid.ID
	docTypes   map[string]pulid.ID
}

func (s *Service) PlanConnectionDraft(
	ctx context.Context,
	req *ediports.ConnectionDraftRequest,
) (*ediports.ConnectionDraft, error) {
	p, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	return p.draft, nil
}

func (s *Service) plan(
	ctx context.Context,
	req *ediports.ConnectionDraftRequest,
) (*planner, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	req = withRedactedQuotes(req)

	p := &planner{
		s:        s,
		req:      req,
		pages:    newPageIndex(req.Document.Pages),
		evidence: newEvidence(),
		values:   map[string]ediports.DraftValue{},
		writes:   &writePlan{},
		docTypes: map[string]pulid.ID{},
		draft: &ediports.ConnectionDraft{
			DocumentID:     req.Document.ID,
			FileName:       req.Document.FileName,
			PageCount:      len(req.Document.Pages),
			DocumentDigest: DocumentDigest(req.Document.Pages),
		},
	}
	p.gatherEvidence(ctx)

	var err error
	if p.match, err = s.matchPartner(ctx, req, p.evidence.hints()); err != nil {
		return nil, err
	}
	p.validateID = p.match.partnerID()
	if p.validateID.IsNil() {
		p.validateID = pulid.MustNew("edip_")
	}
	p.current = readCurrent(
		p.match.setup,
		edi.ConnectionMethod(p.evidence.hint(fieldTransportMethod)),
	)
	if p.identity, err = s.repo.GetInterchangeIdentity(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	if p.identity == nil {
		p.identity = &ediports.InterchangeIdentity{}
	}

	p.resolveValues()
	if err = p.buildPartner(ctx); err != nil {
		return nil, err
	}
	p.buildInterchange()
	p.buildTransport()
	if err = p.buildProfiles(ctx); err != nil {
		return nil, err
	}
	if err = p.buildMappings(ctx); err != nil {
		return nil, err
	}
	p.buildGaps()
	p.buildSamples()
	p.buildConnection()
	p.settle()

	return p, nil
}

func (p *planner) gatherEvidence(ctx context.Context) {
	for inputIndex := range p.req.Values {
		input := &p.req.Values[inputIndex]
		spec := fieldCatalog[input.Field]
		p.evidence.add(input.Field, readValue(p.pages, spec, input))
	}
	for _, resolution := range p.req.Resolutions {
		p.evidence.resolutions[resolution.Field] = resolution
	}

	partnerIDs := make([]string, 0, 2)
	for _, c := range p.evidence.byField[fieldPartnerID] {
		if c.status == ediports.ValueStatusAccepted || c.status == ediports.ValueStatusHeld {
			partnerIDs = append(partnerIDs, c.value)
		}
	}
	p.samples = findSamples(p.pages)
	for _, sample := range p.samples {
		sample.parse(ctx)
		sample.orient(partnerIDs)
		sample.contribute(p.evidence)
	}
}

func (p *planner) resolve(key string, alternative *fallback, required bool) ediports.DraftValue {
	spec := fieldCatalog[key]
	if alternative == nil && spec.fallback != "" {
		alternative = &fallback{
			value:  spec.fallback,
			status: ediports.ValueStatusDefaulted,
			source: ediports.DraftSource{Kind: ediports.SourceKindDefault, Note: defaultNote},
		}
	}
	value := resolveField(spec, p.evidence, p.current.values[key], alternative, required)
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
	method := p.resolve(fieldTransportMethod, nil, true)
	if method.Status.Usable() {
		p.method = edi.ConnectionMethod(method.Value)
	}

	for _, key := range sortedKeys(scopePartner, scopeInterchange, scopeCarrier) {
		spec := fieldCatalog[key]
		p.resolve(key, p.interchangeFallback(key), spec.required)
	}
	for _, key := range []string{fieldPartnerGSID, fieldOurGSID} {
		p.resolve(key, p.interchangeFallback(key), false)
	}

	if p.method != "" {
		for _, key := range sortedKeys(scopeTransport) {
			spec := fieldCatalog[key]
			if key == fieldTransportMethod || !spec.usedBy(p.method) {
				continue
			}
			p.resolve(key, nil, spec.required)
		}
	}

	for _, set := range supportedSets {
		for _, direction := range directions {
			supported := setField(set, direction, setAttrSupported)
			if len(p.evidence.byField[supported]) == 0 &&
				p.current.values[supported] == "" &&
				!p.hasResolution(supported) {
				continue
			}
			p.resolve(supported, nil, false)
		}
	}
	for _, kind := range controlNumberKinds {
		key := controlNumberField(kind)
		if len(p.evidence.byField[key]) > 0 || p.hasResolution(key) {
			p.resolve(key, nil, false)
		}
	}
}

func (p *planner) hasResolution(key string) bool {
	_, ok := p.evidence.resolutions[key]
	return ok
}

func (p *planner) interchangeFallback(key string) *fallback {
	identity := p.identity
	switch key {
	case fieldOurQualifier:
		return identityFallback(identity, identity.Qualifier)
	case fieldOurID:
		return identityFallback(identity, identity.ID)
	case fieldOurGSID:
		ours, ok := p.values[fieldOurID]
		if ok && ours.Source.Kind == ediports.SourceKindConfiguration {
			if configured := identityFallback(identity, identity.ApplicationID); configured != nil {
				return configured
			}
		}
		return p.gsFallback(fieldOurID)
	case fieldPartnerGSID:
		return p.gsFallback(fieldPartnerID)
	default:
		return nil
	}
}

func identityFallback(identity *ediports.InterchangeIdentity, value string) *fallback {
	if identity == nil || identity.Profiles == 0 || value == "" {
		return nil
	}
	return configurationFallback(value, identity.Profiles)
}

func (p *planner) gsFallback(isaKey string) *fallback {
	value, ok := p.usable(isaKey)
	if !ok {
		return nil
	}
	return &fallback{
		value:  value,
		status: ediports.ValueStatusDefaulted,
		source: ediports.DraftSource{Kind: ediports.SourceKindDefault, Note: gsFromISANote},
	}
}

func configurationFallback(value string, profiles int) *fallback {
	return &fallback{
		value:  value,
		status: ediports.ValueStatusDefaulted,
		source: ediports.DraftSource{
			Kind: ediports.SourceKindConfiguration,
			Note: fmt.Sprintf(configurationNote, profiles),
		},
	}
}

func sortedKeys(scopes ...fieldScope) []string {
	keys := make([]string, 0, len(fieldCatalog))
	for key, spec := range fieldCatalog {
		if slices.Contains(scopes, spec.scope) {
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
		fieldPartnerName, fieldPartnerCode, fieldPartnerContactName, fieldPartnerContactEmail,
		fieldPartnerContactPhone, fieldPartnerTimezone, fieldPartnerCountry,
		fieldTransportMethod, fieldTransportEnvironment,
		"as2.localAs2Id", "as2.partnerAs2Id", "as2.endpointUrl", "as2.mdnMode", "as2.mdnUrl",
		"as2.signingAlgorithm", "as2.encryptionAlgorithm", "as2.compressionAlgorithm",
		"as2.basicAuthUsername", "transport.host", "transport.port", "transport.username",
		"transport.authMode", "transport.inboundDirectory", "transport.outboundDirectory",
		"transport.archiveDirectory", "transport.fileNamingPattern", "van.providerName",
		"van.mailboxId",
		fieldPartnerQualifier, fieldPartnerID, fieldPartnerGSID, fieldOurQualifier, fieldOurID,
		fieldOurGSID, fieldX12Version, fieldUsageIndicator, fieldElementSep, fieldSegmentSep,
		fieldComponentSep, fieldRepetitionSep, fieldCarrierSCAC,
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

func (p *planner) fields(keys ...string) []ediports.DraftValue {
	out := make([]ediports.DraftValue, 0, len(keys))
	for _, key := range keys {
		if value, ok := p.values[key]; ok {
			out = append(out, value)
		}
	}
	return out
}

func (p *planner) buildPartner(ctx context.Context) error {
	draft := p.match.draft
	draft.Fields = p.fields(sortedKeys(scopePartner)...)

	if p.req.CustomerID.IsNotNil() {
		customer, err := p.customerValue(ctx)
		if err != nil {
			return err
		}
		if customer.Status == ediports.ValueStatusAccepted {
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

	draft.Change = ediports.ChangeKindAdd
	if p.match.setup != nil {
		draft.Change = ediports.ChangeKindSame
		if slices.ContainsFunc(draft.Fields, func(value ediports.DraftValue) bool {
			return value.Change.Writes()
		}) {
			draft.Change = ediports.ChangeKindChange
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

func (p *planner) customerValue(ctx context.Context) (*ediports.DraftValue, error) {
	labels, err := p.s.repo.ResolveMappingTargets(ctx, &ediports.ResolveEDIMappingTargetsRequest{
		TenantInfo: p.req.TenantInfo,
		Targets:    customerTarget(p.req.CustomerID),
	})
	if err != nil {
		return nil, err
	}
	customer := &ediports.DraftValue{
		Field:  fieldPartnerCustomer,
		Label:  "Customer",
		Status: ediports.ValueStatusAccepted,
		Source: ediports.DraftSource{Kind: ediports.SourceKindAgent, Note: agentCustomerNote},
	}
	name, found := labels[edi.MappingEntityTypeCustomer][p.req.CustomerID]
	if !found {
		customer.Status = ediports.ValueStatusInvalid
		customer.Reason = "No customer with that ID exists in this organization"
		customer.Change = ediports.ChangeKindNone
		return customer, nil
	}
	customer.Value = name
	if p.match.setup != nil && p.match.setup.Partner.CustomerID.IsNotNil() {
		customer.Current = p.match.setup.Partner.CustomerID.String()
		if p.match.setup.Partner.CustomerID == p.req.CustomerID {
			customer.Current = name
		}
	}
	withChange(customer)
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

func applyPartnerField(partner *edi.EDIPartner, value *ediports.DraftValue, customerID pulid.ID) {
	switch value.Field {
	case fieldPartnerCode:
		partner.Code = value.Value
	case fieldPartnerName:
		partner.Name = value.Value
	case fieldPartnerContactName:
		partner.ContactName = value.Value
	case fieldPartnerContactEmail:
		partner.ContactEmail = value.Value
	case fieldPartnerContactPhone:
		partner.ContactPhone = value.Value
	case fieldPartnerTimezone:
		partner.Timezone = value.Value
	case fieldPartnerCountry:
		partner.Country = value.Value
	case fieldPartnerCustomer:
		partner.CustomerID = customerID
	}
}

func (p *planner) buildInterchange() {
	keys := sortedKeys(scopeInterchange, scopeCarrier)
	p.draft.Interchange = p.fields(keys...)
}

func (p *planner) envelope(
	direction edi.DocumentDirection,
	existing *edi.EDIPartnerDocumentProfile,
) edi.X12EnvelopeSettings {
	envelope := edi.DefaultX12EnvelopeSettings()
	set := func(target *string, key string) {
		if value, ok := p.usable(key); ok {
			*target = value
		} else {
			*target = ""
		}
	}
	if existing != nil {
		envelope = existing.Envelope
		set = func(target *string, key string) {
			if value, ok := p.values[key]; ok &&
				value.Status == ediports.ValueStatusAccepted && value.Value != "" {
				*target = value.Value
			}
		}
	}
	partnerQualifier, partnerID := &envelope.InterchangeReceiverQualifier,
		&envelope.InterchangeReceiverID
	ourQualifier, ourID := &envelope.InterchangeSenderQualifier, &envelope.InterchangeSenderID
	partnerGS, ourGS := &envelope.ApplicationReceiverCode, &envelope.ApplicationSenderCode
	if direction == edi.DocumentDirectionInbound {
		partnerQualifier, ourQualifier = ourQualifier, partnerQualifier
		partnerID, ourID = ourID, partnerID
		partnerGS, ourGS = ourGS, partnerGS
	}
	set(partnerQualifier, fieldPartnerQualifier)
	set(partnerID, fieldPartnerID)
	set(ourQualifier, fieldOurQualifier)
	set(ourID, fieldOurID)
	set(partnerGS, fieldPartnerGSID)
	set(ourGS, fieldOurGSID)
	set(&envelope.InterchangeUsageIndicator, fieldUsageIndicator)
	set(&envelope.ElementSeparator, fieldElementSep)
	set(&envelope.SegmentTerminator, fieldSegmentSep)
	set(&envelope.ComponentSeparator, fieldComponentSep)
	set(&envelope.RepetitionSeparator, fieldRepetitionSep)
	return envelope
}

var interchangeConfigKeys = map[string]string{
	"isaSenderQualifier":   fieldOurQualifier,
	"isaSenderId":          fieldOurID,
	"isaReceiverQualifier": fieldPartnerQualifier,
	"isaReceiverId":        fieldPartnerID,
	"gsSenderId":           fieldOurGSID,
	"gsReceiverId":         fieldPartnerGSID,
	"x12Version":           fieldX12Version,
}

func (p *planner) buildTransport() {
	if p.method == "" {
		return
	}
	existing := p.current.transport
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

	change := ediports.ChangeKindAdd
	if existing != nil {
		change = ediports.ChangeKindSame
		if changed {
			change = ediports.ChangeKindChange
		}
	}
	p.draft.Transport = &ediports.DraftTransport{
		ProfileID:   request.ProfileID,
		Version:     request.Version,
		Name:        request.Name,
		Method:      p.method,
		Status:      entity.Status,
		Change:      change,
		Fields:      fields,
		Credentials: ediservice.CredentialRequirements(entity),
	}
	if multiErr := p.s.validator.ValidateCommunicationProfile(entity); multiErr != nil {
		p.addFailures(transportLabel, multiErr.Errors)
	}
	p.writes.transport = request
	p.writes.transportChange = change
}

func (p *planner) transportFields(isNew bool) []ediports.DraftValue {
	keys := []string{fieldTransportMethod}
	for _, key := range sortedKeys(scopeTransport) {
		if key != fieldTransportMethod && fieldCatalog[key].usedBy(p.method) {
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
			fields[index].Change = ediports.ChangeKindAdd
		}
	}
	return fields
}

func (p *planner) transportRequest(
	existing *edi.EDICommunicationProfile,
) (request *ediservice.UpsertEDICommunicationProfileRequest, secrets map[string]string) {
	request = &ediservice.UpsertEDICommunicationProfileRequest{
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

func (p *planner) fillTransportConfig(config map[string]any, fields []ediports.DraftValue) bool {
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
		spec := fieldCatalog[field.Field]
		if spec.configKey != "" && field.Status.Usable() {
			set(spec.configKey, field.Value)
		}
	}
	for _, configKey := range slices.Sorted(maps.Keys(interchangeConfigKeys)) {
		if value, ok := p.usable(interchangeConfigKeys[configKey]); ok {
			set(configKey, value)
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
	value, _ := p.usable(fieldPartnerName)
	return value
}

func (p *planner) buildConnection() {
	connection := ediports.DraftConnection{Method: p.method}
	for profileIndex := range p.draft.DocumentProfiles {
		profile := &p.draft.DocumentProfiles[profileIndex]
		connection.Sets = append(connection.Sets, ediports.DraftSetDirection{
			TransactionSet: profile.TransactionSet,
			Direction:      profile.Direction,
		})
		switch profile.Direction {
		case edi.DocumentDirectionInbound:
			connection.Inbound = true
		case edi.DocumentDirectionOutbound:
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
	if p.draft.Partner.Change == ediports.ChangeKindAdd {
		return
	}
	p.draft.Partner.Change = ediports.ChangeKindChange
}

func (p *planner) addFailures(record string, errs []*errortypes.Error) {
	for _, err := range errs {
		p.draft.Failures = append(p.draft.Failures, ediports.DraftFailure{
			Record:  record,
			Field:   err.Field,
			Message: err.Message,
		})
	}
}

func titleSet(set edi.TransactionSet, direction edi.DocumentDirection) string {
	return fmt.Sprintf("%s %s", set, strings.ToLower(string(direction)))
}

func customerTarget(customerID pulid.ID) map[edi.MappingEntityType][]pulid.ID {
	targets := make(map[edi.MappingEntityType][]pulid.ID, 1)
	targets[edi.MappingEntityTypeCustomer] = []pulid.ID{customerID}
	return targets
}
