package platformaudit

import (
	"sort"

	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
)

type PackReach struct {
	Pack           platformcatalog.Pack
	Features       []platformcatalog.FeatureKey
	RouteCount     int
	GraphQLSources int
}

type Findings struct {
	UnsellableFeatures   []platformcatalog.FeatureKey
	UnreachableRoutes    map[platformcatalog.FeatureKey][]string
	UnreachableGraphQL   map[platformcatalog.FeatureKey][]string
	LegacyOnlyFeatures   []platformcatalog.FeatureKey
	MigrationLostRoutes  map[platformcatalog.FeatureKey][]string
	MigrationLostGraphQL map[platformcatalog.FeatureKey][]string
	PackReach            []PackReach
}

func graphQLSurface(feature *platformcatalog.Feature) []string {
	surface := make([]string, 0, len(feature.GraphQLSources)+len(feature.GraphQLRootFields))
	for _, source := range feature.GraphQLSources {
		surface = append(surface, string(source))
	}
	for _, field := range feature.GraphQLRootFields {
		surface = append(surface, field.Key())
	}

	return surface
}

type Input struct {
	Registry      *platformcatalog.Registry
	Routes        []routelint.Route
	LegacyEntitle []platformcatalog.FeatureKey
}

func Audit(in Input) Findings {
	sellable := sellableFeatures(in.Registry)
	legacy := keySet(in.LegacyEntitle)

	findings := Findings{
		UnreachableRoutes:    map[platformcatalog.FeatureKey][]string{},
		UnreachableGraphQL:   map[platformcatalog.FeatureKey][]string{},
		MigrationLostRoutes:  map[platformcatalog.FeatureKey][]string{},
		MigrationLostGraphQL: map[platformcatalog.FeatureKey][]string{},
	}

	features := in.Registry.ListFeatures()
	for i := range features {
		feature := &features[i]
		surface := graphQLSurface(feature)

		if _, ok := sellable[feature.Key]; !ok {
			findings.UnsellableFeatures = append(findings.UnsellableFeatures, feature.Key)
			if len(surface) > 0 {
				findings.UnreachableGraphQL[feature.Key] = surface
			}
		}
		if len(feature.LegacyGrantingFeatures) > 0 {
			findings.LegacyOnlyFeatures = append(findings.LegacyOnlyFeatures, feature.Key)
		}
		if !grantedBy(in.Registry, feature.Key, legacy) && len(surface) > 0 {
			findings.MigrationLostGraphQL[feature.Key] = surface
		}
	}

	for _, route := range in.Routes {
		policy := in.Registry.PolicyForRoute(route.Method, route.Path)
		if policy.AccessClass != platformcatalog.RouteAccessClassProduct {
			continue
		}
		if _, ok := sellable[policy.FeatureKey]; !ok {
			findings.UnreachableRoutes[policy.FeatureKey] = append(
				findings.UnreachableRoutes[policy.FeatureKey],
				route.Key(),
			)
		}
		if !grantedBy(in.Registry, policy.FeatureKey, legacy) {
			findings.MigrationLostRoutes[policy.FeatureKey] = append(
				findings.MigrationLostRoutes[policy.FeatureKey],
				route.Key(),
			)
		}
	}

	findings.PackReach = packReach(in.Registry, in.Routes)
	sortFeatureKeys(findings.UnsellableFeatures)
	sortFeatureKeys(findings.LegacyOnlyFeatures)

	return findings
}

func sellableFeatures(
	registry *platformcatalog.Registry,
) map[platformcatalog.FeatureKey]struct{} {
	sellable := make(map[platformcatalog.FeatureKey]struct{})
	for _, pack := range registry.ListPacks() {
		granted, ok := registry.PackGrantedFeatures(pack.Key)
		if !ok {
			continue
		}
		for _, featureKey := range granted {
			sellable[featureKey] = struct{}{}
		}
	}

	return sellable
}

func packReach(
	registry *platformcatalog.Registry,
	routes []routelint.Route,
) []PackReach {
	routeFeatures := make([]platformcatalog.FeatureKey, 0, len(routes))
	for _, route := range routes {
		policy := registry.PolicyForRoute(route.Method, route.Path)
		if policy.AccessClass != platformcatalog.RouteAccessClassProduct {
			continue
		}
		routeFeatures = append(routeFeatures, policy.FeatureKey)
	}

	packs := registry.ListPacks()
	reach := make([]PackReach, 0, len(packs))
	for _, pack := range packs {
		grantedKeys, _ := registry.PackGrantedFeatures(pack.Key)
		granted := keySet(grantedKeys)

		entry := PackReach{Pack: pack, Features: grantedKeys}
		for _, featureKey := range routeFeatures {
			if _, ok := granted[featureKey]; ok {
				entry.RouteCount++
			}
		}
		for _, featureKey := range grantedKeys {
			feature, ok := registry.GetFeature(featureKey)
			if !ok {
				continue
			}
			entry.GraphQLSources += len(graphQLSurface(&feature))
		}
		reach = append(reach, entry)
	}

	return reach
}

func grantedBy(
	registry *platformcatalog.Registry,
	featureKey platformcatalog.FeatureKey,
	held map[platformcatalog.FeatureKey]struct{},
) bool {
	for _, candidate := range registry.AuthorizingFeatures(featureKey, true) {
		if _, ok := held[candidate]; ok {
			return true
		}
	}

	return false
}

func keySet(keys []platformcatalog.FeatureKey) map[platformcatalog.FeatureKey]struct{} {
	set := make(map[platformcatalog.FeatureKey]struct{}, len(keys))
	for _, key := range keys {
		set[key] = struct{}{}
	}

	return set
}

func sortFeatureKeys(keys []platformcatalog.FeatureKey) {
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
}
