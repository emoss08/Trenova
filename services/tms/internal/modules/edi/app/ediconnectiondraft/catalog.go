package ediconnectiondraft

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/as2"
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

var fieldCatalog = buildCatalog()

func buildCatalog() map[string]*fieldSpec {
	groups := [][]*fieldSpec{
		partnerFields(),
		transportFields(),
		as2Fields(),
		fileTransferFields(),
		interchangeFields(),
		setFields(),
		controlNumberFields(),
	}
	catalog := make(map[string]*fieldSpec)
	for _, group := range groups {
		for _, spec := range group {
			if len(spec.spelled) > 0 {
				spec.normalize = spelledOut(spec.normalize, spec.spelled)
			}
			catalog[spec.key] = spec
		}
	}
	return catalog
}

func partnerFields() []*fieldSpec {
	return []*fieldSpec{
		{
			key:       fieldPartnerCode,
			label:     "Partner code",
			scope:     scopePartner,
			required:  true,
			normalize: text(100),
		},
		{
			key:       fieldPartnerName,
			label:     "Partner name",
			scope:     scopePartner,
			required:  true,
			normalize: text(200),
		},
		{
			key:       fieldPartnerContactName,
			label:     "Contact name",
			scope:     scopePartner,
			normalize: text(150),
		},
		{
			key:       fieldPartnerContactEmail,
			label:     "Contact email",
			scope:     scopePartner,
			normalize: normalizeEmail,
		},
		{
			key:       fieldPartnerContactPhone,
			label:     "Contact phone",
			scope:     scopePartner,
			normalize: text(30),
		},
		{
			key:       fieldPartnerTimezone,
			label:     "Time zone",
			scope:     scopePartner,
			normalize: normalizeTimezone,
		},
		{
			key:       fieldPartnerCountry,
			label:     "Country",
			scope:     scopePartner,
			normalize: normalizeCountry,
			fallback:  "US",
		},
	}
}

