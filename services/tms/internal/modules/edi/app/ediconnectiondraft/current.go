package ediconnectiondraft

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
)

type currentSetup struct {
	setup     *ediports.EDIPartnerSetup
	transport *edi.EDICommunicationProfile
	profiles  map[string]*edi.EDIPartnerDocumentProfile
	values    map[string]string
}

func profileKey(set edi.TransactionSet, direction edi.DocumentDirection) string {
	return string(set) + "/" + string(direction)
}

func readCurrent(setup *ediports.EDIPartnerSetup, method edi.ConnectionMethod) *currentSetup {
	current := &currentSetup{
		setup:    setup,
		profiles: map[string]*edi.EDIPartnerDocumentProfile{},
		values:   map[string]string{},
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
		current.values[fieldX12Version] = maputils.StringValue(current.transport.Config, "x12Version")
	}

	for _, profile := range setup.DocumentProfiles {
		key := profileKey(profile.TransactionSet, profile.Direction)
		if existing, ok := current.profiles[key]; ok &&
			existing.Status == edi.DocumentStatusActive {
			continue
		}
		current.profiles[key] = profile
		readProfile(current.values, profile)
	}
	readEnvelope(current.values, current.profiles)

	return current
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

func readProfile(values map[string]string, profile *edi.EDIPartnerDocumentProfile) {
	set, direction := profile.TransactionSet, profile.Direction
	values[setField(set, direction, setAttrSupported)] = "true"
	values[setField(set, direction, setAttrVersion)] = profile.X12VersionOverride
	values[setField(set, direction, setAttrFunctionalGroupID)] = profile.FunctionalGroupID
	switch {
	case profile.Acknowledgment.Expected:
		values[setField(set, direction, setAttrAck)] = string(profile.Acknowledgment.Type)
	default:
		values[setField(set, direction, setAttrAck)] = ackNone
	}
	if profile.Acknowledgment.SLAInMinutes > 0 {
		values[setField(set, direction, setAttrAckWindowMinutes)] = strconv.FormatInt(
			profile.Acknowledgment.SLAInMinutes,
			10,
		)
	}
	if direction == edi.DocumentDirectionOutbound {
		if scac := nestedString(profile.PartnerSettings, "carrier", "scac"); scac != "" {
			values[fieldCarrierSCAC] = scac
		}
	}
}

func readEnvelope(values map[string]string, profiles map[string]*edi.EDIPartnerDocumentProfile) {
	for _, direction := range []edi.DocumentDirection{
		edi.DocumentDirectionOutbound,
		edi.DocumentDirectionInbound,
	} {
		for _, set := range supportedSets {
			profile, ok := profiles[profileKey(set, direction)]
			if !ok {
				continue
			}
			envelope := profile.Envelope
			partnerQualifier, partnerID := envelope.InterchangeReceiverQualifier,
				envelope.InterchangeReceiverID
			ourQualifier, ourID := envelope.InterchangeSenderQualifier, envelope.InterchangeSenderID
			partnerGS, ourGS := envelope.ApplicationReceiverCode, envelope.ApplicationSenderCode
			if direction == edi.DocumentDirectionInbound {
				partnerQualifier, ourQualifier = ourQualifier, partnerQualifier
				partnerID, ourID = ourID, partnerID
				partnerGS, ourGS = ourGS, partnerGS
			}
			values[fieldPartnerQualifier] = partnerQualifier
			values[fieldPartnerID] = partnerID
			values[fieldOurQualifier] = ourQualifier
			values[fieldOurID] = ourID
			values[fieldPartnerGSID] = partnerGS
			values[fieldOurGSID] = ourGS
			values[fieldUsageIndicator] = envelope.InterchangeUsageIndicator
			values[fieldElementSep] = envelope.ElementSeparator
			values[fieldSegmentSep] = envelope.SegmentTerminator
			values[fieldComponentSep] = envelope.ComponentSeparator
			values[fieldRepetitionSep] = envelope.RepetitionSeparator
			return
		}
	}
}

func nestedString(values map[string]any, path ...string) string {
	current := any(values)
	for _, key := range path {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = next[key]
	}
	value, _ := current.(string)
	return value
}

func (c *currentSetup) sequence(
	documentTypeID pulid.ID,
	kind edi.ControlNumberKind,
) *edi.EDIControlNumberSequence {
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
