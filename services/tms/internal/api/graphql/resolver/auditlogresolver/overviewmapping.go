package auditlogresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/loaderfields"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditoverview"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditreviewservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
)

func auditOverviewToModel(
	ctx context.Context,
	o *auditoverview.Overview,
) (*gqlmodel.AuditOverview, error) {
	actors, err := auditActorsToModel(ctx, o.TopActors)
	if err != nil {
		return nil, err
	}

	activity := make([]*gqlmodel.AuditActivityBucket, 0, len(o.Activity))
	for _, bucket := range o.Activity {
		activity = append(activity, &gqlmodel.AuditActivityBucket{
			At:           int(bucket.At.Unix()),
			People:       int(bucket.People),
			Agents:       int(bucket.Agents),
			Integrations: int(bucket.Integrations),
			Jobs:         int(bucket.Jobs),
		})
	}

	return &gqlmodel.AuditOverview{
		Days:            o.Days,
		From:            int(o.From),
		To:              int(o.To),
		Timezone:        o.Timezone,
		Unit:            gqlmodel.AuditOverviewUnit(o.Unit),
		Totals:          auditFiguresToModel(o.Totals),
		PreviousEntries: int(o.PreviousEntries),
		Activity:        activity,
		Actors: &gqlmodel.AuditActorCounts{
			People:       o.Actors.People,
			Agents:       o.Actors.Agents,
			Integrations: o.Actors.Integrations,
			Jobs:         o.Actors.Jobs,
		},
		TopActors:       actors,
		OtherActors:     o.OtherActors,
		OtherEntries:    int(o.OtherEntries),
		ReviewsIncluded: o.ReviewsIncluded,
		OpenReviews:     reviewPointers(o.OpenReviews),
		OpenReviewCount: o.OpenReviewCount,
		Summary:         auditSegmentsToModel(o.Summary),
	}, nil
}

func auditFiguresToModel(f auditports.OverviewFigures) *gqlmodel.AuditOverviewFigures {
	return &gqlmodel.AuditOverviewFigures{
		Entries:        int(f.Entries),
		People:         int(f.People),
		AgentsAsked:    int(f.AgentsAsked),
		AgentsOwn:      int(f.AgentsOwn),
		Integrations:   int(f.Integrations),
		Jobs:           int(f.Jobs),
		Sensitive:      int(f.Sensitive),
		Security:       int(f.Security),
		DeletesExports: int(f.DeletesExports),
		PeopleOffHours: int(f.PeopleOffHours),
	}
}

func auditActorsToModel(
	ctx context.Context,
	actors []auditports.ActorStat,
) ([]*gqlmodel.AuditActorStat, error) {
	out := make([]*gqlmodel.AuditActorStat, 0, len(actors))
	for _, actor := range actors {
		model := &gqlmodel.AuditActorStat{
			PrincipalID:   actor.PrincipalID.String(),
			PrincipalType: actor.PrincipalType,
			Name:          actor.Name,
			Entries:       int(actor.Entries),
			Sensitive:     int(actor.Sensitive),
		}
		if audit.ActorKindOfPrincipal(actor.PrincipalType) == audit.ActorPeople {
			user, err := loaderfields.LoadUser(ctx, actor.PrincipalID)
			if err != nil {
				return nil, err
			}
			model.User = user
		}
		out = append(out, model)
	}

	return out, nil
}

func auditSegmentsToModel(segments []auditoverview.Segment) []*gqlmodel.AuditOverviewSegment {
	out := make([]*gqlmodel.AuditOverviewSegment, 0, len(segments))
	for _, segment := range segments {
		model := &gqlmodel.AuditOverviewSegment{Text: segment.Text, Strong: segment.Strong}
		if segment.Tone != "" {
			tone := segment.Tone
			model.Tone = &tone
		}
		if segment.Target != "" {
			target := gqlmodel.AuditOverviewTarget(segment.Target)
			id := segment.TargetID
			model.Target = &target
			model.TargetID = &id
		}
		out = append(out, model)
	}

	return out
}

func reviewPointers(views []auditreviewservice.View) []*auditreviewservice.View {
	out := make([]*auditreviewservice.View, 0, len(views))
	for i := range views {
		out = append(out, &views[i])
	}

	return out
}
