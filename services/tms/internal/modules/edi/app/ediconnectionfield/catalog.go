package ediconnectionfield

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/shared/smime"
)

var qualifierNames = map[string][]string{
	"01": {"duns", "d-u-n-s", "dun & bradstreet", "dun and bradstreet"},
	"02": {"scac", "standard carrier alpha code"},
	"08": {"ucc", "ean", "gs1"},
	"12": {"phone", "telephone"},
	"14": {"duns plus suffix", "duns+suffix"},
	"16": {"duns plus 4", "duns+4"},
	"20": {"hin", "health industry number"},
	"30": {"federal tax", "tax id", "tin"},
	"32": {"federal employer", "fein", "employer identification"},
	"ZZ": {"mutually defined", "mutually-defined"},
}

var Catalog = buildCatalog()

func buildCatalog() map[string]*Spec {
	groups := [][]*Spec{
		partnerFields(),
		transportFields(),
		as2Fields(),
		fileTransferFields(),
		interchangeFields(),
		setFields(),
		controlNumberFields(),
	}
	catalog := make(map[string]*Spec)
	for _, group := range groups {
		for _, spec := range group {
			if len(spec.spelled) > 0 {
				spec.Normalize = spelledOut(spec.Normalize, spec.spelled)
			}
			catalog[spec.Key] = spec
		}
	}
	return catalog
}

func partnerFields() []*Spec {
	return []*Spec{
		{
			Key:       PartnerCode,
			Label:     "Partner code",
			Scope:     ScopePartner,
			Required:  true,
			Normalize: text(100),
		},
		{
			Key:       PartnerName,
			Label:     "Partner name",
			Scope:     ScopePartner,
			Required:  true,
			Normalize: text(200),
		},
		{
			Key:       PartnerContactName,
			Label:     "Contact name",
			Scope:     ScopePartner,
			Normalize: text(150),
		},
		{
			Key:       PartnerContactEmail,
			Label:     "Contact email",
			Scope:     ScopePartner,
			Normalize: normalizeEmail,
		},
		{
			Key:       PartnerContactPhone,
			Label:     "Contact phone",
			Scope:     ScopePartner,
			Normalize: text(30),
		},
		{
			Key:       PartnerTimezone,
			Label:     "Time zone",
			Scope:     ScopePartner,
			Normalize: normalizeTimezone,
		},
		{
			Key:       PartnerCountry,
			Label:     "Country",
			Scope:     ScopePartner,
			Normalize: normalizeCountry,
			Fallback:  "US",
		},
	}
}

func transportFields() []*Spec {
	return []*Spec{
		{
			Key:      TransportMethod,
			Label:    "Connection method",
			Scope:    ScopeTransport,
			Required: true,
			Normalize: choice(map[string]string{
				"as2": valueAS2, "applicability statement 2": valueAS2, "sftp": valueSFTP,
				"secure ftp": valueSFTP, "ssh file transfer": valueSFTP, "van": valueVAN,
				"value added network": valueVAN, "value-added network": valueVAN,
			}),
			spelled: map[string][]string{
				valueAS2:  {"as2", "applicability statement 2"},
				valueSFTP: {"sftp", "secure ftp", "ssh file transfer"},
				valueVAN:  {"van", "value added network", "value-added network"},
			},
		},
		{
			Key:       TransportEnvironment,
			Label:     "Environment",
			Scope:     ScopeTransport,
			ConfigKey: "environment",
			methods:   externalMethods,
			Required:  true,
			Fallback:  ValueTest,
			Normalize: choice(map[string]string{
				ValueTest: ValueTest, "testing": ValueTest, ValueProduction: ValueProduction,
				"prod": ValueProduction, "live": ValueProduction,
			}),
			spelled: map[string][]string{
				ValueTest:       {ValueTest},
				ValueProduction: {ValueProduction, "prod", "live"},
			},
		},
	}
}

