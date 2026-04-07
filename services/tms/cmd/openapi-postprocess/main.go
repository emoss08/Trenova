package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"gopkg.in/yaml.v3"
)

type tagDescriptor struct {
	Name        string
	DisplayName string
	Group       string
}

var tagCatalog = map[string]tagDescriptor{
	"Auth":                     {Name: "Auth", DisplayName: "Authentication", Group: "Getting Started"},
	"System":                   {Name: "System", DisplayName: "System & Version", Group: "Getting Started"},
	"Realtime":                 {Name: "Realtime", DisplayName: "Realtime Access", Group: "Getting Started"},
	"Users":                    {Name: "Users", DisplayName: "Users", Group: "Identity & Access"},
	"Organizations":            {Name: "Organizations", DisplayName: "Organizations", Group: "Identity & Access"},
	"API Keys":                 {Name: "API Keys", DisplayName: "API Keys", Group: "Identity & Access"},
	"Permissions":              {Name: "Permissions", DisplayName: "Permissions", Group: "Identity & Access"},
	"Roles":                    {Name: "Roles", DisplayName: "Roles", Group: "Identity & Access"},
	"Role Assignments":         {Name: "Role Assignments", DisplayName: "Role Assignments", Group: "Identity & Access"},
	"Page Favorites":           {Name: "Page Favorites", DisplayName: "Page Favorites", Group: "Identity & Access"},
	"Shipments":                {Name: "Shipments", DisplayName: "Shipments", Group: "Operations"},
	"Shipment Moves":           {Name: "Shipment Moves", DisplayName: "Shipment Moves", Group: "Operations"},
	"Assignments":              {Name: "Assignments", DisplayName: "Assignments", Group: "Operations"},
	"Documents":                {Name: "Documents", DisplayName: "Documents", Group: "Operations"},
	"Workers":                  {Name: "Workers", DisplayName: "Workers", Group: "Operations"},
	"Worker PTO":               {Name: "Worker PTO", DisplayName: "Worker Time Off", Group: "Operations"},
	"Tractors":                 {Name: "Tractors", DisplayName: "Tractors", Group: "Operations"},
	"Trailers":                 {Name: "Trailers", DisplayName: "Trailers", Group: "Operations"},
	"Customers":                {Name: "Customers", DisplayName: "Customers", Group: "Master Data"},
	"Locations":                {Name: "Locations", DisplayName: "Locations", Group: "Master Data"},
	"Location Categories":      {Name: "Location Categories", DisplayName: "Location Categories", Group: "Master Data"},
	"Service Types":            {Name: "Service Types", DisplayName: "Service Types", Group: "Master Data"},
	"Shipment Types":           {Name: "Shipment Types", DisplayName: "Shipment Types", Group: "Master Data"},
	"Equipment Types":          {Name: "Equipment Types", DisplayName: "Equipment Types", Group: "Master Data"},
	"Equipment Manufacturers":  {Name: "Equipment Manufacturers", DisplayName: "Equipment Manufacturers", Group: "Master Data"},
	"Commodities":              {Name: "Commodities", DisplayName: "Commodities", Group: "Master Data"},
	"Accessorial Charges":      {Name: "Accessorial Charges", DisplayName: "Accessorial Charges", Group: "Master Data"},
	"Fleet Codes":              {Name: "Fleet Codes", DisplayName: "Fleet Codes", Group: "Master Data"},
	"Hold Reasons":             {Name: "Hold Reasons", DisplayName: "Hold Reasons", Group: "Master Data"},
	"Document Types":           {Name: "Document Types", DisplayName: "Document Types", Group: "Master Data"},
	"Account Types":            {Name: "Account Types", DisplayName: "Account Types", Group: "Master Data"},
	"GL Accounts":              {Name: "GL Accounts", DisplayName: "General Ledger Accounts", Group: "Master Data"},
	"Fiscal Years":             {Name: "Fiscal Years", DisplayName: "Fiscal Years", Group: "Master Data"},
	"Fiscal Periods":           {Name: "Fiscal Periods", DisplayName: "Fiscal Periods", Group: "Master Data"},
	"US States":                {Name: "US States", DisplayName: "U.S. States", Group: "Master Data"},
	"DOT Hazmat References":    {Name: "DOT Hazmat References", DisplayName: "DOT Hazmat References", Group: "Master Data"},
	"Hazardous Materials":      {Name: "Hazardous Materials", DisplayName: "Hazardous Materials", Group: "Master Data"},
	"Hazmat Segregation Rules": {Name: "Hazmat Segregation Rules", DisplayName: "Hazmat Segregation Rules", Group: "Master Data"},
	"Accounting Controls":      {Name: "Accounting Controls", DisplayName: "Accounting Controls", Group: "Controls & Configuration"},
	"Billing Queue":            {Name: "Billing Queue", DisplayName: "Billing Queue", Group: "Operations"},
	"Billing Controls":         {Name: "Billing Controls", DisplayName: "Billing Controls", Group: "Controls & Configuration"},
	"Dispatch Controls":        {Name: "Dispatch Controls", DisplayName: "Dispatch Controls", Group: "Controls & Configuration"},
	"Shipment Controls":        {Name: "Shipment Controls", DisplayName: "Shipment Controls", Group: "Controls & Configuration"},
	"Sequence Configs":         {Name: "Sequence Configs", DisplayName: "Sequence Configurations", Group: "Controls & Configuration"},
	"Table Configurations":     {Name: "Table Configurations", DisplayName: "Table Configurations", Group: "Controls & Configuration"},
	"Distance Overrides":       {Name: "Distance Overrides", DisplayName: "Distance Overrides", Group: "Controls & Configuration"},
	"Formula Templates":        {Name: "Formula Templates", DisplayName: "Formula Templates", Group: "Controls & Configuration"},
	"Integrations":             {Name: "Integrations", DisplayName: "Integrations", Group: "Controls & Configuration"},
	"Google Maps":              {Name: "Google Maps", DisplayName: "Google Maps", Group: "Controls & Configuration"},
	"Analytics":                {Name: "Analytics", DisplayName: "Analytics", Group: "Platform & Observability"},
	"Database Sessions":        {Name: "Database Sessions", DisplayName: "Database Sessions", Group: "Platform & Observability"},
	"Audit Entries":            {Name: "Audit Entries", DisplayName: "Audit Entries", Group: "Platform & Observability"},
	"Custom Fields":            {Name: "Custom Fields", DisplayName: "Custom Fields", Group: "Platform & Observability"},
}

