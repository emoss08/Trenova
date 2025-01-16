package search

import (
	"github.com/trenova-app/transport/internal/core/ports/infra"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type batchOpt struct {
	documents []*infra.SearchDocument
	callback  func(error)
}

type Request struct {
	Query       string   `json:"query"`
	Types       []string `json:"types"`
	Limit       int      `json:"limit"`
	Offset      int      `json:"offset"`
	RequesterID pulid.ID `json:"requester_id"`
	OrgID       pulid.ID `json:"org_id"`
	BuID        pulid.ID `json:"bu_id"`
}

type Response struct {
	Results []*infra.SearchDocument `json:"results"`
	Total   int                     `json:"total"`
	Query   string                  `json:"query"`
}