func transportFields() []*fieldSpec {
	return []*fieldSpec{
		{
			key:      fieldTransportMethod,
			label:    "Connection method",
			scope:    scopeTransport,
			required: true,
			normalize: choice(map[string]string{
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
			key:       fieldTransportEnvironment,
			label:     "Environment",
			scope:     scopeTransport,
			configKey: "environment",
			methods:   externalMethods,
			required:  true,
			fallback:  valueTest,
			normalize: choice(map[string]string{
				valueTest: valueTest, "testing": valueTest, valueProduction: valueProduction,
				"prod": valueProduction, "live": valueProduction,
			}),
			spelled: map[string][]string{
				valueTest:       {valueTest},
				valueProduction: {valueProduction, "prod", "live"},
			},
		},
	}
}

func transportText(
	key, label, configKey string,
	methods []edi.ConnectionMethod,
	required bool,
) *fieldSpec {
	return &fieldSpec{
		key:       key,
		label:     label,
		scope:     scopeTransport,
		configKey: configKey,
		methods:   methods,
		required:  required,
		normalize: text(255),
	}
}

func as2Fields() []*fieldSpec {
	endpoint := transportText(
		"as2.endpointUrl",
		"AS2 endpoint URL",
		editransport.ConfigKeyEndpointURL,
		as2Only,
		true,
	)
	endpoint.normalize = normalizeURL
	mdnURL := transportText(
		"as2.mdnUrl",
		"Async MDN return URL",
		editransport.ConfigKeyMDNURL,
		as2Only,
		false,
	)
	mdnURL.normalize = normalizeURL

	return []*fieldSpec{
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
			key:       "as2.mdnMode",
			label:     "MDN mode",
			scope:     scopeTransport,
			configKey: editransport.ConfigKeyMDNMode,
			methods:   as2Only,
			required:  true,
			normalize: choice(map[string]string{
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
			key:       "as2.compressionAlgorithm",
			label:     "Compression",
			scope:     scopeTransport,
			configKey: editransport.ConfigKeyCompressionAlgorithm,
			methods:   as2Only,
			normalize: choice(
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

func fileTransferFields() []*fieldSpec {
	port := transportText("transport.port", "Port", "port", fileMethods, true)
	port.normalize = normalizePort
	port.fallback = "22"
	port.matches = matchesNumber

	directories := make([]*fieldSpec, 0, 3)
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
		spec.normalize = normalizePath
		directories = append(directories, spec)
	}

	fields := []*fieldSpec{
		transportText("transport.host", "Host", "host", fileMethods, true),
		port,
		transportText("transport.username", "Username", "username", fileMethods, true),
		{
			key:       "transport.authMode",
			label:     "Authentication",
			scope:     scopeTransport,
			configKey: "authMode",
			methods:   fileMethods,
			required:  true,
			normalize: choice(map[string]string{
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

func separatorField(key, label, fallback string) *fieldSpec {
	return &fieldSpec{
		key:       key,
		label:     label,
		scope:     scopeInterchange,
		required:  true,
		normalize: normalizeSeparator,
		spelled:   spelledSeparators(),
		fallback:  fallback,
	}
}

func interchangeFields() []*fieldSpec {
	return []*fieldSpec{
		{
			key: fieldPartnerQualifier, label: "Partner ISA qualifier", scope: scopeInterchange,
			required: true, normalize: normalizeQualifier, spelled: qualifierNames,
		},
		{
			key:       fieldPartnerID,
			label:     "Partner ISA ID",
			scope:     scopeInterchange,
			required:  true,
			normalize: normalizeISAID,
		},
		{
			key: fieldOurQualifier, label: "Your ISA qualifier", scope: scopeInterchange,
			required: true, normalize: normalizeQualifier, spelled: qualifierNames,
		},
		{
			key:       fieldOurID,
			label:     "Your ISA ID",
			scope:     scopeInterchange,
			required:  true,
			normalize: normalizeISAID,
		},
		{
			key:       fieldPartnerGSID,
			label:     "Partner GS ID",
			scope:     scopeInterchange,
			normalize: normalizeGSID,
		},
		{key: fieldOurGSID, label: "Your GS ID", scope: scopeInterchange, normalize: normalizeGSID},
		{
			key:      fieldUsageIndicator,
			label:    "Usage indicator",
			scope:    scopeInterchange,
			required: true,
			fallback: "T",
			normalize: choice(
				map[string]string{"t": "T", valueTest: "T", "p": "P", valueProduction: "P"},
			),
			spelled: map[string][]string{"T": {valueTest}, "P": {valueProduction}},
		},
		{
			key: fieldX12Version, label: "X12 version", scope: scopeInterchange,
			required: true, normalize: normalizeVersion, matches: matchesVersion,
		},
		separatorField(fieldElementSep, "Element separator", "*"),
		separatorField(fieldSegmentSep, "Segment terminator", "~"),
		separatorField(fieldComponentSep, "Component separator", ">"),
		separatorField(fieldRepetitionSep, "Repetition separator", "^"),
	}
}

func setFields() []*fieldSpec {
	fields := make([]*fieldSpec, 0, len(supportedSets)*len(directions)*5)
	for _, set := range supportedSets {
		for _, direction := range directions {
			fields = append(fields, setFieldsFor(set, direction)...)
		}
	}
	return fields
}

func setFieldsFor(set edi.TransactionSet, direction edi.DocumentDirection) []*fieldSpec {
	prefix := setFieldPrefix(set, direction)
	name := fmt.Sprintf("%s %s", set, strings.ToLower(string(direction)))
	return []*fieldSpec{
		{
			key:   prefix + setAttrSupported,
			label: name + " exchanged",
			scope: scopeSet,
			normalize: choice(map[string]string{
				valueTrue: valueTrue, "yes": valueTrue, valueFalse: valueFalse, "no": valueFalse,
			}),
			matches: anyQuote,
		},
		{
			key: prefix + setAttrVersion, label: name + " version", scope: scopeSet,
			normalize: normalizeVersion, matches: matchesVersion,
		},
		{
			key:   prefix + setAttrAck,
			label: name + " functional acknowledgment",
			scope: scopeSet,
			normalize: choice(map[string]string{
				"997":          "997",
				"999":          "999",
				valueNone:      ackNone,
				"no":           ackNone,
				"not required": ackNone,
			}),
			spelled: map[string][]string{
				"997": {"functional acknowledgment", "functional acknowledgement"},
				"999": {"implementation acknowledgment", "implementation acknowledgement"},
				ackNone: {
					"not required",
					"no acknowledgment",
					"no acknowledgement",
					"not expected",
					"do not send",
				},
			},
		},
		{
			key:       prefix + setAttrAckWindowMinutes,
			label:     name + " acknowledgment window",
			scope:     scopeSet,
			normalize: normalizeMinutes,
			matches:   matchesMinutes,
		},
	}
}

func controlNumberFields() []*fieldSpec {
	fields := make([]*fieldSpec, 0, len(controlNumberKinds))
	for _, kind := range controlNumberKinds {
		fields = append(fields, &fieldSpec{
			key:       controlNumberField(kind),
			label:     fmt.Sprintf("First %s control number", strings.ToLower(string(kind))),
			scope:     scopeControlNumber,
			normalize: normalizeControlNumber,
			matches:   matchesNumber,
		})
	}
	return fields
}

func setFieldPrefix(set edi.TransactionSet, direction edi.DocumentDirection) string {
	return fmt.Sprintf("sets.%s.%s.", set, strings.ToLower(string(direction)))
}

func setField(set edi.TransactionSet, direction edi.DocumentDirection, attr string) string {
	return setFieldPrefix(set, direction) + attr
}

func controlNumberField(kind edi.ControlNumberKind) string {
	return fmt.Sprintf(
		"controlNumbers.%s.%s",
		strings.ToLower(string(kind)),
		controlNumberAttrStart,
	)
}

func FieldKeys() []string {
	keys := make([]string, 0, len(fieldCatalog))
	for key := range fieldCatalog {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (s *fieldSpec) usedBy(method edi.ConnectionMethod) bool {
	return len(s.methods) == 0 || slices.Contains(s.methods, method)
}

func signingAlgorithmField() *fieldSpec {
	return &fieldSpec{
		key:       "as2.signingAlgorithm",
		label:     "Signing algorithm",
		scope:     scopeTransport,
		configKey: editransport.ConfigKeySigningAlgorithm,
		methods:   as2Only,
		normalize: as2Algorithms(
			as2.SigningAlgorithmSHA1, as2.SigningAlgorithmSHA256,
			as2.SigningAlgorithmSHA384, as2.SigningAlgorithmSHA512,
		),
		spelled: map[string][]string{
			as2.SigningAlgorithmSHA1:   {"sha-1"},
			as2.SigningAlgorithmSHA256: {"sha-256", "sha 256"},
			as2.SigningAlgorithmSHA384: {"sha-384"},
			as2.SigningAlgorithmSHA512: {"sha-512"},
		},
	}
}

func encryptionAlgorithmField() *fieldSpec {
	return &fieldSpec{
		key:       "as2.encryptionAlgorithm",
		label:     "Encryption algorithm",
		scope:     scopeTransport,
		configKey: editransport.ConfigKeyEncryptionAlgorithm,
		methods:   as2Only,
		normalize: as2Algorithms(
			as2.EncryptionAlgorithmTripleDES, as2.EncryptionAlgorithmAES128CBC,
			as2.EncryptionAlgorithmAES256CBC, as2.EncryptionAlgorithmAES128GCM,
			as2.EncryptionAlgorithmAES256GCM,
		),
		spelled: map[string][]string{
			as2.EncryptionAlgorithmTripleDES: {"triple des", "3des", "des3"},
			as2.EncryptionAlgorithmAES128CBC: {"aes 128", "aes-128", "aes128"},
			as2.EncryptionAlgorithmAES256CBC: {"aes 256", "aes-256", "aes256"},
			as2.EncryptionAlgorithmAES128GCM: {"aes-128-gcm", "aes 128 gcm"},
			as2.EncryptionAlgorithmAES256GCM: {"aes-256-gcm", "aes 256 gcm"},
		},
	}
}
