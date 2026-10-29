//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediladingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject/ladingprojectdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxVersionsListed = 100

var (
	projectErrors = dberror.Mapping{Entity: "LadingProject"}
	versionErrors = dberror.Mapping{Entity: "LadingVersion"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type projects struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProjects(p Params) ladingports.LadingProjectRepository {
	return &projects{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-lading-project-repository"),
	}
}

func (r *projects) GetProject(
	ctx context.Context,
	req ladingports.GetLadingProjectRequest,
) (*ladingproject.LadingProject, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingProject, error) {
		p := ladingprojectdb.LadingProjects
		project, err := p.Where(
			keel.When(req.ID.IsNotNil(), p.ID.Eq(req.ID)),
			keel.When(req.PartnerID.IsNotNil(), p.EDIPartnerID.Eq(req.PartnerID)),
		).Load(p.Partner()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, projectErrors.Map(err, req.ID.String()+req.PartnerID.String())
		}
		return project, nil
	})
}

func (r *projects) ListProjects(
	ctx context.Context,
	req *ladingports.ListLadingProjectsRequest,
) (*pagination.ListResult[*ladingproject.LadingProject], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ladingproject.LadingProject], error) {
			p := ladingprojectdb.LadingProjects
			search := ""
			if req.Filter != nil {
				search = req.Filter.Query
			}
			q := p.Where(keel.When(search != "", keel.Or(
				p.Partner().Code.ContainsFold(search),
				p.Partner().Name.ContainsFold(search),
			))).Load(p.Partner()).Order(p.UpdatedAt.Desc())
			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func (r *projects) CreateProject(
	ctx context.Context,
	req *ladingports.CreateLadingProjectRequest,
) (*ladingproject.LadingProject, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingProject, error) {
		db := r.db.Keel(ctx)
		project := req.Project
		project.NextVersionNumber = 2
		if project.Settings == nil {
			project.Settings = map[string]any{}
		}
		if err := ladingprojectdb.LadingProjects.Create(ctx, db, project); err != nil {
			return nil, projectErrors.Map(err, project.EDIPartnerID.String())
		}
		if err := r.putSamples(ctx, project, req.Samples); err != nil {
			return nil, err
		}
		draft := req.Draft
		draft.ProjectID = project.ID
		draft.OrganizationID = project.OrganizationID
		draft.BusinessUnitID = project.BusinessUnitID
		draft.Number = 1
		if err := ladingprojectdb.LadingVersions.Create(ctx, db, draft); err != nil {
			return nil, versionErrors.Map(err, project.ID.String())
		}
		return project, nil
	})
}

func (r *projects) UpdateProjectSettings(
	ctx context.Context,
	project *ladingproject.LadingProject,
) (*ladingproject.LadingProject, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingProject, error) {
		p := ladingprojectdb.LadingProjects
		if err := p.Save(ctx, r.db.Keel(ctx), project, p.Settings); err != nil {
			return nil, projectErrors.Map(err, project.ID.String())
		}
		return project, nil
	})
}

func (r *projects) SetShadowVersion(
	ctx context.Context,
	project *ladingproject.LadingProject,
) (*ladingproject.LadingProject, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingProject, error) {
		p := ladingprojectdb.LadingProjects
		if err := p.Save(ctx, r.db.Keel(ctx), project, p.ShadowVersionID); err != nil {
			return nil, projectErrors.Map(err, project.ID.String())
		}
		return project, nil
	})
}

func (r *projects) GetVersion(
	ctx context.Context,
	req ladingports.GetLadingVersionRequest,
) (*ladingproject.LadingVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingVersion, error) {
		v := ladingprojectdb.LadingVersions
		version, err := v.Where(v.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, versionErrors.Map(err, req.ID.String())
		}
		return version, nil
	})
}

