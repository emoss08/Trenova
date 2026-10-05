package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"go.uber.org/fx"
)

type RegistryParams struct {
	fx.In

	Providers []platformcatalog.CatalogProvider `group:"platform_catalog_providers"`
}

type Registry struct {
	products          map[platformcatalog.ProductKey]platformcatalog.Product
	features          map[platformcatalog.FeatureKey]platformcatalog.Feature
	meters            map[platformcatalog.MeterKey]platformcatalog.Meter
	packs             map[platformcatalog.PackKey]platformcatalog.Pack
	graphQLSources    map[platformcatalog.GraphQLSource]platformcatalog.FeatureKey
	graphQLRootFields map[string]platformcatalog.FeatureKey
}

func NewRegistry(p RegistryParams) (*Registry, error) {
	registry := &Registry{
		products:          make(map[platformcatalog.ProductKey]platformcatalog.Product),
		features:          make(map[platformcatalog.FeatureKey]platformcatalog.Feature),
		meters:            make(map[platformcatalog.MeterKey]platformcatalog.Meter),
		packs:             make(map[platformcatalog.PackKey]platformcatalog.Pack),
		graphQLSources:    make(map[platformcatalog.GraphQLSource]platformcatalog.FeatureKey),
		graphQLRootFields: make(map[string]platformcatalog.FeatureKey),
	}

	for _, provider := range p.Providers {
		if err := registry.registerProvider(provider); err != nil {
			return nil, err
		}
	}

	if err := registry.Validate(); err != nil {
		return nil, err
	}

	return registry, nil
}

func (r *Registry) ListProducts() []platformcatalog.Product {
	products := make([]platformcatalog.Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}
	slices.SortFunc(products, func(a, b platformcatalog.Product) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})
	return products
}

func (r *Registry) ListFeatures() []platformcatalog.Feature {
	features := make([]platformcatalog.Feature, 0, len(r.features))
	for key := range r.features {
		features = append(features, r.features[key])
	}
	slices.SortFunc(features, func(a, b platformcatalog.Feature) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})
	return features
}

func (r *Registry) ListMeters() []platformcatalog.Meter {
	meters := make([]platformcatalog.Meter, 0, len(r.meters))
	for _, meter := range r.meters {
		meters = append(meters, meter)
	}
	slices.SortFunc(meters, func(a, b platformcatalog.Meter) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})
	return meters
}

func (r *Registry) GetProduct(key platformcatalog.ProductKey) (platformcatalog.Product, bool) {
	product, ok := r.products[key]
	return product, ok
}

func (r *Registry) GetFeature(key platformcatalog.FeatureKey) (platformcatalog.Feature, bool) {
	feature, ok := r.features[key]
	return feature, ok
}

func (r *Registry) FeatureForRoute(method, routePattern string) (platformcatalog.Feature, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	routePattern = normalizeRoutePath(routePattern)
	if routePattern == "" {
		return platformcatalog.Feature{}, false
	}

	if routeMatchesAny(method, routePattern, accountShellRouteRefs()) {
		return platformcatalog.Feature{}, false
	}

	return r.featureForRoute(method, routePattern)
}

func (r *Registry) PolicyForRoute(method, routePattern string) platformcatalog.RoutePolicy {
	method = strings.ToUpper(strings.TrimSpace(method))
	routePattern = normalizeRoutePath(routePattern)
	if routePattern == "" {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassUnclassified,
		}
	}

	if routeMatchesAny(method, routePattern, accountShellRouteRefs()) {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassAccountShell,
		}
	}

	feature, ok := r.featureForRoute(method, routePattern)
	if !ok {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassUnclassified,
		}
	}

	return platformcatalog.RoutePolicy{
		AccessClass: platformcatalog.RouteAccessClassProduct,
		FeatureKey:  feature.Key,
	}
}

func (r *Registry) featureForRoute(method, routePattern string) (platformcatalog.Feature, bool) {
	for _, feature := range r.ListFeatures() {
		for _, route := range feature.Routes {
			if routeMatches(method, routePattern, route) {
				return feature, true
			}
		}
	}

	return platformcatalog.Feature{}, false
}

func (r *Registry) GetMeter(key platformcatalog.MeterKey) (platformcatalog.Meter, bool) {
	meter, ok := r.meters[key]
	return meter, ok
}

