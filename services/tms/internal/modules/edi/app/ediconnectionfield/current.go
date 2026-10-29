package ediconnectionfield

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
)

type Current struct {
	Setup     *ediconnectiondraftports.EDIPartnerSetup
	Transport *edi.EDICommunicationProfile
	Version   *ladingproject.LadingVersion
	Exchanged map[string]bool
	Values    map[string]string
}

func ReadCurrent(
	setup *ediconnectiondraftports.EDIPartnerSetup,
	method edi.ConnectionMethod,
) *Current {
	current := &Current{
		Setup:     setup,
		Exchanged: map[string]bool{},
		Values:    map[string]string{},
	}
	if setup == nil {
		return current
	}

	partner := setup.Partner
	current.Values[PartnerCode] = partner.Code
	current.Values[PartnerName] = partner.Name
	current.Values[PartnerContactName] = partner.ContactName
	current.Values[PartnerContactEmail] = partner.ContactEmail
	current.Values[PartnerContactPhone] = partner.ContactPhone
	current.Values[PartnerTimezone] = partner.Timezone
	current.Values[PartnerCountry] = partner.Country

	current.Transport = pickTransport(setup.CommunicationProfiles, method)
	if current.Transport != nil {
		current.Values[TransportMethod] = string(current.Transport.Method)
		for key, spec := range Catalog {
			if spec.Scope == ScopeTransport && spec.ConfigKey != "" {
				current.Values[key] = maputils.StringValue(current.Transport.Config, spec.ConfigKey)
			}
		}
	}

	if setup.LatestVersion != nil {
		current.Version = setup.LatestVersion
		readManifest(current, &setup.LatestVersion.Manifest)
	}
	return current
}

func readManifest(current *Current, manifest *lading.Check) {
	if manifest.Partner != nil && manifest.Partner.Interchange != nil {
		readInterchange(current.Values, manifest.Partner.Interchange)
	}
	readMaps(current, manifest)
	readAcknowledgments(current.Values, manifest)
}

func readInterchange(values map[string]string, interchange *lading.Interchange) {
	if interchange.Them != nil {
		values[PartnerQualifier] = interchange.Them.Qualifier
		values[PartnerID] = interchange.Them.ID
	}
	if interchange.Us != nil {
		values[OurQualifier] = interchange.Us.Qualifier
		values[OurID] = interchange.Us.ID
	}
	if interchange.Group != nil {
		values[PartnerGSID] = interchange.Group.Theirs
		values[OurGSID] = interchange.Group.Ours
	}
	values[X12Version] = interchange.Version
	values[UsageIndicator] = "T"
	if interchange.Usage == ValueProduction {
		values[UsageIndicator] = "P"
	}
	keys := []string{ElementSep, SegmentSep, ComponentSep, RepetitionSep}
	for index, separator := range interchange.Separators {
		if index < len(keys) {
			values[keys[index]] = separator
		}
	}
}

func readMaps(current *Current, manifest *lading.Check) {
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
		current.Exchanged[SetKey(set, direction)] = true
		current.Values[SetField(set, direction, SetAttrSupported)] = ValueTrue
		if version := versions[guide]; version != "" {
			current.Values[SetField(set, direction, SetAttrVersion)] = version
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
				ack = AckNone
			}
			set := edidocument.TransactionSet(clause.Set)
			values[SetField(set, edidocument.DocumentDirectionInbound, SetAttrAck)] = ack
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

func (c *Current) Sequence(
	documentTypeID pulid.ID,
	kind edidocument.ControlNumberKind,
) *edidocument.EDIControlNumberSequence {
	if c.Setup == nil {
		return nil
	}
	for _, sequence := range c.Setup.ControlNumbers {
		if sequence.DocumentTypeID == documentTypeID && sequence.Kind == kind {
			return sequence
		}
	}
	return nil
}

func (c *Current) Mapping(
	entityType edi.MappingEntityType,
	sourceID pulid.ID,
) *edi.EDIMappingProfileItem {
	if c.Setup == nil {
		return nil
	}
	for _, item := range c.Setup.MappingItems {
		if item.EntityType == entityType && item.SourceID == sourceID {
			return item
		}
	}
	return nil
}

func SetKey(set edidocument.TransactionSet, direction edidocument.DocumentDirection) string {
	return string(set) + "/" + string(direction)
}
