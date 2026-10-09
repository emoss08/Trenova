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
// the wrong kind from the right one by its prefix. Every "x-" keyword is
// stripped from what a model is shown (ForModel).
const (
	KeySubsetOf     = "x-subsetOf"
	KeyEnumOf       = "x-enumOf"
	KeyRecordOf     = "x-recordOf"
	extensionPrefix = "x-"
)

// RecordOf reads the resource a property declares its id belongs to.
func RecordOf(property map[string]any) string {
	resource, _ := property[KeyRecordOf].(string)

	return resource
}
