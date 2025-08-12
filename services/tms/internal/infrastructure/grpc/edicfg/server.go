package edicfg

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	domainedi "github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/db"
	portrepo "github.com/emoss08/trenova/internal/core/ports/repositories"
	configadapter "github.com/emoss08/trenova/shared/edi/adapter/configproto"
	configpb "github.com/emoss08/trenova/shared/edi/proto/config/v1"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the EDIConfigService gRPC API.
type Server struct {
	configpb.UnimplementedEDIConfigServiceServer
	conn db.Connection
	l    *zerolog.Logger
	repo portrepo.EDIPartnerConfigRepository
}

// NewServer creates a new EDI config gRPC server.
func NewServer(
	conn db.Connection,
	logger *zerolog.Logger,
	repo portrepo.EDIPartnerConfigRepository,
) *Server {
	log := logger.With().
		Str("service", "edicfg").
		Logger()
	return &Server{conn: conn, l: &log, repo: repo}
}

// Register attaches this service to a gRPC server.
func (s *Server) Register(grpcServer interface {
	RegisterService(*any, any)
},
) {
	// This generic Register helper is optional; prefer direct registration from callers:
	// configpb.RegisterEDIConfigServiceServer(grpcServer, s)
}

// GetPartnerConfig fetches a single partner config by ID or by (BU, Org, Name).
func (s *Server) GetPartnerConfig(
	ctx context.Context,
	req *configpb.GetPartnerConfigRequest,
) (*configpb.GetPartnerConfigResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if s.repo == nil {
		return nil, status.Error(codes.Unavailable, "repository not initialized")
	}
	bu := pulid.ID(req.GetBusinessUnitId())
	org := pulid.ID(req.GetOrganizationId())
	var pc *domainedi.PartnerConfig
	var err error
	if req.GetId() != "" {
		id := pulid.ID(req.GetId())
		pc, err = s.repo.GetByID(ctx, bu, org, id)
	} else {
		if bu.IsNil() || org.IsNil() || req.GetName() == "" {
			return nil, status.Error(codes.InvalidArgument, "provide id or (business_unit_id, organization_id, name)")
		}
		pc, err = s.repo.GetByKey(ctx, bu, org, req.GetName())
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}

	cfg := pc.ToConfigTypes()
	resp := &configpb.GetPartnerConfigResponse{
		Id:             pc.ID.String(),
		BusinessUnitId: pc.BusinessUnitID.String(),
		OrganizationId: pc.OrganizationID.String(),
		Config:         configadapter.ToProto(cfg),
	}
	return resp, nil
}

// ListPartnerConfigs lists partner configs filtered by BU/Org with simple paging.
// NOTE: This is a scaffold; add proper pagination as needed.
func (s *Server) ListPartnerConfigs(
	ctx context.Context,
	req *configpb.ListPartnerConfigsRequest,
) (*configpb.ListPartnerConfigsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	bu := pulid.ID(req.GetBusinessUnitId())
	org := pulid.ID(req.GetOrganizationId())
	if bu.IsNil() || org.IsNil() {
		return nil, status.Error(
			codes.InvalidArgument,
			"business_unit_id and organization_id are required",
		)
	}
	limit := int(req.GetPageSize())
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var afterName string
	var afterID pulid.ID
	if tok := strings.TrimSpace(req.GetPageToken()); tok != "" {
		if name, id, ok := decodeToken(tok); ok {
			afterName, afterID = name, pulid.ID(id)
		} else {
			return nil, status.Error(codes.InvalidArgument, "invalid page_token")
		}
	}
	if s.repo == nil {
		return nil, status.Error(codes.Unavailable, "repository not initialized")
	}
	rows, hint, err := s.repo.List(ctx, bu, org, limit, afterName, afterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query error: %v", err)
	}
	nextToken := ""
	if hint == "more" && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeToken(last.Name, last.ID.String())
	}
	out := make([]*configpb.GetPartnerConfigResponse, 0, len(rows))
	for _, pc := range rows {
		out = append(
			out,
			&configpb.GetPartnerConfigResponse{
				Id:             pc.ID.String(),
				BusinessUnitId: pc.BusinessUnitID.String(),
				OrganizationId: pc.OrganizationID.String(),
				Config:         configadapter.ToProto(pc.ToConfigTypes()),
			},
		)
	}
	return &configpb.ListPartnerConfigsResponse{Items: out, NextPageToken: nextToken}, nil
}

func encodeToken(name, id string) string {
	raw := fmt.Sprintf("%s|%s", name, id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeToken(tok string) (name, id string, ok bool) {
	b, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
