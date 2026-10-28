package ediconnectiondraft

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
)

type currentSetup struct {
	setup     *ediconnectiondraftports.EDIPartnerSetup
	transport *edi.EDICommunicationProfile
	version   *ladingproject.LadingVersion
	exchanged map[string]bool
	values    map[string]string
}

func readCurrent(
	setup *ediconnectiondraftports.EDIPartnerSetup,
	method edi.ConnectionMethod,
) *currentSetup {
	current := &currentSetup{
		setup:     setup,
		exchanged: map[string]bool{},
		values:    map[string]string{},
	}
	if setup == nil {
		return current
	}

	partner := setup.Partner
	current.values[fieldPartnerCode] = partner.Code
	current.values[fieldPartnerName] = partner.Name
	current.values[fieldPartnerContactName] = partner.ContactName
	current.values[fieldPartnerContactEmail] = partner.ContactEmail
	current.values[fieldPartnerContactPhone] = partner.ContactPhone
	current.values[fieldPartnerTimezone] = partner.Timezone
	current.values[fieldPartnerCountry] = partner.Country

	current.transport = pickTransport(setup.CommunicationProfiles, method)
	if current.transport != nil {
		current.values[fieldTransportMethod] = string(current.transport.Method)
		for key, spec := range fieldCatalog {
			if spec.scope == scopeTransport && spec.configKey != "" {
				current.values[key] = maputils.StringValue(current.transport.Config, spec.configKey)
			}
		}
	}

	if setup.LatestVersion != nil {
		current.version = setup.LatestVersion
		readManifest(current, &setup.LatestVersion.Manifest)
	}
	return current
}

func readManifest(current *currentSetup, manifest *lading.Check) {
	if manifest.Partner != nil && manifest.Partner.Interchange != nil {
		readInterchange(current.values, manifest.Partner.Interchange)
	}
	readMaps(current, manifest)
	readAcknowledgments(current.values, manifest)
}

func readInterchange(values map[string]string, interchange *lading.Interchange) {
	if interchange.Them != nil {
		values[fieldPartnerQualifier] = interchange.Them.Qualifier
		values[fieldPartnerID] = interchange.Them.ID
	}
	if interchange.Us != nil {
		values[fieldOurQualifier] = interchange.Us.Qualifier
		values[fieldOurID] = interchange.Us.ID
	}
	if interchange.Group != nil {
		values[fieldPartnerGSID] = interchange.Group.Theirs
		values[fieldOurGSID] = interchange.Group.Ours
	}
	values[fieldX12Version] = interchange.Version
	values[fieldUsageIndicator] = "T"
	if interchange.Usage == valueProduction {
		values[fieldUsageIndicator] = "P"
	}
	keys := []string{fieldElementSep, fieldSegmentSep, fieldComponentSep, fieldRepetitionSep}
	for index, separator := range interchange.Separators {
		if index < len(keys) {
			values[keys[index]] = separator
		}
	}
}

func readMaps(current *currentSetup, manifest *lading.Check) {
	versions := make(map[string]string, len(manifest.Guides))
	for _, guide := range manifest.Guides {
		versions[guide.Name] = guide.Version
	}
	for index := range manifest.Maps {
		item := &manifest.Maps[index]
		if item.Set == "" {
			continue
		}
		set := edidocument.TransactionSet(item.Set)
		direction := edidocument.DocumentDirectionOutbound
		guide := item.To
		if item.Direction == lading.DirectionInbound {
			direction = edidocument.DocumentDirectionInbound
			guide = item.From
		}
		current.exchanged[setKey(set, direction)] = true
		current.values[setField(set, direction, setAttrSupported)] = valueTrue
		if version := versions[guide]; version != "" {
			current.values[setField(set, direction, setAttrVersion)] = version
		}
	}
}

func readAcknowledgments(values map[string]string, manifest *lading.Check) {
	for _, flow := range manifest.Flows {
		for index := range flow.Clauses {
			clause := &flow.Clauses[index]
			if clause.Kind != lading.ClauseReceive || clause.Set == "" {
				continue
			}
			ack := clause.Acknowledge
			if ack == lading.AcknowledgeNone {
				ack = ackNone
			}
			set := edidocument.TransactionSet(clause.Set)
			values[setField(set, edidocument.DocumentDirectionInbound, setAttrAck)] = ack
		}
	}
}

func pickTransport(
	profiles []*edi.EDICommunicationProfile,
	method edi.ConnectionMethod,
) *edi.EDICommunicationProfile {
	var first *edi.EDICommunicationProfile
	for _, profile := range profiles {
		if profile.Method == edi.ConnectionMethodInternal {
			continue
		}
		if method != "" && profile.Method == method {
			return profile
		}
		if first == nil {
			first = profile
		}
	}
	if method != "" {
		return nil
	}
	return first
}

func (c *currentSetup) sequence(
	documentTypeID pulid.ID,
	kind edidocument.ControlNumberKind,
) *edidocument.EDIControlNumberSequence {
	if c.setup == nil {
		return nil
	}
	for _, sequence := range c.setup.ControlNumbers {
		if sequence.DocumentTypeID == documentTypeID && sequence.Kind == kind {
			return sequence
		}
	}
	return nil
}

func (c *currentSetup) mapping(
	entityType edi.MappingEntityType,
	sourceID pulid.ID,
) *edi.EDIMappingProfileItem {
	if c.setup == nil {
		return nil
	}
	for _, item := range c.setup.MappingItems {
		if item.EntityType == entityType && item.SourceID == sourceID {
			return item
		}
	}
	return nil
}
