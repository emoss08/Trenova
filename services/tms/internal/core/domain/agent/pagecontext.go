package agent

import (
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	maxPagePathLength   = 500
	maxPageTitleLength  = 200
	maxPageEntityLength = 100
)

// PageContext is what the person was looking at when they asked. It is data
// about the page, never an instruction, and only the listed record kinds are
// accepted so a client cannot smuggle arbitrary text in as a "record".
type PageContext struct {
	Path       string `json:"path"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Title      string `json:"title"`
}

var knownPageEntityTypes = map[string]struct{}{
	"shipment":           {},
	"shipment_move":      {},
	"order":              {},
	"worker":             {},
	"tractor":            {},
	"trailer":            {},
	"customer":           {},
	"carrier":            {},
	"location":           {},
	"billing_queue_item": {},
	"invoice":            {},
	"document":           {},
	"rate_matrix":        {},
	"agent_definition":   {},
}

func KnownPageEntityTypes() []string {
	kinds := make([]string, 0, len(knownPageEntityTypes))
	for kind := range knownPageEntityTypes {
		kinds = append(kinds, kind)
	}

	return kinds
}

func IsKnownPageEntityType(kind string) bool {
	_, ok := knownPageEntityTypes[kind]
	return ok
}

// Validate reports field errors under the given prefix (for example
// "context"), so a client can show them next to the request that carried them.
func (p *PageContext) Validate(prefix string, multiErr *errortypes.MultiError) {
	field := func(name string) string {
		if prefix == "" {
			return name
		}

		return prefix + "." + name
	}

	path := strings.TrimSpace(p.Path)
	switch {
	case path == "":
		multiErr.Add(field("path"), errortypes.ErrRequired, "Path is required")
	case !strings.HasPrefix(path, "/"):
		multiErr.Add(field("path"), errortypes.ErrInvalid, "Path must be an application path")
	case len(path) > maxPagePathLength:
		multiErr.Add(field("path"), errortypes.ErrInvalid, "Path is too long")
	}

	if len(p.Title) > maxPageTitleLength {
		multiErr.Add(field("title"), errortypes.ErrInvalid, "Title is too long")
	}

	if p.EntityType != "" && !IsKnownPageEntityType(p.EntityType) {
		multiErr.Add(field("entityType"), errortypes.ErrInvalid, "Unknown record type")
	}

	if len(p.EntityID) > maxPageEntityLength {
		multiErr.Add(field("entityId"), errortypes.ErrInvalid, "Record identifier is too long")
	}

	if p.EntityID != "" && p.EntityType == "" {
		multiErr.Add(field("entityType"), errortypes.ErrRequired, "Record type is required with a record identifier")
	}
}

// Normalized trims what a person cannot see and drops a record reference the
// validator would reject, so the stored context is exactly what was used.
func (p *PageContext) Normalized() *PageContext {
	if p == nil {
		return nil
	}

	return &PageContext{
		Path:       strings.TrimSpace(p.Path),
		EntityType: strings.TrimSpace(p.EntityType),
		EntityID:   strings.TrimSpace(p.EntityID),
		Title:      strings.TrimSpace(p.Title),
	}
}
