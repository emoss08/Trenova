package helpers

import "net/http"

type ProblemType string

const (
	ProblemTypeBlank          = ProblemType("about:blank")
	ProblemTypeValidation     = ProblemType("validation-error")
	ProblemTypeBusiness       = ProblemType("business-rule-violation")
	ProblemTypeDatabase       = ProblemType("database-error")
	ProblemTypeAuthentication = ProblemType("authentication-error")
	ProblemTypeAuthorization  = ProblemType("authorization-error")
	ProblemTypeNotFound       = ProblemType("resource-not-found")
	ProblemTypeRateLimit      = ProblemType("rate-limit-exceeded")
	ProblemTypeConflict       = ProblemType("resource-conflict")
	ProblemTypeInternal       = ProblemType("internal-error")
)

type ProblemTypeInfo struct {
	Type       ProblemType
	Title      string
	StatusCode int
	ShouldLog  bool
}

var problemTypeRegistry = map[ProblemType]ProblemTypeInfo{ //nolint:exhaustive // we don't need to convert blank
	ProblemTypeValidation: {
		Type:       ProblemTypeValidation,
		Title:      "Validation Failed",
		StatusCode: http.StatusBadRequest,
		ShouldLog:  false,
	},
	ProblemTypeBusiness: {
		Type:       ProblemTypeBusiness,
		Title:      "Business Rule Violation",
		StatusCode: http.StatusUnprocessableEntity,
		ShouldLog:  true,
	},
	ProblemTypeDatabase: {
		Type:       ProblemTypeDatabase,
		Title:      "Database Operation Failed",
		StatusCode: http.StatusInternalServerError,
		ShouldLog:  true,
	},
	ProblemTypeAuthentication: {
		Type:       ProblemTypeAuthentication,
		Title:      "Authentication Required",
		StatusCode: http.StatusUnauthorized,
		ShouldLog:  true,
	},
	ProblemTypeAuthorization: {
		Type:       ProblemTypeAuthorization,
		Title:      "Authorization Required",
		StatusCode: http.StatusForbidden,
		ShouldLog:  true,
	},
	ProblemTypeNotFound: {
		Type:       ProblemTypeNotFound,
		Title:      "Resource Not Found",
		StatusCode: http.StatusNotFound,
		ShouldLog:  false,
	},
	ProblemTypeRateLimit: {
		Type:       ProblemTypeRateLimit,
		Title:      "Rate Limit Exceeded",
		StatusCode: http.StatusTooManyRequests,
		ShouldLog:  false,
	},
	ProblemTypeConflict: {
		Type:       ProblemTypeConflict,
		Title:      "Resource Conflict",
		StatusCode: http.StatusConflict,
		ShouldLog:  true,
	},
	ProblemTypeInternal: {
		Type:       ProblemTypeInternal,
		Title:      "Internal Server Error",
		StatusCode: http.StatusInternalServerError,
		ShouldLog:  true,
	},
}

func (t ProblemType) Info() ProblemTypeInfo {
	if info, ok := problemTypeRegistry[t]; ok {
		return info
	}
	return problemTypeRegistry[ProblemTypeInternal]
}

func (t ProblemType) IsInternal() bool {
	return t == ProblemTypeInternal || t == ProblemTypeDatabase
}
