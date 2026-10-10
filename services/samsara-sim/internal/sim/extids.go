package sim

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	externalIDSeparator         = ":"
	samsaraExternalIDPrefix     = "samsara."
	externalIDVinKey            = "samsara.vin"
	externalIDSerialKey         = "samsara.serial"
	externalIDTagNameKey        = "samsara.name"
	maxExternalIDKeyLength      = 32
	maxExternalIDKeysPerType    = 30
	fieldExternalIDs            = "externalIds"
	externalIDValueAllowedChars = "@._%+-"
)

type recordRef struct {
	Raw   string
	Key   string
	Value string
}

type autoExternalIDs func(record Record) map[string]string

func parseRecordRef(raw string) recordRef {
	clean := strings.TrimSpace(raw)
	key, value, found := strings.Cut(clean, externalIDSeparator)
	if !found {
		return recordRef{Raw: clean}
	}
	return recordRef{Raw: clean, Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)}
}

func (r recordRef) isExternal() bool {
	return r.Key != ""
}

func externalIDsOf(record Record) map[string]string {
	raw, ok := anyAsMap(record[fieldExternalIDs])
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		if text, isString := value.(string); isString {
			out[key] = text
		}
	}
	return out
}

func (r recordRef) matches(record Record, auto autoExternalIDs) bool {
	if !r.isExternal() {
		return r.Raw != "" && recordID(record) == r.Raw
	}
	if value, ok := externalIDsOf(record)[r.Key]; ok && value == r.Value {
		return true
	}
	if auto == nil {
		return false
	}
	value, ok := auto(record)[r.Key]
	return ok && value != "" && value == r.Value
}

func findRecordByRef(
	records []Record,
	raw string,
	auto autoExternalIDs,
) (record Record, index int) {
	ref := parseRecordRef(raw)
	if ref.Raw == "" {
		return nil, -1
	}
	for idx, candidate := range records {
		if ref.matches(candidate, auto) {
			return candidate, idx
		}
	}
	return nil, -1
}

func resolveRecordRefs(records []Record, raws []string, auto autoExternalIDs) map[string]struct{} {
	out := make(map[string]struct{}, len(raws))
	for _, raw := range raws {
		if record, idx := findRecordByRef(records, raw, auto); idx >= 0 {
			out[recordID(record)] = struct{}{}
		}
	}
	return out
}

func vehicleAutoExternalIDs(record Record) map[string]string {
	out := make(map[string]string, 2)
	if vin := stringValue(record, keyVIN); vin != "" {
		out[externalIDVinKey] = vin
	}
	if serial := assetGatewaySerial(record); serial != "" {
		out[externalIDSerialKey] = serial
	}
	return out
}

func assetGatewaySerial(record Record) string {
	for _, key := range []string{"gateway", "installedGateway"} {
		if serial := nestedString(record, key, keySerial); serial != "" {
			return serial
		}
	}
	return ""
}

func renderExternalIDs(record Record, auto autoExternalIDs) map[string]any {
	stored := externalIDsOf(record)
	var generated map[string]string
	if auto != nil {
		generated = auto(record)
	}
	out := make(map[string]any, len(stored)+len(generated))
	for key, value := range generated {
		out[key] = value
	}
	for key, value := range stored {
		out[key] = value
	}
	return out
}

func tagAutoExternalIDs(record Record) map[string]string {
	name := stringValue(record, keyName)
	if name == "" {
		return nil
	}
	return map[string]string{externalIDTagNameKey: name}
}

func validateExternalIDs(raw any) (map[string]any, error) {
	mapped, ok := anyAsMap(raw)
	if !ok {
		return nil, fmt.Errorf("%w: externalIds must be an object of string values", ErrInvalidBody)
	}
	out := make(map[string]any, len(mapped))
	for key, value := range mapped {
		if err := validateExternalIDKey(key); err != nil {
			return nil, err
		}
		text, err := externalIDValueString(key, value)
		if err != nil {
			return nil, err
		}
		if text == "" {
			continue
		}
		if err = validateExternalIDValue(key, text); err != nil {
			return nil, err
		}
		out[key] = text
	}
	return out, nil
}

