package toolschema

const (
	KeyType                 = "type"
	KeyProperties           = "properties"
	KeyRequired             = "required"
	KeyAdditionalProperties = "additionalProperties"
	KeyDescription          = "description"
	KeyMinimum              = "minimum"
	KeyMaximum              = "maximum"
	KeyItems                = "items"
	KeyEnum                 = "enum"
	KeyMaxLength            = "maxLength"
	KeyMinItems             = "minItems"
	KeyMaxItems             = "maxItems"

	TypeObject  = "object"
	TypeString  = "string"
	TypeInteger = "integer"
	TypeBoolean = "boolean"
	TypeArray   = "array"
)

// KeySubsetOf is this package's extension keyword marking an array of ids as a
// subset of one resource's records (see RecordSubset). KeyEnumOf names the
// registered source an enum's values were taken from, so a contract test can
// hold the list to the domain that owns it. KeyRecordOf names the resource
// whose record id a string parameter takes, so the runtime can tell an id of
// the wrong kind from the right one by its prefix. KeyRecordKinds names the
// kinds of record a string parameter's id may be, for an id that is one of
// several kinds or a kind no resource stands for alone. Every "x-" keyword is
// stripped from what a model is shown (ForModel).
const (
	KeySubsetOf     = "x-subsetOf"
	KeyEnumOf       = "x-enumOf"
	KeyRecordOf     = "x-recordOf"
	KeyRecordKinds  = "x-recordKinds"
	KeyKeepEmpty    = "x-keepEmpty"
	extensionPrefix = "x-"
)

// RecordOf reads the resource a property declares its id belongs to.
func RecordOf(property map[string]any) string {
	resource, _ := property[KeyRecordOf].(string)

	return resource
}

// RecordKinds reads the kinds of record a property declares its id may be. A
// schema built in Go holds them as strings; one read back from JSON holds
// them as values of any type, and both are read.
func RecordKinds(property map[string]any) []string {
	switch kinds := property[KeyRecordKinds].(type) {
	case []string:
		return kinds
	case []any:
		out := make([]string, 0, len(kinds))
		for _, kind := range kinds {
			if name, ok := kind.(string); ok && name != "" {
				out = append(out, name)
			}
		}

		return out
	default:
		return nil
	}
}
