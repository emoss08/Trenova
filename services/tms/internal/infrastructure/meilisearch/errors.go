package meilisearch

import "errors"

var (
	ErrInvalidEntityType = errors.New("invalid entity type")
	ErrDisabled          = errors.New("search functionality is disabled")
	ErrNoDocuments       = errors.New("no documents to index")
)
