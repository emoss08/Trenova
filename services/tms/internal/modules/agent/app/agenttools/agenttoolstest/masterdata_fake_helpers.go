package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type FakeMaster[T any] struct {
	Guard   *WriteGuard
	Stored  *T
	Refusal error
	Created *T
	Updated *T
	Status  any
	SetID   func(*T)
}

func (f *FakeMaster[T]) MasterGet() (*T, error) {
	copied := *f.Stored

	return &copied, nil
}

func (f *FakeMaster[T]) MasterPlanCreate(_ context.Context, entity *T) (*T, error) {
	if f.Refusal != nil {
		return nil, f.Refusal
	}

	return entity, nil
}

func (f *FakeMaster[T]) MasterCreate(
	ctx context.Context,
	entity *T,
	_ *permissionports.RequestActor,
) (*T, error) {
	if err := f.Guard.Write(); err != nil {
		return nil, err
	}
	planned, err := f.MasterPlanCreate(ctx, entity)
	if err != nil {
		return nil, err
	}
	f.SetID(planned)
	f.Created = planned

	return planned, nil
}

func (f *FakeMaster[T]) MasterPlanUpdate(
	_ context.Context,
	entity *T,
) (*kernelports.RecordChange[T], error) {
	if f.Refusal != nil {
		return nil, f.Refusal
	}

	return &kernelports.RecordChange[T]{Before: f.Stored, After: entity}, nil
}

func (f *FakeMaster[T]) MasterUpdate(
	_ context.Context,
	entity *T,
	_ *permissionports.RequestActor,
) (*T, error) {
	if err := f.Guard.Write(); err != nil {
		return nil, err
	}
	f.Updated = entity

	return entity, nil
}
