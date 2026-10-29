package formulaquerytools

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxLanguageLineDoc = 110

	termReefer       = "reefer"
	termRefrigerated = "refrigerated"
	termTemperature  = "temperature"
	termDistance     = "distance"
	termTables       = "tables"
	termWeight       = "weight"
	termClass        = "class"
)

var formulaVocabulary = map[string][]string{
	termReefer:       {termTemperature, termRefrigerated, termReefer},
	termRefrigerated: {termTemperature, termRefrigerated, termReefer},
	"temp":           {termTemperature},
	"cold":           {termTemperature, termReefer},
	"hazardous":      {"hazmat", "hazardous"},
	"miles":          {termDistance, "mi"},
	"mileage":        {termDistance, "mi"},
	"hundredweight":  {"cwt", termWeight},
	"pallets":        {"pieces", "pallet"},
	"fuel":           {"fuel", "diesel", "gal"},
	"ltl":            {"ltl", "break", "deficit", termClass},
	"matrix":         {termTables, "cell", "band"},
	"table":          {termTables, "cell", "band"},
}

type catalogDoc string

func (d *catalogDoc) UnmarshalJSON(data []byte) error {
	var text string
	if err := sonic.Unmarshal(data, &text); err == nil {
		*d = catalogDoc(text)
		return nil
	}
	var message struct {
		Message string `json:"message"`
	}
	if err := sonic.Unmarshal(data, &message); err != nil {
		return err
	}
	*d = catalogDoc(message.Message)
	return nil
}

type catalogField struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Doc    catalogDoc     `json:"doc"`
	Fields []catalogField `json:"fields"`
}

type catalogRecord struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Doc    catalogDoc     `json:"doc"`
	Fields []catalogField `json:"fields"`
}

type catalogDeclaration struct {
	Kind      string     `json:"kind"`
	Name      string     `json:"name"`
	Signature string     `json:"signature"`
	Doc       catalogDoc `json:"doc"`
}

type catalogModule struct {
	Path         string               `json:"path"`
	Doc          catalogDoc           `json:"doc"`
	Declarations []catalogDeclaration `json:"declarations"`
}

type catalogMethod struct {
	Name      string     `json:"name"`
	Receiver  string     `json:"receiver"`
	Signature string     `json:"signature"`
	Doc       catalogDoc `json:"doc"`
}

type catalogNamed struct {
	Name string     `json:"name"`
	Doc  catalogDoc `json:"doc"`
}

type catalogUnit struct {
	Dimension string `json:"dimension"`
	Unit      string `json:"unit"`
}

type tariffCatalog struct {
	Schema struct {
		Roots   []catalogRecord `json:"roots"`
		Records []catalogRecord `json:"records"`
	} `json:"schema"`
	Types        []catalogNamed  `json:"types"`
	Units        []catalogUnit   `json:"units"`
	Methods      []catalogMethod `json:"methods"`
	TableMethods []catalogNamed  `json:"tableMethods"`
	Modules      []catalogModule `json:"modules"`
}

type languageLine struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	Doc  string `json:"doc,omitempty"`
}

type languageModule struct {
	Path         string         `json:"path"`
	Doc          string         `json:"doc,omitempty"`
	Declarations []languageLine `json:"declarations"`
}

type languageTable struct {
	Code      string   `json:"code"`
	Name      string   `json:"name,omitempty"`
	Axes      []string `json:"axes"`
	ValueType string   `json:"valueType"`
}

type languageLibrary struct {
	Path          string `json:"path"`
	Description   string `json:"description,omitempty"`
	VersionNumber int64  `json:"versionNumber"`
	Source        string `json:"source,omitempty"`
}

type formulaLanguageView struct {
	Query        string              `json:"query,omitempty"`
	Record       []languageLine      `json:"record"`
	Types        []languageLine      `json:"types"`
	Units        map[string][]string `json:"units"`
	Methods      []languageLine      `json:"methods"`
	TableMethods []languageLine      `json:"tableMethods"`
	Modules      []languageModule    `json:"modules"`
	Tables       []languageTable     `json:"tables"`
	Codes        []string            `json:"accessorialCodes"`
	DistanceUnit string              `json:"distanceUnit,omitempty"`
	Libraries    []languageLibrary   `json:"libraries"`
	Note         string              `json:"note"`
}