func transportText(
	key, label, configKey string,
	methods []edi.ConnectionMethod,
	required bool,
) *Spec {
	return &Spec{
		Key:       key,
		Label:     label,
		Scope:     ScopeTransport,
		ConfigKey: configKey,
		methods:   methods,
		Required:  required,
		Normalize: text(255),
	}
}

func as2Fields() []*Spec {
	endpoint := transportText(
		"as2.endpointUrl",
		"AS2 endpoint URL",
		editransport.ConfigKeyEndpointURL,
		as2Only,
		true,
	)
	endpoint.Normalize = normalizeURL
	mdnURL := transportText(
		"as2.mdnUrl",
		"Async MDN return URL",
		editransport.ConfigKeyMDNURL,
		as2Only,
		false,
	)
	mdnURL.Normalize = normalizeURL

	return []*Spec{
		transportText(
			"as2.localAs2Id",
			"Your AS2 ID",
			editransport.ConfigKeyLocalAS2ID,
			as2Only,
			true,
		),
		transportText(
			"as2.partnerAs2Id",
			"Partner AS2 ID",
			editransport.ConfigKeyPartnerAS2ID,
			as2Only,
			true,
		),
		endpoint,
		{
			Key:       "as2.mdnMode",
			Label:     "MDN mode",
			Scope:     ScopeTransport,
			ConfigKey: editransport.ConfigKeyMDNMode,
			methods:   as2Only,
			Required:  true,
			Normalize: choice(map[string]string{
				valueSync: valueSync, "synchronous": valueSync,
				valueAsync: valueAsync, "asynchronous": valueAsync,
			}),
			spelled: map[string][]string{
				valueSync:  {"synchronous", valueSync},
				valueAsync: {"asynchronous", valueAsync},
			},
		},
		mdnURL,
		signingAlgorithmField(),
		encryptionAlgorithmField(),
		{
			Key:       "as2.compressionAlgorithm",
			Label:     "Compression",
			Scope:     ScopeTransport,
			ConfigKey: editransport.ConfigKeyCompressionAlgorithm,
			methods:   as2Only,
			Normalize: choice(
				map[string]string{valueNone: valueNone, "zlib": editransport.CompressionZlib},
			),
			spelled: map[string][]string{valueNone: {"no compression", "uncompressed"}},
		},
		transportText(
			"as2.basicAuthUsername", "Basic auth username",
			editransport.ConfigKeyBasicAuthUsername, as2Only, false,
		),
	}
}

func fileTransferFields() []*Spec {
	port := transportText("transport.port", "Port", "port", fileMethods, true)
	port.Normalize = normalizePort
	port.Fallback = "22"
	port.matches = matchesNumber

	directories := make([]*Spec, 0, 3)
	for _, directory := range []struct{ key, label string }{
		{"inboundDirectory", "Inbound directory"},
		{"outboundDirectory", "Outbound directory"},
		{"archiveDirectory", "Archive directory"},
	} {
		spec := transportText(
			"transport."+directory.key,
			directory.label,
			directory.key,
			fileMethods,
			false,
		)
		spec.Normalize = normalizePath
		directories = append(directories, spec)
	}

	fields := []*Spec{
		transportText("transport.host", "Host", "host", fileMethods, true),
		port,
		transportText("transport.username", "Username", "username", fileMethods, true),
		{
			Key:       "transport.authMode",
			Label:     "Authentication",
			Scope:     ScopeTransport,
			ConfigKey: "authMode",
			methods:   fileMethods,
			Required:  true,
			Normalize: choice(map[string]string{
				valuePassword: valuePassword,
				"privatekey":  valuePrivateKey,
				"private key": valuePrivateKey,
				"ssh key":     valuePrivateKey,
				"public key":  valuePrivateKey,
				"key":         valuePrivateKey,
			}),
			spelled: map[string][]string{
				valuePassword:   {valuePassword},
				valuePrivateKey: {"private key", "ssh key", "public key", "key-based", "key based"},
			},
		},
	}
	fields = append(fields, directories...)
	return append(
		fields,
		transportText(
			"transport.fileNamingPattern",
			"File naming pattern",
			"fileNamingPattern",
			fileMethods,
			false,
		),
		transportText("van.providerName", "VAN provider", "providerName", vanOnly, true),
		transportText("van.mailboxId", "VAN mailbox ID", "mailboxId", vanOnly, true),
	)
}

