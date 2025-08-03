/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package accessorialcharge

type Method string

const (
	MethodFlat       = Method("Flat")
	MethodDistance   = Method("Distance")
	MethodPercentage = Method("Percentage")
)