func (r *Registry) ListPacks() []platformcatalog.Pack {
	packs := make([]platformcatalog.Pack, 0, len(r.packs))
	for key := range r.packs {
		packs = append(packs, r.packs[key])
	}
	slices.SortFunc(packs, func(a, b platformcatalog.Pack) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})

	return packs
}

func (r *Registry) GetPack(key platformcatalog.PackKey) (platformcatalog.Pack, bool) {
	pack, ok := r.packs[key]
	return pack, ok
}

func (r *Registry) PackFeatureClosure(
	key platformcatalog.PackKey,
) ([]platformcatalog.FeatureKey, bool) {
	pack, ok := r.packs[key]
	if !ok {
		return nil, false
	}

	resolved := make(map[platformcatalog.FeatureKey]struct{}, len(pack.Features))
	for _, featureKey := range pack.Features {
		r.collectRequiredFeatures(featureKey, resolved)
	}

	return sortedFeatureKeys(resolved), true
}

func (r *Registry) PackGrantedFeatures(
	key platformcatalog.PackKey,
) ([]platformcatalog.FeatureKey, bool) {
	pack, ok := r.packs[key]
	if !ok {
		return nil, false
	}

	granted := make(map[platformcatalog.FeatureKey]struct{}, len(pack.Features))
	visited := make(map[platformcatalog.PackKey]struct{})
	r.collectPackFeatures(key, &pack, granted, visited)

	return sortedFeatureKeys(granted), true
}

func (r *Registry) collectPackFeatures(
	key platformcatalog.PackKey,
	pack *platformcatalog.Pack,
	granted map[platformcatalog.FeatureKey]struct{},
	visited map[platformcatalog.PackKey]struct{},
) {
	if _, seen := visited[key]; seen {
		return
	}
	visited[key] = struct{}{}

	for _, featureKey := range pack.Features {
		granted[featureKey] = struct{}{}
	}
	for _, requiredKey := range pack.RequiresPacks {
		requiredPack, ok := r.packs[requiredKey]
		if !ok {
			continue
		}
		r.collectPackFeatures(requiredKey, &requiredPack, granted, visited)
	}
}

func sortedFeatureKeys(keys map[platformcatalog.FeatureKey]struct{}) []platformcatalog.FeatureKey {
	sorted := make([]platformcatalog.FeatureKey, 0, len(keys))
	for featureKey := range keys {
		sorted = append(sorted, featureKey)
	}
	slices.SortFunc(sorted, func(a, b platformcatalog.FeatureKey) int {
		return strings.Compare(string(a), string(b))
	})

	return sorted
}

func (r *Registry) collectRequiredFeatures(
	key platformcatalog.FeatureKey,
	resolved map[platformcatalog.FeatureKey]struct{},
) {
	if _, seen := resolved[key]; seen {
		return
	}
	feature, ok := r.features[key]
	if !ok {
		return
	}
	resolved[key] = struct{}{}

	for _, required := range feature.RequiresFeatures {
		r.collectRequiredFeatures(required, resolved)
	}
}

func (r *Registry) AuthorizingFeatures(
	key platformcatalog.FeatureKey,
	includeLegacy bool,
) []platformcatalog.FeatureKey {
	feature, ok := r.features[key]
	if !ok || !includeLegacy {
		return []platformcatalog.FeatureKey{key}
	}

	authorizing := make([]platformcatalog.FeatureKey, 0, len(feature.LegacyGrantingFeatures)+1)
	authorizing = append(authorizing, key)
	for _, legacy := range feature.LegacyGrantingFeatures {
		if legacy == key {
			continue
		}
		authorizing = append(authorizing, legacy)
	}

	return authorizing
}

func (r *Registry) PolicyForGraphQLRootField(
	operation string,
	field string,
	source platformcatalog.GraphQLSource,
) platformcatalog.RoutePolicy {
	rootField := platformcatalog.GraphQLRootField{Operation: operation, Field: field}
	if featureKey, ok := r.graphQLRootFields[rootField.Key()]; ok {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassProduct,
			FeatureKey:  featureKey,
		}
	}

	if graphQLSourceIsShell(source) {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassAccountShell,
		}
	}

	if featureKey, ok := r.graphQLSources[source]; ok {
		return platformcatalog.RoutePolicy{
			AccessClass: platformcatalog.RouteAccessClassProduct,
			FeatureKey:  featureKey,
		}
	}

	return platformcatalog.RoutePolicy{AccessClass: platformcatalog.RouteAccessClassUnclassified}
}