func separatorField(key, label, fallback string) *Spec {
	return &Spec{
		Key:       key,
		Label:     label,
		Scope:     ScopeInterchange,
		Required:  true,
		Normalize: normalizeSeparator,
		spelled:   spelledSeparators(),
		Fallback:  fallback,
	}
}

func interchangeFields() []*Spec {
	return []*Spec{
		{
			Key: PartnerQualifier, Label: "Partner ISA qualifier", Scope: ScopeInterchange,
			Required: true, Normalize: normalizeQualifier, spelled: qualifierNames,
		},
		{
			Key:       PartnerID,
			Label:     "Partner ISA ID",
			Scope:     ScopeInterchange,
			Required:  true,
			Normalize: normalizeISAID,
		},
		{
			Key: OurQualifier, Label: "Your ISA qualifier", Scope: ScopeInterchange,
			Required: true, Normalize: normalizeQualifier, spelled: qualifierNames,
		},
		{
			Key:       OurID,
			Label:     "Your ISA ID",
			Scope:     ScopeInterchange,
			Required:  true,
			Normalize: normalizeISAID,
		},
		{
			Key:       PartnerGSID,
			Label:     "Partner GS ID",
			Scope:     ScopeInterchange,
			Normalize: normalizeGSID,
		},
		{Key: OurGSID, Label: "Your GS ID", Scope: ScopeInterchange, Normalize: normalizeGSID},
		{
			Key:      UsageIndicator,
			Label:    "Usage indicator",
			Scope:    ScopeInterchange,
			Required: true,
			Fallback: "T",
			Normalize: choice(
				map[string]string{"t": "T", ValueTest: "T", "p": "P", ValueProduction: "P"},
			),
			spelled: map[string][]string{"T": {ValueTest}, "P": {ValueProduction}},
		},
		{
			Key: X12Version, Label: "X12 version", Scope: ScopeInterchange,
			Required: true, Normalize: normalizeVersion, matches: matchesVersion,
		},
		separatorField(ElementSep, "Element separator", "*"),
		separatorField(SegmentSep, "Segment terminator", "~"),
		separatorField(ComponentSep, "Component separator", ">"),
		separatorField(RepetitionSep, "Repetition separator", "^"),
	}
}

func setFields() []*Spec {
	fields := make([]*Spec, 0, len(SupportedSets)*len(Directions)*5)
	for _, set := range SupportedSets {
		for _, direction := range Directions {
			fields = append(fields, setFieldsFor(set, direction)...)
		}
	}
	return fields
}

func setFieldsFor(
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) []*Spec {
	prefix := setFieldPrefix(set, direction)
	name := fmt.Sprintf("%s %s", set, strings.ToLower(string(direction)))
	return []*Spec{
		{
			Key:   prefix + SetAttrSupported,
			Label: name + " exchanged",
			Scope: ScopeSet,
			Normalize: choice(map[string]string{
				ValueTrue: ValueTrue, "yes": ValueTrue, ValueFalse: ValueFalse, "no": ValueFalse,
			}),
			matches: anyQuote,
		},
		{
			Key: prefix + SetAttrVersion, Label: name + " version", Scope: ScopeSet,
			Normalize: normalizeVersion, matches: matchesVersion,
		},
		{
			Key:   prefix + SetAttrAck,
			Label: name + " functional acknowledgment",
			Scope: ScopeSet,
			Normalize: choice(map[string]string{
				"997":          "997",
				"999":          "999",
				valueNone:      AckNone,
				"no":           AckNone,
				"not required": AckNone,
			}),
			spelled: map[string][]string{
				"997": {"functional acknowledgment", "functional acknowledgement"},
				"999": {"implementation acknowledgment", "implementation acknowledgement"},
				AckNone: {
					"not required",
					"no acknowledgment",
					"no acknowledgement",
					"not expected",
					"do not send",
				},
			},
		},
		{
			Key:       prefix + SetAttrAckWindowMinutes,
			Label:     name + " acknowledgment window",
			Scope:     ScopeSet,
			Normalize: normalizeMinutes,
			matches:   matchesMinutes,
		},
	}
}

