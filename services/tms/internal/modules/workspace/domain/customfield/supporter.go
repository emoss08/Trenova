package customfield

type CustomFieldsSupporter interface {
	GetResourceType() string
	GetResourceID() string
}
