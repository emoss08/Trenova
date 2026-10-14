package ediconnectiondraft

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/as2"
)

type fieldScope int

const (
	scopePartner fieldScope = iota
	scopeTransport
	scopeInterchange
	scopeSet
	scopeControlNumber
	scopeCarrier
)

type normalizer func(string) (string, error)

type quoteMatcher func(value, quote string) bool

type fieldSpec struct {
	key       string
	label     string
	scope     fieldScope
	configKey string
	methods   []edi.ConnectionMethod
	required  bool
	normalize normalizer
	spelled   map[string][]string
	matches   quoteMatcher
	fallback  string
}

const (
	fieldPartnerCode         = "partner.code"
	fieldPartnerName         = "partner.name"
	fieldPartnerContactName  = "partner.contactName"
	fieldPartnerContactEmail = "partner.contactEmail"
	fieldPartnerContactPhone = "partner.contactPhone"
	fieldPartnerTimezone     = "partner.timezone"
	fieldPartnerCountry      = "partner.country"

	fieldTransportMethod      = "transport.method"
	fieldTransportEnvironment = "transport.environment"

	fieldPartnerQualifier = "isa.partnerQualifier"
	fieldPartnerID        = "isa.partnerId"
	fieldOurQualifier     = "isa.ourQualifier"
	fieldOurID            = "isa.ourId"
	fieldPartnerGSID      = "gs.partnerId"
	fieldOurGSID          = "gs.ourId"
	fieldUsageIndicator   = "isa.usageIndicator"
	fieldX12Version       = "x12.version"
	fieldElementSep       = "separators.element"
	fieldSegmentSep       = "separators.segment"
	fieldComponentSep     = "separators.component"
	fieldRepetitionSep    = "separators.repetition"

	fieldCarrierSCAC = "carrier.scac"

	setAttrSupported         = "supported"
	setAttrVersion           = "version"
	setAttrAck               = "ack"
	setAttrAckWindowMinutes  = "ackWindowMinutes"
	setAttrFunctionalGroupID = "functionalGroupId"

	controlNumberAttrStart = "start"

	ackNone = "None"
)