var groupOrder = []string{
	"Getting Started",
	"Identity & Access",
	"Operations",
	"Master Data",
	"Controls & Configuration",
	"Platform & Observability",
	"Other",
}

func main() {
	root := filepath.Join(".", "docs")
	jsonPath := filepath.Join(root, "swagger.json")
	yamlPath := filepath.Join(root, "swagger.yaml")
	openAPI3JSONPath := filepath.Join(root, "openapi-3.json")
	openAPI3YAMLPath := filepath.Join(root, "openapi-3.yaml")

	doc, err := readJSON(jsonPath)
	if err != nil {
		fail(err)
	}

	applyScalarTagMetadata(doc)

	if err = writeJSON(jsonPath, doc); err != nil {
		fail(err)
	}

	if err = writeYAML(yamlPath, doc); err != nil {
		fail(err)
	}

	doc3, err := convertToOpenAPI3(doc)
	if err != nil {
		fail(err)
	}

	if err = writeJSON(openAPI3JSONPath, doc3); err != nil {
		fail(err)
	}

	if err = writeYAML(openAPI3YAMLPath, doc3); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "openapi postprocess failed: %v\n", err)
	os.Exit(1)
}

func readJSON(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc map[string]any
	if err = json.Unmarshal(content, &doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func writeJSON(path string, doc map[string]any) error {
	content, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		return err
	}

	content = append(content, '\n')
	return os.WriteFile(path, content, 0o644)
}

func writeYAML(path string, doc map[string]any) error {
	content, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0o644)
}

func convertToOpenAPI3(doc map[string]any) (map[string]any, error) {
	content, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	var doc2 openapi2.T
	if err = json.Unmarshal(content, &doc2); err != nil {
		return nil, err
	}

	doc3, err := openapi2conv.ToV3(&doc2)
	if err != nil {
		return nil, err
	}

	content, err = json.Marshal(doc3)
	if err != nil {
		return nil, err
	}

	var converted map[string]any
	if err = json.Unmarshal(content, &converted); err != nil {
		return nil, err
	}

	applyScalarTagMetadata(converted)
	return converted, nil
}

func applyScalarTagMetadata(doc map[string]any) {
	tagNames := collectTagNames(doc)
	descriptors := make([]tagDescriptor, 0, len(tagNames))
	for _, name := range tagNames {
		descriptors = append(descriptors, tagFor(name))
	}

	sort.SliceStable(descriptors, func(i, j int) bool {
		left := descriptors[i]
		right := descriptors[j]
		leftGroup := slices.Index(groupOrder, left.Group)
		rightGroup := slices.Index(groupOrder, right.Group)

		if leftGroup != rightGroup {
			return leftGroup < rightGroup
		}

		return left.DisplayName < right.DisplayName
	})

	tags := make([]map[string]any, 0, len(descriptors))
	groupedTags := make(map[string][]string)
	for _, tag := range descriptors {
		tags = append(tags, map[string]any{
			"name":          tag.Name,
			"x-displayName": tag.DisplayName,
		})
		groupedTags[tag.Group] = append(groupedTags[tag.Group], tag.Name)
	}

	tagGroups := make([]map[string]any, 0, len(groupOrder))
	for _, groupName := range groupOrder {
		names := groupedTags[groupName]
		if len(names) == 0 {
			continue
		}

		tagGroups = append(tagGroups, map[string]any{
			"name": groupName,
			"tags": names,
		})
	}

	doc["tags"] = tags
	doc["x-tagGroups"] = tagGroups
}

func collectTagNames(doc map[string]any) []string {
	seen := make(map[string]struct{})

	if rawTags, ok := doc["tags"].([]any); ok {
		for _, rawTag := range rawTags {
			tag, ok := rawTag.(map[string]any)
			if !ok {
				continue
			}

			name, ok := tag["name"].(string)
			if ok && name != "" {
				seen[name] = struct{}{}
			}
		}
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		return sortedTagNames(seen)
	}

	for _, rawPath := range paths {
		operations, ok := rawPath.(map[string]any)
		if !ok {
			continue
		}

		for _, rawOperation := range operations {
			operation, ok := rawOperation.(map[string]any)
			if !ok {
				continue
			}

			rawTags, ok := operation["tags"].([]any)
			if !ok {
				continue
			}

			for _, rawTag := range rawTags {
				name, ok := rawTag.(string)
				if ok && name != "" {
					seen[name] = struct{}{}
				}
			}
		}
	}

	return sortedTagNames(seen)
}

func sortedTagNames(seen map[string]struct{}) []string {
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func tagFor(name string) tagDescriptor {
	if tag, ok := tagCatalog[name]; ok {
		return tag
	}

	return tagDescriptor{
		Name:        name,
		DisplayName: name,
		Group:       "Other",
	}
}