func (r *Registry) UnclassifiedGraphQLSources(
	sources []platformcatalog.GraphQLSource,
) []platformcatalog.GraphQLSource {
	unclassified := make([]platformcatalog.GraphQLSource, 0)
	seen := make(map[platformcatalog.GraphQLSource]struct{}, len(sources))
	for _, source := range sources {
		if _, duplicate := seen[source]; duplicate {
			continue
		}
		seen[source] = struct{}{}

		if graphQLSourceIsShell(source) {
			continue
		}
		if _, ok := r.graphQLSources[source]; ok {
			continue
		}
		unclassified = append(unclassified, source)
	}
	slices.SortFunc(unclassified, func(a, b platformcatalog.GraphQLSource) int {
		return strings.Compare(string(a), string(b))
	})

	return unclassified
}

func (r *Registry) FeaturesByProduct(
	productKey platformcatalog.ProductKey,
) []platformcatalog.Feature {
	features := make([]platformcatalog.Feature, 0, len(r.features))
	for key := range r.features {
		feature := r.features[key]
		if feature.ProductKey == productKey {
			features = append(features, feature)
		}
	}
	slices.SortFunc(features, func(a, b platformcatalog.Feature) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})
	return features
}

func (r *Registry) Validate() error {
	if err := r.validateProducts(); err != nil {
		return err
	}

	if err := r.validateFeatures(); err != nil {
		return err
	}

	if err := r.validateMeters(); err != nil {
		return err
	}

	if err := r.validatePacks(); err != nil {
		return err
	}

	return r.validateRoutes()
}

