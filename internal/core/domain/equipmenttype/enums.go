// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package equipmenttype

type Class string

const (
	ClassTractor   = Class("Tractor")
	ClassTrailer   = Class("Trailer")
	ClassContainer = Class("Container")
	ClassOther     = Class("Other")
)
