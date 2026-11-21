package accountguard

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type SystemActors interface {
	SystemActorID(ctx context.Context, businessUnitID pulid.ID) (pulid.ID, error)
}

func RequirePeople(
	ctx context.Context,
	actors SystemActors,
	businessUnitID pulid.ID,
	userIDs ...pulid.ID,
) error {
	if !slices.ContainsFunc(userIDs, pulid.ID.IsNotNil) {
		return nil
	}
	systemActor, err := actors.SystemActorID(ctx, businessUnitID)
	if errortypes.IsNotFoundError(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if slices.Contains(userIDs, systemActor) {
		return errUnknownUser()
	}
	return nil
}

func WithoutSystemActor(
	ctx context.Context,
	actors SystemActors,
	businessUnitID pulid.ID,
	userIDs []pulid.ID,
) ([]pulid.ID, error) {
	if len(userIDs) == 0 {
		return userIDs, nil
	}
	systemActor, err := actors.SystemActorID(ctx, businessUnitID)
	if errortypes.IsNotFoundError(err) {
		return userIDs, nil
	}
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(slices.Clone(userIDs), func(id pulid.ID) bool {
		return id == systemActor
	}), nil
}

func errUnknownUser() error {
	return errortypes.NewNotFoundError("{0} not found within your organization", "User")
}