type languageFilter struct {
	terms map[string]struct{}
}

func (f languageFilter) all() bool { return f.terms == nil }

func (f languageFilter) keeps(name, doc string) bool {
	return f.all() || matchesTerms(f.terms, name, doc, "")
}

func (f languageFilter) doc(doc catalogDoc) string {
	text := strings.TrimSpace(string(doc))
	if f.all() {
		return stringutils.Ellipsize(stringutils.FirstSentence(text), maxLanguageLineDoc)
	}
	return text
}

func languageView(
	language *formulaassistantservice.Language,
	query string,
) (*formulaLanguageView, error) {
	var catalog tariffCatalog
	if err := sonic.Unmarshal(language.Catalog, &catalog); err != nil {
		return nil, fmt.Errorf("read the tariff catalog: %w", err)
	}

	query = strings.TrimSpace(query)
	filter := languageFilter{terms: formulaTerms(query)}
	view := &formulaLanguageView{
		Query:        query,
		Record:       []languageLine{},
		Types:        []languageLine{},
		Units:        map[string][]string{},
		Methods:      []languageLine{},
		TableMethods: []languageLine{},
		Modules:      []languageModule{},
		Tables:       []languageTable{},
		Codes:        []string{},
		Libraries:    []languageLibrary{},
	}

	for i := range catalog.Schema.Roots {
		root := &catalog.Schema.Roots[i]
		if len(root.Fields) == 0 {
			if filter.keeps(root.Name, string(root.Doc)) {
				view.Record = append(view.Record, languageLine{
					Name: root.Name, Doc: filter.doc(root.Doc),
				})
			}
			continue
		}
		view.Record = appendFields(view.Record, filter, root.Name, root.Fields)
	}
	for i := range catalog.Schema.Records {
		record := &catalog.Schema.Records[i]
		view.Record = appendFields(view.Record, filter, record.Name, record.Fields)
	}
	for i := range catalog.Types {
		named := &catalog.Types[i]
		if filter.keeps(named.Name, string(named.Doc)) {
			view.Types = append(
				view.Types,
				languageLine{Name: named.Name, Doc: filter.doc(named.Doc)},
			)
		}
	}
	for _, unit := range catalog.Units {
		if filter.keeps(unit.Unit+" "+unit.Dimension, "") {
			view.Units[unit.Dimension] = append(view.Units[unit.Dimension], unit.Unit)
		}
	}
	for i := range catalog.Methods {
		method := &catalog.Methods[i]
		if filter.keeps(method.Name+" "+method.Receiver, string(method.Doc)) {
			view.Methods = append(view.Methods, languageLine{
				Name: method.Receiver + "." + method.Signature,
				Doc:  filter.doc(method.Doc),
			})
		}
	}
	for i := range catalog.TableMethods {
		method := &catalog.TableMethods[i]
		if filter.keeps(method.Name+" table", string(method.Doc)) {
			view.TableMethods = append(view.TableMethods, languageLine{
				Name: "tables.<code>." + method.Name,
				Doc:  filter.doc(method.Doc),
			})
		}
	}
	for i := range catalog.Modules {
		if module, ok := moduleView(&catalog.Modules[i], filter); ok {
			view.Modules = append(view.Modules, module)
		}
	}
	addContext(view, language, filter)
	view.Note = languageNote(view, query, filter)
	return view, nil
}

func appendFields(
	lines []languageLine,
	filter languageFilter,
	prefix string,
	fields []catalogField,
) []languageLine {
	for i := range fields {
		field := &fields[i]
		path := prefix + "." + field.Name
		if filter.keeps(path, string(field.Doc)) {
			lines = append(lines, languageLine{
				Name: path,
				Type: field.Type,
				Doc:  filter.doc(field.Doc),
			})
		}
		if len(field.Fields) > 0 {
			lines = appendFields(lines, filter, path, field.Fields)
		}
	}
	return lines
}