func controlNumberFields() []*Spec {
	fields := make([]*Spec, 0, len(ControlNumberKinds))
	for _, kind := range ControlNumberKinds {
		fields = append(fields, &Spec{
			Key:       ControlNumberField(kind),
			Label:     fmt.Sprintf("First %s control number", strings.ToLower(string(kind))),
			Scope:     ScopeControlNumber,
			Normalize: normalizeControlNumber,
			matches:   matchesNumber,
		})
	}
	return fields
}

func setFieldPrefix(
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
) string {
	return fmt.Sprintf("sets.%s.%s.", set, strings.ToLower(string(direction)))
}

func SetField(
	set edidocument.TransactionSet,
	direction edidocument.DocumentDirection,
	attr string,
) string {
	return setFieldPrefix(set, direction) + attr
}

func ControlNumberField(kind edidocument.ControlNumberKind) string {
	return fmt.Sprintf(
		"controlNumbers.%s.%s",
		strings.ToLower(string(kind)),
		controlNumberAttrStart,
	)
}

func Keys() []string {
	keys := make([]string, 0, len(Catalog))
	for key := range Catalog {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (s *Spec) UsedBy(method edi.ConnectionMethod) bool {
	return len(s.methods) == 0 || slices.Contains(s.methods, method)
}

func signingAlgorithmField() *Spec {
	return &Spec{
		Key:       "as2.signingAlgorithm",
		Label:     "Signing algorithm",
		Scope:     ScopeTransport,
		ConfigKey: editransport.ConfigKeySigningAlgorithm,
		methods:   as2Only,
		Normalize: as2Algorithms(
			smime.SigningAlgorithmSHA1, smime.SigningAlgorithmSHA256,
			smime.SigningAlgorithmSHA384, smime.SigningAlgorithmSHA512,
		),
		spelled: map[string][]string{
			smime.SigningAlgorithmSHA1:   {"sha-1"},
			smime.SigningAlgorithmSHA256: {"sha-256", "sha 256"},
			smime.SigningAlgorithmSHA384: {"sha-384"},
			smime.SigningAlgorithmSHA512: {"sha-512"},
		},
	}
}

func encryptionAlgorithmField() *Spec {
	return &Spec{
		Key:       "as2.encryptionAlgorithm",
		Label:     "Encryption algorithm",
		Scope:     ScopeTransport,
		ConfigKey: editransport.ConfigKeyEncryptionAlgorithm,
		methods:   as2Only,
		Normalize: as2Algorithms(
			smime.EncryptionAlgorithmTripleDES, smime.EncryptionAlgorithmAES128CBC,
			smime.EncryptionAlgorithmAES256CBC, smime.EncryptionAlgorithmAES128GCM,
			smime.EncryptionAlgorithmAES256GCM,
		),
		spelled: map[string][]string{
			smime.EncryptionAlgorithmTripleDES: {"triple des", "3des", "des3"},
			smime.EncryptionAlgorithmAES128CBC: {"aes 128", "aes-128", "aes128"},
			smime.EncryptionAlgorithmAES256CBC: {"aes 256", "aes-256", "aes256"},
			smime.EncryptionAlgorithmAES128GCM: {"aes-128-gcm", "aes 128 gcm"},
			smime.EncryptionAlgorithmAES256GCM: {"aes-256-gcm", "aes 256 gcm"},
		},
	}
}
