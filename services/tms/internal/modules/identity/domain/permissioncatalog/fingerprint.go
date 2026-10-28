package permissioncatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

func (r *Registry) GrantFingerprint() string {
	defs := r.All()
	slices.SortFunc(defs, func(a, b *ResourceDefinition) int {
		return strings.Compare(a.Resource, b.Resource)
	})

	h := sha256.New()
	for _, def := range defs {
		ops := make([]string, 0, len(def.Operations))
		for _, op := range def.Operations {
			ops = append(ops, string(op.Operation))
		}
		slices.Sort(ops)

		h.Write([]byte(def.Resource))
		h.Write([]byte{0})
		h.Write([]byte(strings.Join(ops, ",")))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil))
}