func moduleView(module *catalogModule, filter languageFilter) (languageModule, bool) {
	out := languageModule{Path: module.Path, Declarations: []languageLine{}}
	moduleMatches := filter.keeps(module.Path, string(module.Doc))
	for i := range module.Declarations {
		declaration := &module.Declarations[i]
		if !moduleMatches && !filter.keeps(declaration.Name, string(declaration.Doc)) {
			continue
		}
		name := declaration.Signature
		if name == "" {
			name = declaration.Kind + " " + declaration.Name
		}
		out.Declarations = append(out.Declarations, languageLine{
			Name: name,
			Doc:  filter.doc(declaration.Doc),
		})
	}
	if !moduleMatches && len(out.Declarations) == 0 {
		return out, false
	}
	out.Doc = filter.doc(module.Doc)
	return out, true
}

func contextTables(tariffCtx *rules.TariffContext, filter languageFilter) []languageTable {
	out := make([]languageTable, 0, len(tariffCtx.Tables))
	for i := range tariffCtx.Tables {
		table := &tariffCtx.Tables[i]
		if filter.keeps(table.Code+" "+table.Name+" table tables", "") {
			out = append(out, tableView(table))
		}
	}
	return out
}

func contextCodes(tariffCtx *rules.TariffContext, filter languageFilter) []string {
	if tariffCtx.Codes == nil {
		return nil
	}
	out := make([]string, 0, len(*tariffCtx.Codes))
	for _, code := range *tariffCtx.Codes {
		if !code.Active || !filter.keeps(code.Code+" accessorial code", code.Description) {
			continue
		}
		line := code.Code
		if code.Description != "" {
			line += ": " + code.Description
		}
		out = append(out, line)
	}
	return out
}

func addContext(
	view *formulaLanguageView,
	language *formulaassistantservice.Language,
	filter languageFilter,
) {
	tariffCtx := language.Context
	if tariffCtx != nil {
		view.DistanceUnit = tariffCtx.Settings.DistanceUnit
		view.Tables = append(view.Tables, contextTables(tariffCtx, filter)...)
		view.Codes = append(view.Codes, contextCodes(tariffCtx, filter)...)
	}

	sources := map[string]string{}
	if tariffCtx != nil {
		for _, library := range tariffCtx.Libraries {
			sources[library.Path] = library.Source
		}
	}
	for i := range language.Libraries {
		library := &language.Libraries[i]
		matched := filter.keeps(library.Path+" "+library.Name+" library", library.Description)
		if !matched {
			continue
		}
		line := languageLibrary{
			Path:          library.Path,
			Description:   library.Description,
			VersionNumber: library.VersionNumber,
		}
		if !filter.all() {
			line.Source = sources[library.Path]
		}
		view.Libraries = append(view.Libraries, line)
	}
}

func tableView(table *rules.TariffTable) languageTable {
	out := languageTable{
		Code:      table.Code,
		Name:      table.Name,
		ValueType: table.ValueType,
		Axes:      make([]string, 0, len(table.Axes)),
	}
	for _, axis := range table.Axes {
		text := axis.Match + " (" + axis.Kind
		if axis.Unit != "" {
			text += ", " + axis.Unit
		}
		text += ")"
		if axis.Label != "" {
			text = axis.Label + ": " + text
		}
		out.Axes = append(out.Axes, text)
	}
	return out
}

func languageNote(view *formulaLanguageView, query string, filter languageFilter) string {
	if filter.all() {
		return "Everything is listed, shortened, and libraries without their source. Call " +
			"again with query for the entries you need in full, or with a library's name for " +
			"its source. A name not listed is not in the language: a figure no record field " +
			"holds is an input you declare."
	}
	empty := len(view.Record) == 0 && len(view.Methods) == 0 && len(view.Modules) == 0 &&
		len(view.Tables) == 0 && len(view.Libraries) == 0 && len(view.Types) == 0
	if empty {
		return fmt.Sprintf(
			"Nothing in the language or this organisation matches %q. A figure no record "+
				"field holds is an input the formula declares (input name: Type = default); "+
				"never use a name that is not listed.", query,
		)
	}
	return fmt.Sprintf(
		"Only the entries matching %q are listed, in full. A name not listed here or in the "+
			"full list is not in the language.", query,
	)
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
