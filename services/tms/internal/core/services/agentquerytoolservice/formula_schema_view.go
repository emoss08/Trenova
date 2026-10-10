package agentquerytoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/services/agentsearch"
	"github.com/emoss08/trenova/internal/core/services/formulaassistantservice"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxSchemaLineDescription = 110

var formulaVocabulary = map[string][]string{
	"reefer":       {"temperature", "refrigerated", "reefer"},
	"refrigerated": {"temperature", "refrigerated"},
	"temp":         {"temperature"},
	"cold":         {"temperature"},
	"hazmat":       {"hazmat", "hazardous"},
	"hazardous":    {"hazmat", "hazardous"},
	"miles":        {"distance", "mile"},
	"mileage":      {"distance", "mile"},
	"cwt":          {"weight", "hundredweight"},
	"pallets":      {"pieces", "pallet"},
	"stops":        {"stop", "stops"},
	"fuel":         {"fuel", "diesel"},
}

type schemaVariableLine struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Computed    bool     `json:"computed,omitempty"`
	Values      []string `json:"values,omitempty"`
}

type schemaFunctionLine struct {
	Signature   string `json:"signature"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

type formulaSchemaView struct {
	SchemaID   string                              `json:"schemaId"`
	Query      string                              `json:"query,omitempty"`
	Variables  []schemaVariableLine                `json:"variables"`
	Functions  []schemaFunctionLine                `json:"functions"`
	RateTables []formulaassistantservice.RateTable `json:"rateTables"`
	Note       string                              `json:"note,omitempty"`
}

func schemaView(reference *formulaassistantservice.Reference, query string) *formulaSchemaView {
	query = strings.TrimSpace(query)
	view := &formulaSchemaView{
		SchemaID:   reference.SchemaID,
		Query:      query,
		Variables:  make([]schemaVariableLine, 0, len(reference.Variables)),
		Functions:  make([]schemaFunctionLine, 0, len(reference.Functions)),
		RateTables: reference.RateTables,
	}
	terms := formulaTerms(query)

	for idx := range reference.Variables {
		variable := &reference.Variables[idx]
		if terms != nil && !matchesTerms(terms, variable.Name, variable.Description, variable.Category) {
			continue
		}
		view.Variables = append(view.Variables, variableLine(variable, terms != nil))
	}
	for idx := range reference.Functions {
		function := &reference.Functions[idx]
		if terms != nil && !matchesTerms(terms, function.Name, function.Description, function.Category) {
			continue
		}
		line := schemaFunctionLine{Signature: function.Signature}
		if line.Signature == "" {
			line.Signature = function.Name
		}
		if terms != nil {
			line.Description = function.Description
			line.Example = function.Example
		}
		view.Functions = append(view.Functions, line)
	}

	switch {
	case terms == nil:
		view.Note = "Every variable and function is listed, shortened. Call again with query " +
			"for the ones you need described in full, with examples."
	case len(view.Variables) == 0 && len(view.Functions) == 0:
		view.Note = fmt.Sprintf(
			"Nothing in the schema matches %q, so no shipment field holds it. Make it a "+
				"template variable with a default, or ask the person what marks it; never "+
				"use a name that is not listed.", query,
		)
	default:
		view.Note = fmt.Sprintf(
			"%d of %d variables and %d of %d functions match %q. A name not listed here or "+
				"in the full list is not a shipment field: a quantity no listed field holds, "+
				"such as hours at a stop, is a template variable with a default that you "+
				"declare in propose_formula, not something to keep searching for.",
			len(view.Variables), len(reference.Variables),
			len(view.Functions), len(reference.Functions), query,
		)
	}

	return view
}

func variableLine(variable *formulatemplatetypes.SchemaVariableInfo, full bool) schemaVariableLine {
	description := variable.Description
	if !full {
		description = stringutils.Ellipsize(stringutils.FirstSentence(description), maxSchemaLineDescription)
	}

	return schemaVariableLine{
		Name:        variable.Name,
		Type:        variable.Type,
		Description: description,
		Computed:    variable.Computed,
		Values:      variable.Enum,
	}
}

func formulaTerms(query string) map[string]struct{} {
	if query == "" {
		return nil
	}
	terms := agentsearch.Terms(agentsearch.SplitCamel(query) + " " + query)
	for term := range agentsearch.Terms(agentsearch.SplitCamel(query)) {
		for _, synonym := range formulaVocabulary[term] {
			terms[synonym] = struct{}{}
		}
	}
	if len(terms) == 0 {
		return nil
	}

	return terms
}

func matchesTerms(terms map[string]struct{}, name, description, category string) bool {
	tokens := agentsearch.TokenSet(agentsearch.SplitCamel(name) + " " + name + " " +
		description + " " + category)
	for term := range terms {
		if _, ok := tokens[term]; ok {
			return true
		}
		for token := range tokens {
			if len(term) >= 4 && strings.HasPrefix(token, term) {
				return true
			}
		}
	}

	return false
}
