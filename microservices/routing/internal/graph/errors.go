// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package graph

import "errors"

var (
	ErrNodeNotFound        = errors.New("node not found in graph")
	ErrNoPathFound         = errors.New("no path found between nodes")
	ErrSearchSpaceLimit    = errors.New("search space exceeded limit")
	ErrSearchLimitExceeded = errors.New("search limit exceeded")
	ErrTimeout             = errors.New("path finding timed out")
)