var (
	errBlank           = errors.New("the value is blank")
	isaQualifierRe     = regexp.MustCompile(`^[A-Z0-9]{2}$`)
	versionDigitsRe    = regexp.MustCompile(`^\d{4,6}$`)
	scacRe             = regexp.MustCompile(`^[A-Z0-9]{2,4}$`)
	functionalIDRe     = regexp.MustCompile(`^[A-Z]{2}$`)
	numberRe           = regexp.MustCompile(`\d+`)
	durationRe         = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(minutes?|mins?|hours?|hrs?|days?|business days?)`)
	countryRe          = regexp.MustCompile(`^[A-Z]{2}$`)
	externalMethods    = []edi.ConnectionMethod{edi.ConnectionMethodAS2, edi.ConnectionMethodSFTP, edi.ConnectionMethodVAN}
	fileMethods        = []edi.ConnectionMethod{edi.ConnectionMethodSFTP, edi.ConnectionMethodVAN}
	as2Only            = []edi.ConnectionMethod{edi.ConnectionMethodAS2}
	vanOnly            = []edi.ConnectionMethod{edi.ConnectionMethodVAN}
	supportedSets      = []edi.TransactionSet{edi.TransactionSet204, edi.TransactionSet210, edi.TransactionSet214, edi.TransactionSet990}
	directions         = []edi.DocumentDirection{edi.DocumentDirectionInbound, edi.DocumentDirectionOutbound}
	controlNumberKinds = []edi.ControlNumberKind{
		edi.ControlNumberKindInterchange,
		edi.ControlNumberKindGroup,
		edi.ControlNumberKindTransaction,
	}
)

var secretWords = []string{
	"password", "passphrase", "privatekey", "private_key", "secret", "token",
	"credential", "certificate", "knownhostkey", "hostkey", "apikey", "authorization",
	"security",
}

func isSecretField(key string) bool {
	compact := strings.ToLower(strings.NewReplacer(".", "", "-", "", " ", "").Replace(key))
	for _, word := range secretWords {
		if strings.Contains(compact, word) {
			return true
		}
	}
	return false
}

func text(maxLength int) normalizer {
	return func(value string) (string, error) {
		value = strings.Join(strings.Fields(value), " ")
		if value == "" {
			return "", errBlank
		}
		if len(value) > maxLength {
			return "", fmt.Errorf("must be at most %d characters", maxLength)
		}
		return value, nil
	}
}

func choice(allowed map[string]string) normalizer {
	return func(value string) (string, error) {
		key := strings.ToLower(strings.Join(strings.Fields(value), " "))
		if key == "" {
			return "", errBlank
		}
		if normalized, ok := allowed[key]; ok {
			return normalized, nil
		}
		options := make([]string, 0, len(allowed))
		for _, normalized := range allowed {
			if !slices.Contains(options, normalized) {
				options = append(options, normalized)
			}
		}
		slices.Sort(options)
		return "", fmt.Errorf("must be one of %s", strings.Join(options, ", "))
	}
}

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return "", errors.New("must be an email address")
	}
	return value, nil
}

func normalizeTimezone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	if _, err := time.LoadLocation(value); err != nil || !strings.Contains(value, "/") {
		return "", errors.New("must be an IANA time zone such as America/Chicago")
	}
	return value, nil
}

func normalizeCountry(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !countryRe.MatchString(value) {
		return "", errors.New("must be a two-letter country code")
	}
	return value, nil
}

func normalizeQualifier(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !isaQualifierRe.MatchString(value) {
		return "", errors.New("must be a two-character ISA qualifier")
	}
	return value, nil
}

func normalizeISAID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	if len(value) > 15 {
		return "", errors.New("must be at most 15 characters")
	}
	return value, nil
}

func normalizeGSID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || len(value) > 15 {
		return "", errors.New("must be 2 to 15 characters")
	}
	return value, nil
}

func normalizeVersion(value string) (string, error) {
	digits := strings.TrimSpace(value)
	if !versionDigitsRe.MatchString(digits) {
		return "", errors.New("must be an X12 version such as 004010")
	}
	switch len(digits) {
	case 4:
		digits = "00" + digits
	case 5:
		digits += "0"
	}
	return digits, nil
}

func versionCore(version string) string {
	normalized, err := normalizeVersion(version)
	if err != nil {
		return ""
	}
	return strings.TrimLeft(normalized, "0")[:3]
}

func matchesVersion(value, quote string) bool {
	core := versionCore(value)
	if core == "" {
		return false
	}
	for _, found := range numberRe.FindAllString(quote, -1) {
		if versionDigitsRe.MatchString(found) && versionCore(found) == core {
			return true
		}
	}
	return false
}

var separatorNames = map[string]string{
	"asterisk": "*", "star": "*", "tilde": "~", "greater than": ">", "greater-than": ">",
	"caret": "^", "pipe": "|", "vertical bar": "|", "colon": ":", "plus": "+",
	"exclamation": "!", "exclamation mark": "!", "back tick": "`", "backtick": "`",
	"new line": "\n", "newline": "\n", "line feed": "\n", "carriage return": "\n",
	"crlf": "\n", "apostrophe": "'", "single quote": "'", "comma": ",",
}

func normalizeSeparator(value string) (string, error) {
	if value == "\n" || value == "\r\n" {
		return "\n", nil
	}
	trimmed := strings.TrimSpace(value)
	if named, ok := separatorNames[strings.ToLower(trimmed)]; ok {
		return named, nil
	}
	if len([]rune(trimmed)) != 1 {
		return "", errors.New("must be a single character")
	}
	return trimmed, nil
}

func spelledSeparators() map[string][]string {
	spelled := make(map[string][]string, len(separatorNames))
	for name, char := range separatorNames {
		spelled[char] = append(spelled[char], name)
	}
	return spelled
}

func normalizeURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", errors.New("must be an absolute http or https URL")
	}
	return value, nil
}

func normalizePort(value string) (string, error) {
	value = strings.TrimSpace(value)
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("must be a port number between 1 and 65535")
	}
	return strconv.Itoa(port), nil
}

func normalizePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	if len(value) > 255 || strings.Contains(value, "..") {
		return "", errors.New("must be a directory path of at most 255 characters without ..")
	}
	return value, nil
}

func normalizeSCAC(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !scacRe.MatchString(value) {
		return "", errors.New("must be a SCAC of 2 to 4 letters or digits")
	}
	return value, nil
}

func normalizeFunctionalGroup(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !functionalIDRe.MatchString(value) {
		return "", errors.New("must be a two-letter functional identifier code")
	}
	return value, nil
}

func normalizeMinutes(value string) (string, error) {
	value = strings.TrimSpace(value)
	if minutes, err := strconv.Atoi(value); err == nil {
		if minutes < 1 || minutes > 60*24*30 {
			return "", errors.New("must be between 1 minute and 30 days")
		}
		return strconv.Itoa(minutes), nil
	}
	durations := durationMinutes(value)
	if len(durations) != 1 {
		return "", errors.New("must be a number of minutes")
	}
	return normalizeMinutes(strconv.Itoa(durations[0]))
}

func durationMinutes(text string) []int {
	matches := durationRe.FindAllStringSubmatch(text, -1)
	out := make([]int, 0, len(matches))
	for _, match := range matches {
		amount, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		unit := strings.ToLower(match[2])
		switch {
		case strings.HasPrefix(unit, "min"):
		case strings.HasPrefix(unit, "h"):
			amount *= 60
		case strings.HasPrefix(unit, "business"):
			amount *= 60 * 24
		case strings.HasPrefix(unit, "day"):
			amount *= 60 * 24
		}
		out = append(out, int(amount))
	}
	return out
}

func matchesMinutes(value, quote string) bool {
	minutes, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	for _, found := range durationMinutes(quote) {
		if found == minutes {
			return true
		}
	}
	return false
}

func normalizeControlNumber(value string) (string, error) {
	value = strings.TrimLeft(strings.TrimSpace(value), "0")
	if value == "" {
		value = "0"
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number < 1 || number > 999999999 {
		return "", errors.New("must be a whole number between 1 and 999999999")
	}
	return strconv.FormatInt(number, 10), nil
}

func matchesNumber(value, quote string) bool {
	for _, found := range numberRe.FindAllString(quote, -1) {
		if strings.TrimLeft(found, "0") == strings.TrimLeft(value, "0") {
			return true
		}
	}
	return false
}

func anyQuote(string, string) bool { return true }

func as2Algorithms(values ...string) normalizer {
	allowed := make(map[string]string, len(values))
	for _, value := range values {
		allowed[value] = value
	}
	return choice(allowed)
}

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

func buildCatalog() map[string]fieldSpec {
	specs := []fieldSpec{
		{key: fieldPartnerCode, label: "Partner code", scope: scopePartner, required: true, normalize: text(100)},
		{key: fieldPartnerName, label: "Partner name", scope: scopePartner, required: true, normalize: text(200)},
		{key: fieldPartnerContactName, label: "Contact name", scope: scopePartner, normalize: text(150)},
		{key: fieldPartnerContactEmail, label: "Contact email", scope: scopePartner, normalize: normalizeEmail},
		{key: fieldPartnerContactPhone, label: "Contact phone", scope: scopePartner, normalize: text(30)},
		{key: fieldPartnerTimezone, label: "Time zone", scope: scopePartner, normalize: normalizeTimezone},
		{
			key: fieldPartnerCountry, label: "Country", scope: scopePartner,
			normalize: normalizeCountry, fallback: "US",
		},
		{
			key: fieldTransportMethod, label: "Connection method", scope: scopeTransport, required: true,
			normalize: choice(map[string]string{
				"as2": "AS2", "applicability statement 2": "AS2", "sftp": "SFTP",
				"secure ftp": "SFTP", "ssh file transfer": "SFTP", "van": "VAN",
				"value added network": "VAN", "value-added network": "VAN",
			}),
			spelled: map[string][]string{
				"AS2":  {"as2", "applicability statement 2"},
				"SFTP": {"sftp", "secure ftp", "ssh file transfer"},
				"VAN":  {"van", "value added network", "value-added network"},
			},
		},
		{
			key: fieldTransportEnvironment, label: "Environment", scope: scopeTransport,
			configKey: "environment", methods: externalMethods, required: true, fallback: "test",
			normalize: choice(map[string]string{
				"test": "test", "testing": "test", "production": "production", "prod": "production",
				"live": "production",
			}),
			spelled: map[string][]string{"test": {"test"}, "production": {"production", "prod", "live"}},
		},
		{key: "as2.localAs2Id", label: "Your AS2 ID", scope: scopeTransport, configKey: editransport.ConfigKeyLocalAS2ID, methods: as2Only, required: true, normalize: text(128)},
		{key: "as2.partnerAs2Id", label: "Partner AS2 ID", scope: scopeTransport, configKey: editransport.ConfigKeyPartnerAS2ID, methods: as2Only, required: true, normalize: text(128)},
		{key: "as2.endpointUrl", label: "AS2 endpoint URL", scope: scopeTransport, configKey: editransport.ConfigKeyEndpointURL, methods: as2Only, required: true, normalize: normalizeURL},
		{
			key: "as2.mdnMode", label: "MDN mode", scope: scopeTransport,
			configKey: editransport.ConfigKeyMDNMode, methods: as2Only, required: true,
			normalize: choice(map[string]string{
				"sync": "sync", "synchronous": "sync", "async": "async", "asynchronous": "async",
			}),
			spelled: map[string][]string{"sync": {"synchronous", "sync"}, "async": {"asynchronous", "async"}},
		},
		{key: "as2.mdnUrl", label: "Async MDN return URL", scope: scopeTransport, configKey: editransport.ConfigKeyMDNURL, methods: as2Only, normalize: normalizeURL},
		{
			key: "as2.signingAlgorithm", label: "Signing algorithm", scope: scopeTransport,
			configKey: editransport.ConfigKeySigningAlgorithm, methods: as2Only,
			normalize: as2Algorithms(as2.SigningAlgorithmSHA1, as2.SigningAlgorithmSHA256, as2.SigningAlgorithmSHA384, as2.SigningAlgorithmSHA512),
			spelled: map[string][]string{
				as2.SigningAlgorithmSHA1: {"sha-1"}, as2.SigningAlgorithmSHA256: {"sha-256", "sha 256"},
				as2.SigningAlgorithmSHA384: {"sha-384"}, as2.SigningAlgorithmSHA512: {"sha-512"},
			},
		},
		{
			key: "as2.encryptionAlgorithm", label: "Encryption algorithm", scope: scopeTransport,
			configKey: editransport.ConfigKeyEncryptionAlgorithm, methods: as2Only,
			normalize: as2Algorithms(as2.EncryptionAlgorithmTripleDES, as2.EncryptionAlgorithmAES128CBC, as2.EncryptionAlgorithmAES256CBC, as2.EncryptionAlgorithmAES128GCM, as2.EncryptionAlgorithmAES256GCM),
			spelled: map[string][]string{
				as2.EncryptionAlgorithmTripleDES: {"triple des", "3des", "des3"},
				as2.EncryptionAlgorithmAES128CBC: {"aes 128", "aes-128", "aes128"},
				as2.EncryptionAlgorithmAES256CBC: {"aes 256", "aes-256", "aes256"},
				as2.EncryptionAlgorithmAES128GCM: {"aes-128-gcm", "aes 128 gcm"},
				as2.EncryptionAlgorithmAES256GCM: {"aes-256-gcm", "aes 256 gcm"},
			},
		},
		{
			key: "as2.compressionAlgorithm", label: "Compression", scope: scopeTransport,
			configKey: editransport.ConfigKeyCompressionAlgorithm, methods: as2Only,
			normalize: choice(map[string]string{"none": "none", "zlib": editransport.CompressionZlib}),
			spelled:   map[string][]string{"none": {"no compression", "uncompressed"}},
		},
		{key: "as2.basicAuthUsername", label: "Basic auth username", scope: scopeTransport, configKey: editransport.ConfigKeyBasicAuthUsername, methods: as2Only, normalize: text(128)},
		{key: "transport.host", label: "Host", scope: scopeTransport, configKey: "host", methods: fileMethods, required: true, normalize: text(255)},
		{key: "transport.port", label: "Port", scope: scopeTransport, configKey: "port", methods: fileMethods, required: true, normalize: normalizePort, fallback: "22", matches: matchesNumber},
		{key: "transport.username", label: "Username", scope: scopeTransport, configKey: "username", methods: fileMethods, required: true, normalize: text(128)},
		{
			key: "transport.authMode", label: "Authentication", scope: scopeTransport,
			configKey: "authMode", methods: fileMethods, required: true,
			normalize: choice(map[string]string{
				"password": "password", "privatekey": "privateKey", "private key": "privateKey",
				"ssh key": "privateKey", "public key": "privateKey", "key": "privateKey",
			}),
			spelled: map[string][]string{
				"password":   {"password"},
				"privateKey": {"private key", "ssh key", "public key", "key-based", "key based"},
			},
		},
		{key: "transport.inboundDirectory", label: "Inbound directory", scope: scopeTransport, configKey: "inboundDirectory", methods: fileMethods, normalize: normalizePath},
		{key: "transport.outboundDirectory", label: "Outbound directory", scope: scopeTransport, configKey: "outboundDirectory", methods: fileMethods, normalize: normalizePath},
		{key: "transport.archiveDirectory", label: "Archive directory", scope: scopeTransport, configKey: "archiveDirectory", methods: fileMethods, normalize: normalizePath},
		{key: "transport.fileNamingPattern", label: "File naming pattern", scope: scopeTransport, configKey: "fileNamingPattern", methods: fileMethods, normalize: text(255)},
		{key: "van.providerName", label: "VAN provider", scope: scopeTransport, configKey: "providerName", methods: vanOnly, required: true, normalize: text(128)},
		{key: "van.mailboxId", label: "VAN mailbox ID", scope: scopeTransport, configKey: "mailboxId", methods: vanOnly, required: true, normalize: text(128)},
		{key: fieldPartnerQualifier, label: "Partner ISA qualifier", scope: scopeInterchange, required: true, normalize: normalizeQualifier, spelled: qualifierNames},
		{key: fieldPartnerID, label: "Partner ISA ID", scope: scopeInterchange, required: true, normalize: normalizeISAID},
		{key: fieldOurQualifier, label: "Your ISA qualifier", scope: scopeInterchange, required: true, normalize: normalizeQualifier, spelled: qualifierNames},
		{key: fieldOurID, label: "Your ISA ID", scope: scopeInterchange, required: true, normalize: normalizeISAID},
		{key: fieldPartnerGSID, label: "Partner GS ID", scope: scopeInterchange, normalize: normalizeGSID},
		{key: fieldOurGSID, label: "Your GS ID", scope: scopeInterchange, normalize: normalizeGSID},
		{
			key: fieldUsageIndicator, label: "Usage indicator", scope: scopeInterchange, required: true,
			fallback:  "T",
			normalize: choice(map[string]string{"t": "T", "test": "T", "p": "P", "production": "P"}),
			spelled:   map[string][]string{"T": {"test"}, "P": {"production"}},
		},
		{key: fieldX12Version, label: "X12 version", scope: scopeInterchange, required: true, normalize: normalizeVersion, matches: matchesVersion},
		{key: fieldElementSep, label: "Element separator", scope: scopeInterchange, required: true, normalize: normalizeSeparator, spelled: spelledSeparators(), fallback: "*"},
		{key: fieldSegmentSep, label: "Segment terminator", scope: scopeInterchange, required: true, normalize: normalizeSeparator, spelled: spelledSeparators(), fallback: "~"},
		{key: fieldComponentSep, label: "Component separator", scope: scopeInterchange, required: true, normalize: normalizeSeparator, spelled: spelledSeparators(), fallback: ">"},
		{key: fieldRepetitionSep, label: "Repetition separator", scope: scopeInterchange, required: true, normalize: normalizeSeparator, spelled: spelledSeparators(), fallback: "^"},
		{key: fieldCarrierSCAC, label: "Your SCAC", scope: scopeCarrier, normalize: normalizeSCAC},
	}
	for _, set := range supportedSets {
		for _, direction := range directions {
			prefix := setFieldPrefix(set, direction)
			name := fmt.Sprintf("%s %s", set, strings.ToLower(string(direction)))
			specs = append(specs,
				fieldSpec{
					key: prefix + setAttrSupported, label: name + " exchanged", scope: scopeSet,
					normalize: choice(map[string]string{"true": "true", "yes": "true", "false": "false", "no": "false"}),
					matches:   anyQuote,
				},
				fieldSpec{key: prefix + setAttrVersion, label: name + " version", scope: scopeSet, normalize: normalizeVersion, matches: matchesVersion},
				fieldSpec{
					key: prefix + setAttrAck, label: name + " functional acknowledgment", scope: scopeSet,
					normalize: choice(map[string]string{
						"997": "997", "999": "999", "none": ackNone, "no": ackNone, "not required": ackNone,
					}),
					spelled: map[string][]string{
						"997":   {"functional acknowledgment", "functional acknowledgement"},
						"999":   {"implementation acknowledgment", "implementation acknowledgement"},
						ackNone: {"not required", "no acknowledgment", "no acknowledgement", "not expected", "do not send"},
					},
				},
				fieldSpec{key: prefix + setAttrAckWindowMinutes, label: name + " acknowledgment window", scope: scopeSet, normalize: normalizeMinutes, matches: matchesMinutes},
				fieldSpec{key: prefix + setAttrFunctionalGroupID, label: name + " functional group", scope: scopeSet, normalize: normalizeFunctionalGroup},
			)
		}
	}
	for _, kind := range controlNumberKinds {
		specs = append(specs, fieldSpec{
			key:       controlNumberField(kind),
			label:     fmt.Sprintf("First %s control number", strings.ToLower(string(kind))),
			scope:     scopeControlNumber,
			normalize: normalizeControlNumber,
			matches:   matchesNumber,
		})
	}

	catalog := make(map[string]fieldSpec, len(specs))
	for _, spec := range specs {
		catalog[spec.key] = spec
	}
	return catalog
}

func setFieldPrefix(set edi.TransactionSet, direction edi.DocumentDirection) string {
	return fmt.Sprintf("sets.%s.%s.", set, strings.ToLower(string(direction)))
}

func setField(set edi.TransactionSet, direction edi.DocumentDirection, attr string) string {
	return setFieldPrefix(set, direction) + attr
}

func controlNumberField(kind edi.ControlNumberKind) string {
	return fmt.Sprintf("controlNumbers.%s.%s", strings.ToLower(string(kind)), controlNumberAttrStart)
}

func FieldKeys() []string {
	keys := make([]string, 0, len(fieldCatalog))
	for key := range fieldCatalog {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (s fieldSpec) usedBy(method edi.ConnectionMethod) bool {
	return len(s.methods) == 0 || slices.Contains(s.methods, method)
}
