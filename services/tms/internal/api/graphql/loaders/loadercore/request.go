package loadercore

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"sync"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

const SourceGroup = "graphql_loader_sources"

var ErrNotAttached = errors.New("graphql loaders are not attached to this request")

type Source interface {
	Attach(ctx context.Context, tenantInfo pagination.TenantInfo) context.Context
}

type Key[F Source, V any] struct {
	source reflect.Type
	build  func(F, pagination.TenantInfo) V
}

func NewKey[F Source, V any](build func(F, pagination.TenantInfo) V) *Key[F, V] {
	return &Key[F, V]{source: reflect.TypeFor[F](), build: build}
}

type RegistryParams struct {
	fx.In

	Sources []Source `group:"graphql_loader_sources"`
}

type Registry struct {
	sources map[reflect.Type]Source
}

func NewRegistry(p RegistryParams) *Registry {
	sources := make(map[reflect.Type]Source, len(p.Sources))
	for _, source := range p.Sources {
		sources[reflect.TypeOf(source)] = source
	}

	return &Registry{sources: sources}
}

func (r *Registry) Attach(ctx context.Context, tenantInfo pagination.TenantInfo) context.Context {
	return context.WithValue(
		ctx,
		requestKey{},
		&request{tenantInfo: tenantInfo, sources: r.sources},
	)
}

type requestKey struct{}

type request struct {
	tenantInfo pagination.TenantInfo
	sources    map[reflect.Type]Source

	mu     sync.Mutex
	values map[any]any
}

func Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sources ...Source,
) context.Context {
	next := &request{
		tenantInfo: tenantInfo,
		sources:    make(map[reflect.Type]Source, len(sources)),
		values:     make(map[any]any),
	}
	if current, ok := fromContext(ctx); ok {
		current.mu.Lock()
		maps.Copy(next.sources, current.sources)
		maps.Copy(next.values, current.values)
		current.mu.Unlock()
	}
	for _, source := range sources {
		next.sources[reflect.TypeOf(source)] = source
	}

	return context.WithValue(ctx, requestKey{}, next)
}

func Available(ctx context.Context) bool {
	_, ok := fromContext(ctx)
	return ok
}

func fromContext(ctx context.Context) (*request, bool) {
	r, ok := ctx.Value(requestKey{}).(*request)
	return r, ok && r != nil
}

func Get[F Source, V any](ctx context.Context, k *Key[F, V]) (V, bool) {
	var zero V
	r, ok := fromContext(ctx)
	if !ok {
		return zero, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if value, built := r.values[k]; built {
		typed, ok := value.(V)
		return typed, ok
	}
	source, registered := r.sources[k.source].(F)
	if !registered {
		return zero, false
	}
	value := k.build(source, r.tenantInfo)
	if r.values == nil {
		r.values = make(map[any]any)
	}
	r.values[k] = value

	return value, true
}

func Load[F Source, V any](
	ctx context.Context,
	k *Key[F, *dataloadgen.Loader[string, V]],
	key string,
) (V, error) {
	loader, ok := Get(ctx, k)
	if !ok {
		var zero V
		return zero, ErrNotAttached
	}

	return loader.Load(ctx, key)
}

func LoadAll[F Source, V any](
	ctx context.Context,
	k *Key[F, *dataloadgen.Loader[string, V]],
	keys []string,
) ([]V, error) {
	loader, ok := Get(ctx, k)
	if !ok {
		return nil, ErrNotAttached
	}

	return loader.LoadAll(ctx, keys)
}

func LoadThunk[F Source, V any](
	ctx context.Context,
	k *Key[F, *dataloadgen.Loader[string, V]],
	key string,
) func() (V, error) {
	loader, ok := Get(ctx, k)
	if !ok {
		return func() (V, error) {
			var zero V
			return zero, ErrNotAttached
		}
	}

	return loader.LoadThunk(ctx, key)
}
