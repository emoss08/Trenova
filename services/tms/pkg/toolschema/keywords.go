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
// subset of one resource's records (see RecordSubset). Every "x-" keyword is
// stripped from what a model is shown (ForModel).
const (
	KeySubsetOf     = "x-subsetOf"
	extensionPrefix = "x-"
)
