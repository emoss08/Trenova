package permissioncatalog

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
)

// referenceBookkeeping is part of every kind's reference view: a record's id,
// its tenant and its revision say nothing about its content.
var referenceBookkeeping = []string{
	"id",
	"organizationId",
	"businessUnitId",
	"version",
	"createdAt",
	"updatedAt",
}

func inReferenceView(def *permissionspec.ResourceDefinition, field string) bool {
	return slices.Contains(referenceBookkeeping, field) || slices.Contains(def.ReferenceView, field)
}

// InReferenceView reports whether a field of a kind is in its reference view:
// what any record that links to it may show, without the kind's own read.
func (r *Registry) InReferenceView(resource, field string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.resources[resource]
	if !ok {
		return slices.Contains(referenceBookkeeping, field)
	}
	return inReferenceView(def, field)
}

// ReferenceView is every field of a kind's reference view, sorted.
func (r *Registry) ReferenceView(resource string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := slices.Clone(referenceBookkeeping)
	if def, ok := r.resources[resource]; ok {
		for _, field := range def.ReferenceView {
			if !slices.Contains(out, field) {
				out = append(out, field)
			}
		}
	}
	slices.Sort(out)

	return out
}

// Family is the resource a kind belongs to through its parents: a shipment's
// stops and moves are the shipment's, a worker's credentials the worker's.
func (r *Registry) Family(resource string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for range len(r.resources) {
		def, ok := r.resources[resource]
		if !ok || def.ParentResource == "" {
			return resource
		}
		resource = def.ParentResource
	}

	return resource
}

// PartOf reports whether a kind carried by another is part of that record
// rather than a record it links to: a child of the same family, such as a
// shipment's stop under its move. A family's root is never part of anything,
// so a worker under one of their own credentials is still a linked record.
func (r *Registry) PartOf(kind, carrier string) bool {
	if kind == carrier {
		return true
	}
	family := r.Family(kind)

	return family != kind && family == r.Family(carrier)
}
