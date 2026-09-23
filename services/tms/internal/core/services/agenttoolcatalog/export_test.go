package agenttoolcatalog

// Families exposes the family table to the contract test, which builds every
// registered tool and so lives outside the package.
func Families() [][]string {
	out := make([][]string, 0, len(families))
	for _, family := range families {
		out = append(out, append([]string(nil), family...))
	}

	return out
}

const MaxFamilySize = maxFamilySize
