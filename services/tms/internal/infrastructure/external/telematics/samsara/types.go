/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package samsara

// Samsara API Pagination
type Pagination struct {
	EndCursor   string `json:"endCursor"`
	HasNextPage bool   `json:"hasNextPage"`
}

// DataResponse is the response from the Samsara API.
type DataResponse[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// TaggedObject is a generic object that can be tagged with a tag.
type TaggedObject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ParentTag is a tag that can be used to group tags.
type ParentTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Tag is a tag that can be used to group tagged objects.
type Tag struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	ParentTagID string         `json:"parentTagId"`
	ParentTag   *ParentTag     `json:"parentTag"`
	Vehicles    []TaggedObject `json:"vehicles"    optional:"true"`
	Assets      []TaggedObject `json:"assets"      optional:"true"`
	Sensors     []TaggedObject `json:"sensors"     optional:"true"`
	Addresses   []TaggedObject `json:"addresses"   optional:"true"`
}
