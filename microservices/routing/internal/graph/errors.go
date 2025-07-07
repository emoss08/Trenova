package graph

import "errors"

var (
	ErrNodeNotFound        = errors.New("node not found in graph")
	ErrNoPathFound         = errors.New("no path found between nodes")
	ErrSearchSpaceLimit    = errors.New("search space exceeded limit")
	ErrSearchLimitExceeded = errors.New("search limit exceeded")
	ErrTimeout             = errors.New("path finding timed out")
)