func (r *Registry) validatePacks() error {
	for key, pack := range r.packs {
		if key == "" {
			return errors.New("platform catalog pack key is required")
		}
		if len(pack.Features) == 0 {
			return fmt.Errorf("platform catalog pack %q must reference at least one feature", key)
		}

		seen := make(map[platformcatalog.FeatureKey]struct{}, len(pack.Features))
		for _, featureKey := range pack.Features {
			if _, ok := r.features[featureKey]; !ok {
				return fmt.Errorf(
					"platform catalog pack %q references missing feature %q",
					key,
					featureKey,
				)
			}
			if _, duplicate := seen[featureKey]; duplicate {
				return fmt.Errorf(
					"platform catalog pack %q lists feature %q more than once",
					key,
					featureKey,
				)
			}
			seen[featureKey] = struct{}{}
		}

		if err := r.validatePackClosure(key, &pack, seen); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) validatePackClosure(
	key platformcatalog.PackKey,
	pack *platformcatalog.Pack,
	included map[platformcatalog.FeatureKey]struct{},
) error {
	if pack.Standalone && len(pack.RequiresPacks) > 0 {
		return fmt.Errorf(
			"platform catalog standalone pack %q cannot require other packs",
			key,
		)
	}

	available, err := r.packProvidedFeatures(key, pack, included)
	if err != nil {
		return err
	}

	for _, featureKey := range pack.Features {
		closure := make(map[platformcatalog.FeatureKey]struct{})
		r.collectRequiredFeatures(featureKey, closure)

		for required := range closure {
			if _, ok := available[required]; !ok {
				return fmt.Errorf(
					"platform catalog pack %q includes feature %q which requires feature %q; "+
						"add it to the pack or declare a pack that provides it in requiresPacks",
					key,
					featureKey,
					required,
				)
			}
		}
	}

	return nil
}

func (r *Registry) packProvidedFeatures(
	key platformcatalog.PackKey,
	pack *platformcatalog.Pack,
	included map[platformcatalog.FeatureKey]struct{},
) (map[platformcatalog.FeatureKey]struct{}, error) {
	available := make(map[platformcatalog.FeatureKey]struct{}, len(included))
	for featureKey := range included {
		available[featureKey] = struct{}{}
	}

	for _, requiredKey := range pack.RequiresPacks {
		if requiredKey == key {
			return nil, fmt.Errorf("platform catalog pack %q cannot require itself", key)
		}
	}

	visited := make(map[platformcatalog.PackKey]struct{}, len(pack.RequiresPacks)+1)
	visited[key] = struct{}{}
	queue := append([]platformcatalog.PackKey(nil), pack.RequiresPacks...)
	for len(queue) > 0 {
		requiredKey := queue[0]
		queue = queue[1:]

		if requiredKey == key {
			return nil, fmt.Errorf(
				"platform catalog pack %q takes part in a pack requirement cycle",
				key,
			)
		}
		if _, seen := visited[requiredKey]; seen {
			continue
		}
		visited[requiredKey] = struct{}{}

		requiredPack, ok := r.packs[requiredKey]
		if !ok {
			return nil, fmt.Errorf(
				"platform catalog pack %q requires missing pack %q",
				key,
				requiredKey,
			)
		}
		for _, featureKey := range requiredPack.Features {
			available[featureKey] = struct{}{}
		}
		queue = append(queue, requiredPack.RequiresPacks...)
	}

	return available, nil
}

func (r *Registry) registerProvider(provider platformcatalog.CatalogProvider) error {
	for _, product := range provider.Products() {
		if _, exists := r.products[product.Key]; exists {
			return fmt.Errorf("platform catalog duplicate product %q", product.Key)
		}
		r.products[product.Key] = product
	}

	providerFeatures := provider.Features()
	for i := range providerFeatures {
		feature := providerFeatures[i]
		if _, exists := r.features[feature.Key]; exists {
			return fmt.Errorf("platform catalog duplicate feature %q", feature.Key)
		}
		r.features[feature.Key] = feature

		if err := r.registerGraphQLOwnership(&feature); err != nil {
			return err
		}
	}

	for _, meter := range provider.Meters() {
		if _, exists := r.meters[meter.Key]; exists {
			return fmt.Errorf("platform catalog duplicate meter %q", meter.Key)
		}
		r.meters[meter.Key] = meter
	}

	packProvider, ok := provider.(platformcatalog.PackProvider)
	if !ok {
		return nil
	}

	for _, pack := range packProvider.Packs() {
		if _, exists := r.packs[pack.Key]; exists {
			return fmt.Errorf("platform catalog duplicate pack %q", pack.Key)
		}
		r.packs[pack.Key] = pack
	}

	return nil
}

func (r *Registry) registerGraphQLOwnership(feature *platformcatalog.Feature) error {
	for _, source := range feature.GraphQLSources {
		if graphQLSourceIsShell(source) {
			return fmt.Errorf(
				"platform catalog feature %q claims GraphQL shell source %q",
				feature.Key,
				source,
			)
		}
		if owner, exists := r.graphQLSources[source]; exists {
			return fmt.Errorf(
				"platform catalog GraphQL source %q is assigned to both feature %q and feature %q",
				source,
				owner,
				feature.Key,
			)
		}
		r.graphQLSources[source] = feature.Key
	}

	for _, field := range feature.GraphQLRootFields {
		if field.Operation != platformcatalog.GraphQLOperationQuery &&
			field.Operation != platformcatalog.GraphQLOperationMutation {
			return fmt.Errorf(
				"platform catalog feature %q GraphQL root field %q has invalid operation %q",
				feature.Key,
				field.Field,
				field.Operation,
			)
		}
		if owner, exists := r.graphQLRootFields[field.Key()]; exists {
			return fmt.Errorf(
				"platform catalog GraphQL root field %q is assigned to both feature %q and feature %q",
				field.Key(),
				owner,
				feature.Key,
			)
		}
		r.graphQLRootFields[field.Key()] = feature.Key
	}

	return nil
}

func (r *Registry) validateProducts() error {
	for key, product := range r.products {
		if key == "" {
			return errors.New("platform catalog product key is required")
		}

		if err := r.validateProductFeatures(key, product.Features); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) validateProductFeatures(
	productKey platformcatalog.ProductKey,
	featureKeys []platformcatalog.FeatureKey,
) error {
	for _, featureKey := range featureKeys {
		feature, ok := r.features[featureKey]
		if !ok {
			return fmt.Errorf(
				"platform catalog product %q references missing feature %q",
				productKey,
				featureKey,
			)
		}
		if feature.ProductKey != productKey {
			return fmt.Errorf(
				"platform catalog product %q references feature %q owned by product %q",
				productKey,
				featureKey,
				feature.ProductKey,
			)
		}
	}

	return nil
}

func (r *Registry) validateFeatures() error {
	for key := range r.features {
		feature := r.features[key]
		if _, ok := r.products[feature.ProductKey]; !ok {
			return fmt.Errorf(
				"platform catalog feature %q references missing product %q",
				key,
				feature.ProductKey,
			)
		}

		if err := r.validateRequiredFeatures(key, feature.RequiresFeatures); err != nil {
			return err
		}

		if err := r.validateLegacyGrantingFeatures(
			key,
			feature.LegacyGrantingFeatures,
		); err != nil {
			return err
		}

		if err := r.validateFeatureMeters(key, feature.Meters); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) validateRequiredFeatures(
	featureKey platformcatalog.FeatureKey,
	requiredKeys []platformcatalog.FeatureKey,
) error {
	for _, requiredKey := range requiredKeys {
		if requiredKey == featureKey {
			return fmt.Errorf("platform catalog feature %q cannot require itself", featureKey)
		}
		if _, ok := r.features[requiredKey]; !ok {
			return fmt.Errorf(
				"platform catalog feature %q requires missing feature %q",
				featureKey,
				requiredKey,
			)
		}
	}

	return nil
}

func (r *Registry) validateLegacyGrantingFeatures(
	featureKey platformcatalog.FeatureKey,
	legacyKeys []platformcatalog.FeatureKey,
) error {
	for _, legacyKey := range legacyKeys {
		if legacyKey == featureKey {
			return fmt.Errorf(
				"platform catalog feature %q cannot be legacy granted by itself",
				featureKey,
			)
		}
		if _, ok := r.features[legacyKey]; !ok {
			return fmt.Errorf(
				"platform catalog feature %q is legacy granted by missing feature %q",
				featureKey,
				legacyKey,
			)
		}
	}

	return nil
}

func (r *Registry) validateFeatureMeters(
	featureKey platformcatalog.FeatureKey,
	meterKeys []platformcatalog.MeterKey,
) error {
	for _, meterKey := range meterKeys {
		meter, ok := r.meters[meterKey]
		if !ok {
			return fmt.Errorf(
				"platform catalog feature %q references missing meter %q",
				featureKey,
				meterKey,
			)
		}
		if meter.FeatureKey != "" && meter.FeatureKey != featureKey {
			return fmt.Errorf(
				"platform catalog feature %q references meter %q owned by feature %q",
				featureKey,
				meterKey,
				meter.FeatureKey,
			)
		}
	}

	return nil
}

func (r *Registry) validateMeters() error {
	for key, meter := range r.meters {
		if _, ok := r.products[meter.ProductKey]; !ok {
			return fmt.Errorf(
				"platform catalog meter %q references missing product %q",
				key,
				meter.ProductKey,
			)
		}
		if meter.FeatureKey == "" {
			continue
		}
		if _, ok := r.features[meter.FeatureKey]; !ok {
			return fmt.Errorf(
				"platform catalog meter %q references missing feature %q",
				key,
				meter.FeatureKey,
			)
		}
	}

	return nil
}

func (r *Registry) validateRoutes() error {
	routeOwners := make(map[string]platformcatalog.FeatureKey)
	for _, feature := range r.features {
		for _, route := range feature.Routes {
			method := strings.ToUpper(strings.TrimSpace(route.Method))
			path := normalizeRoutePath(route.Path)
			if method == "" {
				return fmt.Errorf(
					"platform catalog feature %q route method is required",
					feature.Key,
				)
			}
			if path == "" {
				return fmt.Errorf("platform catalog feature %q route path is required", feature.Key)
			}

			routeKey := method + " " + path
			owner, exists := routeOwners[routeKey]
			if exists {
				return fmt.Errorf(
					"platform catalog route %q is assigned to both feature %q and feature %q",
					routeKey,
					owner,
					feature.Key,
				)
			}
			routeOwners[routeKey] = feature.Key
		}
	}

	return nil
}

func normalizeRoutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func routeMatches(method, routePattern string, ref platformcatalog.RouteRef) bool {
	refMethod := strings.ToUpper(strings.TrimSpace(ref.Method))
	if refMethod != "" && refMethod != "*" && refMethod != method {
		return false
	}

	refPath := normalizeRoutePath(ref.Path)
	if refPath == "" {
		return false
	}
	if before, ok := strings.CutSuffix(refPath, "*"); ok {
		return strings.HasPrefix(routePattern, before)
	}
	if strings.HasSuffix(refPath, "/") {
		return routePattern == refPath || strings.HasPrefix(routePattern, refPath)
	}
	return routePattern == refPath
}

func routeMatchesAny(method, routePattern string, refs []platformcatalog.RouteRef) bool {
	for _, ref := range refs {
		if routeMatches(method, routePattern, ref) {
			return true
		}
	}

	return false
}
