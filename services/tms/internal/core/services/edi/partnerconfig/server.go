package partnerconfig

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/grpcutils"
	"github.com/emoss08/trenova/shared/edi/adapter/configproto"
	configpb "github.com/emoss08/trenova/shared/edi/proto/config/v1"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServerParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
	Repo   repositories.EDIPartnerConfigRepository
}

type Server struct {
	configpb.UnimplementedEDIConfigServiceServer
	db   db.Connection
	l    *zerolog.Logger
	repo repositories.EDIPartnerConfigRepository
}

func NewServer(p ServerParams) *Server {
	log := p.Logger.With().
		Str("service", "edi-partner-config-server").
		Logger()

	return &Server{
		db:   p.DB,
		l:    &log,
		repo: p.Repo,
	}
}

func (s *Server) GetPartnerConfig(
	ctx context.Context,
	req *configpb.GetPartnerConfigRequest,
) (*configpb.GetPartnerConfigResponse, error) {
	log := s.l.With().Str("method", "GetPartnerConfig").Logger()
	if req == nil {
		log.Error().Msg("received nil request")
		return nil, status.Error(codes.InvalidArgument, "request is required to get partner config")
	}

	buID, err := pulid.MustParse(req.GetBusinessUnitId())
	if err != nil {
		log.Error().Err(err).Str("id", req.GetBusinessUnitId()).Msg("invalid business unit id")
		return nil, status.Error(codes.InvalidArgument, "business unit is not a valid pulid")
	}

	orgID, err := pulid.MustParse(req.GetOrganizationId())
	if err != nil {
		log.Error().Err(err).Str("id", req.GetOrganizationId()).Msg("invalid organization id")
		return nil, status.Error(codes.InvalidArgument, "organization is not a valid pulid")
	}

	pc, err := s.repo.GetByKey(ctx, buID, orgID, req.GetName())
	if err != nil {
		log.Error().Err(err).Str("id", req.GetId()).Msg("failed to get partner config by id")
		return nil, status.Errorf(codes.Internal, "failed to get partner config by id: %v", err)
	}

	cfg := pc.ToConfigTypes()
	resp := &configpb.GetPartnerConfigResponse{
		Id:             pc.ID.String(),
		BusinessUnitId: pc.BusinessUnitID.String(),
		OrganizationId: pc.OrganizationID.String(),
		Config:         configproto.ToProto(cfg),
	}

	return resp, nil
}

func (s *Server) ListPartnerConfigs(
	ctx context.Context,
	req *configpb.ListPartnerConfigsRequest,
) (*configpb.ListPartnerConfigsResponse, error) {
	log := s.l.With().Str("method", "ListPartnerConfigs").Logger()
	if req == nil {
		log.Error().Msg("received nil request")
		return nil, status.Error(
			codes.InvalidArgument,
			"request is required to list partner configs",
		)
	}

	buID, err := pulid.MustParse(req.GetBusinessUnitId())
	if err != nil {
		log.Error().Err(err).Msg("invalid business unit id")
		return nil, status.Error(codes.InvalidArgument, "business unit is not a valid pulid")
	}

	orgID, err := pulid.MustParse(req.GetOrganizationId())
	if err != nil {
		log.Error().Err(err).Msg("invalid organization id")
		return nil, status.Error(codes.InvalidArgument, "organization is not a valid pulid")
	}

	limit := int(req.GetPageSize())
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var afterName string
	var afterID pulid.ID
	if tok := strings.TrimSpace(req.GetPageToken()); tok != "" {
		if name, id, ok := grpcutils.DecodeToken(tok); ok {
			afterName, afterID = name, pulid.ID(id)
		} else {
			return nil, status.Error(codes.InvalidArgument, "invalid page_token")
		}
	}

	rows, hint, err := s.repo.List(ctx, buID, orgID, limit, afterName, afterID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list partner configs")
		return nil, status.Errorf(codes.Internal, "failed to list partner configs: %v", err)
	}

	nextToken := ""
	if hint == "more" && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = grpcutils.EncodeToken(last.Name, last.ID.String())
	}

	out := make([]*configpb.GetPartnerConfigResponse, 0, len(rows))
	for _, pc := range rows {
		out = append(out, &configpb.GetPartnerConfigResponse{
			Id:             pc.ID.String(),
			BusinessUnitId: pc.BusinessUnitID.String(),
			OrganizationId: pc.OrganizationID.String(),
			Config:         configproto.ToProto(pc.ToConfigTypes()),
		})
	}

	return &configpb.ListPartnerConfigsResponse{
		Items:         out,
		NextPageToken: nextToken,
	}, nil
}