func (r *projects) ListVersions(
	ctx context.Context,
	req ladingports.ListLadingVersionsRequest,
) ([]*ladingproject.LadingVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ladingproject.LadingVersion, error) {
		v := ladingprojectdb.LadingVersions
		limit := req.Limit
		if limit <= 0 || limit > maxVersionsListed {
			limit = maxVersionsListed
		}
		return v.Omit(v.Files, v.Samples, v.Manifest).
			Where(v.ProjectID.Eq(req.ProjectID)).
			Order(v.Number.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *projects) CreateVersion(
	ctx context.Context,
	version *ladingproject.LadingVersion,
	samples []*ladingproject.LadingSample,
) (*ladingproject.LadingVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingVersion, error) {
		db := r.db.Keel(ctx)
		p := ladingprojectdb.LadingProjects
		numbered, err := p.Where(p.ID.Eq(version.ProjectID)).
			UpdateReturning(ctx, db, p.NextVersionNumber.Increment(1))
		if err != nil {
			return nil, projectErrors.Map(err, version.ProjectID.String())
		}
		if len(numbered) != 1 {
			return nil, projectErrors.Map(keel.ErrNotFound, version.ProjectID.String())
		}
		project := numbered[0]
		version.Number = project.NextVersionNumber - 1
		if err = r.putSamples(ctx, project, samples); err != nil {
			return nil, err
		}
		if err = ladingprojectdb.LadingVersions.Create(ctx, db, version); err != nil {
			return nil, versionErrors.Map(err, version.ProjectID.String())
		}
		return version, nil
	})
}

func (r *projects) SaveDraft(
	ctx context.Context,
	version *ladingproject.LadingVersion,
	samples []*ladingproject.LadingSample,
) (*ladingproject.LadingVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingVersion, error) {
		project := &ladingproject.LadingProject{
			ID:             version.ProjectID,
			OrganizationID: version.OrganizationID,
			BusinessUnitID: version.BusinessUnitID,
		}
		if err := r.putSamples(ctx, project, samples); err != nil {
			return nil, err
		}
		v := ladingprojectdb.LadingVersions
		err := v.Save(
			ctx,
			r.db.Keel(ctx),
			version,
			v.Files,
			v.Samples,
			v.Manifest,
			v.Readiness,
			v.Notes,
			v.ErrorCount,
			v.WarningCount,
		)
		if err != nil {
			return nil, versionErrors.Map(err, version.ID.String())
		}
		return version, nil
	})
}

func (r *projects) UpdateVersionStatus(
	ctx context.Context,
	version *ladingproject.LadingVersion,
) (*ladingproject.LadingVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ladingproject.LadingVersion, error) {
		v := ladingprojectdb.LadingVersions
		err := v.Save(
			ctx,
			r.db.Keel(ctx),
			version,
			v.Status,
			v.Readiness,
			v.ShadowUntil,
			v.CertifiedByID,
			v.CertifiedAt,
			v.SupersededAt,
			v.ArchivedAt,
		)
		if err != nil {
			return nil, versionErrors.Map(err, version.ID.String())
		}
		return version, nil
	})
}