func externalIDValueString(key string, value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "", nil
	case string:
		return strings.TrimSpace(typed), nil
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return "", fmt.Errorf("%w: externalIds.%s must be a string", ErrInvalidBody, key)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(typed), nil
	default:
		return "", fmt.Errorf("%w: externalIds.%s must be a string", ErrInvalidBody, key)
	}
}

func validateExternalIDValue(key, value string) error {
	for idx := 0; idx < len(value); idx++ {
		char := value[idx]
		if isASCIIAlphanumeric(char) || strings.IndexByte(externalIDValueAllowedChars, char) >= 0 {
			continue
		}
		return fmt.Errorf(
			"%w: externalIds.%s value %q may only contain letters, digits and @ . _ %% + -",
			ErrInvalidBody,
			key,
			value,
		)
	}
	return nil
}

func isASCIIAlphanumeric(char byte) bool {
	return (char >= '0' && char <= '9') || (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z')
}

func validateExternalIDKey(key string) error {
	if strings.HasPrefix(strings.ToLower(key), samsaraExternalIDPrefix) {
		return fmt.Errorf(
			"%w: externalIds key %q uses the reserved %q prefix",
			ErrInvalidBody,
			key,
			samsaraExternalIDPrefix,
		)
	}
	if key == "" || len(key) > maxExternalIDKeyLength {
		return fmt.Errorf(
			"%w: externalIds keys must be 1-%d characters",
			ErrInvalidBody,
			maxExternalIDKeyLength,
		)
	}
	for idx := 0; idx < len(key); idx++ {
		if !isASCIIAlphanumeric(key[idx]) {
			return fmt.Errorf(
				"%w: externalIds key %q may only contain letters and digits",
				ErrInvalidBody,
				key,
			)
		}
	}
	return nil
}

type externalIDHolder struct {
	Kind  string
	Owner string
	IDs   map[string]string
}

type externalIDClaim struct {
	Kind  string
	Owner string
	IDs   map[string]any
}

type externalIDSource func(records []Record, emit func(holder externalIDHolder))

const (
	externalIDKindDrivers    = "drivers"
	externalIDKindVehicles   = "vehicles"
	externalIDKindTrailers   = "trailers"
	externalIDKindAssets     = "assets"
	externalIDKindTags       = "tags"
	externalIDKindAddresses  = "addresses"
	externalIDKindRoutes     = "routes"
	externalIDKindRouteStops = "routeStops"
)

var externalIDSources = map[Resource]externalIDSource{
	ResourceDrivers:   flatExternalIDSource(ResourceDrivers, externalIDKindDrivers),
	ResourceTags:      flatExternalIDSource(ResourceTags, externalIDKindTags),
	ResourceAddresses: flatExternalIDSource(ResourceAddresses, externalIDKindAddresses),
	ResourceAssets:    assetExternalIDSource,
	ResourceRoutes:    routeExternalIDSource,
}

func externalIDOwner(resource Resource, id string) string {
	clean := strings.TrimSpace(id)
	if clean == "" {
		return ""
	}
	return string(resource) + ":" + clean
}

func nestedExternalIDOwner(parent, child, id string) string {
	if parent == "" || strings.TrimSpace(id) == "" {
		return ""
	}
	return parent + "/" + child + ":" + strings.TrimSpace(id)
}

func flatExternalIDSource(resource Resource, kind string) externalIDSource {
	return func(records []Record, emit func(holder externalIDHolder)) {
		for _, record := range records {
			if ids := externalIDsOf(record); len(ids) > 0 {
				emit(externalIDHolder{
					Kind:  kind,
					Owner: externalIDOwner(resource, recordID(record)),
					IDs:   ids,
				})
			}
		}
	}
}

func assetExternalIDKind(asset Record) string {
	switch assetType(asset) {
	case assetTypeVehicle:
		return externalIDKindVehicles
	case assetTypeTrailer:
		return externalIDKindTrailers
	default:
		return externalIDKindAssets
	}
}

func assetExternalIDSource(records []Record, emit func(holder externalIDHolder)) {
	for _, record := range records {
		if ids := externalIDsOf(record); len(ids) > 0 {
			emit(externalIDHolder{
				Kind:  assetExternalIDKind(record),
				Owner: externalIDOwner(ResourceAssets, recordID(record)),
				IDs:   ids,
			})
		}
	}
}

func routeExternalIDSource(records []Record, emit func(holder externalIDHolder)) {
	for _, record := range records {
		owner := externalIDOwner(ResourceRoutes, recordID(record))
		if ids := externalIDsOf(record); len(ids) > 0 {
			emit(externalIDHolder{Kind: externalIDKindRoutes, Owner: owner, IDs: ids})
		}
		for _, rawStop := range listOf(record["stops"]) {
			stop, ok := anyAsMap(rawStop)
			if !ok {
				continue
			}
			if ids := externalIDsOf(Record(stop)); len(ids) > 0 {
				emit(externalIDHolder{
					Kind:  externalIDKindRouteStops,
					Owner: nestedExternalIDOwner(owner, "stops", stringOf(stop[keyID])),
					IDs:   ids,
				})
			}
		}
	}
}

func ownedBy(owner, scope string) bool {
	if scope == "" || owner == "" {
		return false
	}
	return owner == scope || strings.HasPrefix(owner, scope+"/")
}

func (tx *storeTx) ensureExternalIDs(claims ...externalIDClaim) error {
	return tx.ensureExternalIDsExcept("", claims...)
}

func (tx *storeTx) ensureExternalIDsExcept(scope string, claims ...externalIDClaim) error {
	active := make([]externalIDClaim, 0, len(claims))
	for _, claim := range claims {
		if len(claim.IDs) > 0 {
			active = append(active, claim)
		}
	}
	if len(active) == 0 {
		return nil
	}
	if err := ensureClaimsDistinct(active); err != nil {
		return err
	}
	keysByKind := make(map[string]map[string]struct{}, len(active))
	for _, claim := range active {
		if keysByKind[claim.Kind] == nil {
			keysByKind[claim.Kind] = map[string]struct{}{}
		}
		for key := range claim.IDs {
			keysByKind[claim.Kind][key] = struct{}{}
		}
	}
	var conflict error
	excluded := func(owner string) bool {
		if ownedBy(owner, scope) {
			return true
		}
		for _, claim := range active {
			if claim.Owner != "" && claim.Owner == owner {
				return true
			}
		}
		return false
	}
	for resource, source := range externalIDSources {
		source(tx.records(resource), func(holder externalIDHolder) {
			if conflict != nil || excluded(holder.Owner) {
				return
			}
			if keys, ok := keysByKind[holder.Kind]; ok {
				for key := range holder.IDs {
					keys[key] = struct{}{}
				}
			}
			conflict = claimConflict(active, holder.IDs)
		})
		if conflict != nil {
			return conflict
		}
	}
	for kind, keys := range keysByKind {
		if len(keys) > maxExternalIDKeysPerType {
			return fmt.Errorf(
				"%w: at most %d unique external ID keys may be used across %s; this write would make %d",
				ErrInvalidBody,
				maxExternalIDKeysPerType,
				kind,
				len(keys),
			)
		}
	}
	return nil
}

func ensureClaimsDistinct(claims []externalIDClaim) error {
	for idx := range claims {
		for other := idx + 1; other < len(claims); other++ {
			if err := claimConflict(claims[idx:idx+1], externalIDStrings(claims[other].IDs)); err != nil {
				return err
			}
		}
	}
	return nil
}

func externalIDStrings(ids map[string]any) map[string]string {
	out := make(map[string]string, len(ids))
	for key, value := range ids {
		if text, ok := value.(string); ok {
			out[key] = text
		}
	}
	return out
}

func claimConflict(claims []externalIDClaim, existing map[string]string) error {
	if len(existing) == 0 {
		return nil
	}
	for _, claim := range claims {
		keys := make([]string, 0, len(claim.IDs))
		for key := range claim.IDs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if value, ok := existing[key]; ok && value == claim.IDs[key] {
				return fmt.Errorf(
					"%w: external ID %s:%v is already assigned to another object; external ID values must be unique across all objects",
					ErrUniqueConflict,
					key,
					claim.IDs[key],
				)
			}
		}
	}
	return nil
}
