package utils

type ContextKey string

const (
	CTXKeyDisableLogger = ContextKey("disableLogger")
	CTXOrganizationID   = ContextKey("organizationID")
	CTXBusinessUnitID   = ContextKey("businessUnitID")
	CTXUserID           = ContextKey("userID")
	CTXDB               = ContextKey("db")
)
