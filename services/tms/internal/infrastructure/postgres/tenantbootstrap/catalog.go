package tenantbootstrap

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
)

func DocumentTypeCodes() []string {
	docTypes := systemDocumentTypes(Scope{})
	codes := make([]string, 0, len(docTypes))
	for _, docType := range docTypes {
		codes = append(codes, docType.Code)
	}

	return codes
}

func ServiceFailureReasonCodes() []string {
	codes := make([]string, 0, len(defaultServiceFailureReasonCodes))
	for _, def := range defaultServiceFailureReasonCodes {
		codes = append(codes, def.code)
	}

	return codes
}

func SystemAgentKeys() []string {
	definitions := SystemAgentDefinitions("", "")
	keys := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		keys = append(keys, definition.SystemKey)
	}

	return keys
}

func TemplateStarterKinds() []string {
	kinds := documenttemplate.AllKinds()
	out := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, string(kind))
	}

	return out
}
