package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
)

// ProductGuideSearchRequest asks where something is, or how it is done, on
// behalf of one person.
type ProductGuideSearchRequest struct {
	Actor      *RequestActor
	TenantInfo pagination.TenantInfo
	// Query is the person's question in their own words.
	Query string
	// Page narrows the answer to one page's tasks, by its path.
	Page        string
	Limit       int
	Attribution AIUsageAttribution
}

// ProductGuideMatch is one page that answers a question.
type ProductGuideMatch struct {
	Page *productguide.Page
	// Task is the page's task that best answers the question, when one does.
	Task *productguide.Task
	// CanOpen is whether this person may open the page. A page they cannot
	// open is still an answer — "that needs billing access" is worth saying —
	// so it is returned with what is missing rather than hidden.
	CanOpen bool
	Missing []string
}

// ProductGuideDestinationRequest names a place to take somebody: a page by its
// path, or one record by its kind and id.
type ProductGuideDestinationRequest struct {
	Actor      *RequestActor
	TenantInfo pagination.TenantInfo
	Page       string
	Entity     string
	RecordID   string
	// Create opens the page's create form, where it has one.
	Create bool
}

// ProductGuideDestination is a place the person may be taken, checked.
type ProductGuideDestination struct {
	Path  string
	Page  *productguide.Page
	Label string
}

// ProductGuide answers where things are in Trenova and how to do them, for
// the person asking.
type ProductGuide interface {
	Search(ctx context.Context, req *ProductGuideSearchRequest) ([]ProductGuideMatch, error)
	// Destination resolves and checks a place to take somebody. It refuses a
	// page that does not exist, a record kind with no page, and a page the
	// person may not open, each with a sentence saying so.
	Destination(
		ctx context.Context,
		req *ProductGuideDestinationRequest,
	) (*ProductGuideDestination, error)
	// PageForPath is the page a location belongs to, with no access check:
	// it names where somebody already is.
	PageForPath(path string) (*productguide.Page, bool)
}
