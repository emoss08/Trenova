package platformcatalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.uber.org/fx"
)

type RegistryParams struct {
	fx.In

	Providers []CatalogProvider `group:"platform_catalog_providers"`
}

type Registry struct {
	products map[ProductKey]Product
	features map[FeatureKey]Feature
	meters   map[MeterKey]Meter
}

func NewRegistry(p RegistryParams) (*Registry, error) {
	registry := &Registry{
		products: make(map[ProductKey]Product),
		features: make(map[FeatureKey]Feature),
		meters:   make(map[MeterKey]Meter),
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

func (r *Registry) ListProducts() []Product {
	products := make([]Product, 0, len(r.products))
	for _, product := range r.products {
		products = append(products, product)
	}
	slices.SortFunc(products, func(a, b Product) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})
	return products
}

func (r *Registry) ListFeatures() []Feature {
	features := make([]Feature, 0, len(r.features))
	for key := range r.features {
		features = append(features, r.features[key])
	}
	slices.SortFunc(features, func(a, b Feature) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})
	return features
}

func (r *Registry) ListMeters() []Meter {
	meters := make([]Meter, 0, len(r.meters))
	for _, meter := range r.meters {
		meters = append(meters, meter)
	}
	slices.SortFunc(meters, func(a, b Meter) int {
		return strings.Compare(string(a.Key), string(b.Key))
	})
	return meters
}

func (r *Registry) GetProduct(key ProductKey) (Product, bool) {
	product, ok := r.products[key]
	return product, ok
}

func (r *Registry) GetFeature(key FeatureKey) (Feature, bool) {
	feature, ok := r.features[key]
	return feature, ok
}

func (r *Registry) FeatureForRoute(method, routePattern string) (Feature, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	routePattern = normalizeRoutePath(routePattern)
	if routePattern == "" {
		return Feature{}, false
	}

	if routeMatchesAny(method, routePattern, accountShellRouteRefs()) {
		return Feature{}, false
	}

	return r.featureForRoute(method, routePattern)
}

func (r *Registry) PolicyForRoute(method, routePattern string) RoutePolicy {
	method = strings.ToUpper(strings.TrimSpace(method))
	routePattern = normalizeRoutePath(routePattern)
	if routePattern == "" {
		return RoutePolicy{AccessClass: RouteAccessClassUnclassified}
	}

	if routeMatchesAny(method, routePattern, accountShellRouteRefs()) {
		return RoutePolicy{AccessClass: RouteAccessClassAccountShell}
	}

	feature, ok := r.featureForRoute(method, routePattern)
	if !ok {
		return RoutePolicy{AccessClass: RouteAccessClassUnclassified}
	}

	return RoutePolicy{
		AccessClass: RouteAccessClassProduct,
		FeatureKey:  feature.Key,
	}
}

func (r *Registry) featureForRoute(method, routePattern string) (Feature, bool) {
	for _, feature := range r.ListFeatures() {
		for _, route := range feature.Routes {
			if routeMatches(method, routePattern, route) {
				return feature, true
			}
		}
	}

	return Feature{}, false
}

func (r *Registry) GetMeter(key MeterKey) (Meter, bool) {
	meter, ok := r.meters[key]
	return meter, ok
}

func (r *Registry) FeaturesByProduct(productKey ProductKey) []Feature {
	features := make([]Feature, 0, len(r.features))
	for key := range r.features {
		feature := r.features[key]
		if feature.ProductKey == productKey {
			features = append(features, feature)
		}
	}
	slices.SortFunc(features, func(a, b Feature) int {
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

	return r.validateRoutes()
}

func (r *Registry) registerProvider(provider CatalogProvider) error {
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
	}

	for _, meter := range provider.Meters() {
		if _, exists := r.meters[meter.Key]; exists {
			return fmt.Errorf("platform catalog duplicate meter %q", meter.Key)
		}
		r.meters[meter.Key] = meter
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

func (r *Registry) validateProductFeatures(productKey ProductKey, featureKeys []FeatureKey) error {
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

		if err := r.validateFeatureMeters(key, feature.Meters); err != nil {
			return err
		}
	}

	return nil
}

func (r *Registry) validateRequiredFeatures(
	featureKey FeatureKey,
	requiredKeys []FeatureKey,
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

func (r *Registry) validateFeatureMeters(featureKey FeatureKey, meterKeys []MeterKey) error {
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
	routeOwners := make(map[string]FeatureKey)
	for _, feature := range r.features {
		for _, route := range feature.Routes {
			method := strings.ToUpper(strings.TrimSpace(route.Method))
			path := normalizeRoutePath(route.Path)
			if method == "" {
				return fmt.Errorf("platform catalog feature %q route method is required", feature.Key)
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

func routeMatches(method, routePattern string, ref RouteRef) bool {
	refMethod := strings.ToUpper(strings.TrimSpace(ref.Method))
	if refMethod != "" && refMethod != "*" && refMethod != method {
		return false
	}

	refPath := normalizeRoutePath(ref.Path)
	if refPath == "" {
		return false
	}
	if strings.HasSuffix(refPath, "*") {
		return strings.HasPrefix(routePattern, strings.TrimSuffix(refPath, "*"))
	}
	if strings.HasSuffix(refPath, "/") {
		return routePattern == refPath || strings.HasPrefix(routePattern, refPath)
	}
	return routePattern == refPath
}

func routeMatchesAny(method, routePattern string, refs []RouteRef) bool {
	for _, ref := range refs {
		if routeMatches(method, routePattern, ref) {
			return true
		}
	}

	return false
}