func (r *projects) ActivateVersion(
	ctx context.Context,
	req ladingports.ActivateLadingVersionRequest,
) (*ladingports.ActivatedLadingVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*ladingports.ActivatedLadingVersion, error) {
			db := r.db.Keel(ctx)
			p := ladingprojectdb.LadingProjects
			v := ladingprojectdb.LadingVersions
			project, err := p.Where(p.ID.Eq(req.ProjectID)).Lock(keel.ForUpdate).One(ctx, db)
			if err != nil {
				return nil, projectErrors.Map(err, req.ProjectID.String())
			}
			now := timeutils.NowUnix()
			result := &ladingports.ActivatedLadingVersion{}
			if project.ActiveVersionID.IsNotNil() && project.ActiveVersionID != req.VersionID {
				previous, getErr := v.Omit(v.Files, v.Samples, v.Manifest).
					Where(v.ID.Eq(project.ActiveVersionID)).
					One(ctx, db)
				if getErr != nil {
					return nil, versionErrors.Map(getErr, project.ActiveVersionID.String())
				}
				previous.Status = ladingproject.LadingVersionStatusSuperseded
				previous.SupersededAt = &now
				if err = v.Save(ctx, db, previous, v.Status, v.SupersededAt); err != nil {
					return nil, versionErrors.Map(err, previous.ID.String())
				}
				result.Superseded = previous
			}
			version, err := v.Omit(v.Files, v.Samples, v.Manifest).
				Where(v.ID.Eq(req.VersionID), v.ProjectID.Eq(req.ProjectID)).
				One(ctx, db)
			if err != nil {
				return nil, versionErrors.Map(err, req.VersionID.String())
			}
			version.Status = ladingproject.LadingVersionStatusActive
			version.ActivatedAt = &now
			version.ActivatedByID = req.ActorID
			version.ShadowUntil = nil
			if err = v.Save(
				ctx,
				db,
				version,
				v.Status,
				v.ActivatedAt,
				v.ActivatedByID,
				v.ShadowUntil,
			); err != nil {
				return nil, versionErrors.Map(err, version.ID.String())
			}
			project.ActiveVersionID = version.ID
			if project.ShadowVersionID == version.ID {
				project.ShadowVersionID = ""
			}
			if err = p.Save(ctx, db, project, p.ActiveVersionID, p.ShadowVersionID); err != nil {
				return nil, projectErrors.Map(err, project.ID.String())
			}
			result.Version = version
			return result, nil
		},
	)
}

func (r *projects) GetSamples(
	ctx context.Context,
	req ladingports.GetLadingSamplesRequest,
) ([]*ladingproject.LadingSample, error) {
	if len(req.Digests) == 0 {
		return nil, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ladingproject.LadingSample, error) {
		s := ladingprojectdb.LadingSamples
		return s.Where(s.ProjectID.Eq(req.ProjectID), s.Digest.In(req.Digests...)).
			Limit(len(req.Digests)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *projects) GetRunnableVersions(
	ctx context.Context,
	req ladingports.GetRunnableLadingVersionRequest,
) (*ladingports.RunnableLadingVersions, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*ladingports.RunnableLadingVersions, error) {
			db := r.db.Keel(ctx)
			p := ladingprojectdb.LadingProjects
			project, err := p.Where(p.EDIPartnerID.Eq(req.PartnerID)).One(ctx, db)
			if err != nil {
				return nil, projectErrors.Map(err, req.PartnerID.String())
			}
			result := &ladingports.RunnableLadingVersions{Project: project}
			ids := make([]pulid.ID, 0, 2)
			if project.ActiveVersionID.IsNotNil() {
				ids = append(ids, project.ActiveVersionID)
			}
			if project.ShadowVersionID.IsNotNil() {
				ids = append(ids, project.ShadowVersionID)
			}
			if len(ids) == 0 {
				return result, nil
			}
			v := ladingprojectdb.LadingVersions
			versions, err := v.Omit(v.Samples).
				Where(v.ProjectID.Eq(project.ID), v.ID.In(ids...)).
				Limit(len(ids)).
				List(ctx, db)
			if err != nil {
				return nil, versionErrors.Map(err, project.ID.String())
			}
			for _, version := range versions {
				switch version.ID {
				case project.ActiveVersionID:
					result.Active = version
				case project.ShadowVersionID:
					result.Shadow = version
				}
			}
			return result, nil
		},
	)
}

func (r *projects) putSamples(
	ctx context.Context,
	project *ladingproject.LadingProject,
	samples []*ladingproject.LadingSample,
) error {
	if len(samples) == 0 {
		return nil
	}
	for _, sample := range samples {
		sample.ProjectID = project.ID
		sample.OrganizationID = project.OrganizationID
		sample.BusinessUnitID = project.BusinessUnitID
	}
	s := ladingprojectdb.LadingSamples
	if _, err := s.CreateOrSkip(
		ctx,
		r.db.Keel(ctx),
		samples,
		s.ProjectID,
		s.BusinessUnitID,
		s.OrganizationID,
		s.Digest,
	); err != nil {
		return fmt.Errorf("store Lading samples: %w", err)
	}
	return nil
}
