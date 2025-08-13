package endpoints

import (
	"context"

	"github.com/emoss08/trenova/shared/edi/internal/core/services"
	"github.com/go-kit/kit/endpoint"
)

// ImportProfileRequest represents the request for importing a profile
type ImportProfileRequest struct {
	ProfileJSON []byte `json:"profile_json"`
}

// ImportProfileResponse represents the response from importing a profile
type ImportProfileResponse struct {
	PartnerID string `json:"partner_id,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// MakeImportProfileEndpoint creates an endpoint for importing profiles
func MakeImportProfileEndpoint(svc *services.ProfileService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ImportProfileRequest)
		partnerID, err := svc.ImportProfile(ctx, req.ProfileJSON)
		if err != nil {
			return ImportProfileResponse{Error: err.Error()}, nil
		}
		return ImportProfileResponse{
			PartnerID: partnerID,
			Message:   "Profile imported successfully",
		}, nil
	}
}

// ListProfilesRequest represents the request for listing profiles
type ListProfilesRequest struct {
	ActiveOnly bool `json:"active_only"`
}

// ListProfilesResponse represents the response for listing profiles
type ListProfilesResponse struct {
	Profiles []ProfileSummary `json:"profiles,omitempty"`
	Count    int              `json:"count"`
	Error    string           `json:"error,omitempty"`
}

// ProfileSummary represents a summary of a profile
type ProfileSummary struct {
	PartnerID   string `json:"partner_id"`
	PartnerName string `json:"partner_name"`
	Active      bool   `json:"active"`
	Description string `json:"description,omitempty"`
}

// MakeListProfilesEndpoint creates an endpoint for listing profiles
func MakeListProfilesEndpoint(svc *services.ProfileService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListProfilesRequest)
		profiles, err := svc.ListProfiles(ctx, req.ActiveOnly)
		if err != nil {
			return ListProfilesResponse{Error: err.Error()}, nil
		}

		summaries := make([]ProfileSummary, 0, len(profiles))
		for _, p := range profiles {
			summaries = append(summaries, ProfileSummary{
				PartnerID:   p.PartnerID,
				PartnerName: p.PartnerName,
				Active:      p.Active,
				Description: p.Description,
			})
		}

		return ListProfilesResponse{
			Profiles: summaries,
			Count:    len(summaries),
		}, nil
	}
}

// GetProfileRequest represents the request for getting a profile
type GetProfileRequest struct {
	PartnerID string `json:"partner_id"`
}

// GetProfileResponse represents the response for getting a profile
type GetProfileResponse struct {
	Profile interface{} `json:"profile,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// MakeGetProfileEndpoint creates an endpoint for retrieving a profile
func MakeGetProfileEndpoint(svc *services.ProfileService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetProfileRequest)
		profile, err := svc.GetProfile(ctx, req.PartnerID)
		if err != nil {
			return GetProfileResponse{Error: err.Error()}, nil
		}

		return GetProfileResponse{Profile: profile}, nil
	}
}

// DeleteProfileRequest represents the request for deleting a profile
type DeleteProfileRequest struct {
	PartnerID string `json:"partner_id"`
}

// DeleteProfileResponse represents the response from deleting a profile
type DeleteProfileResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// MakeDeleteProfileEndpoint creates an endpoint for deleting profiles
func MakeDeleteProfileEndpoint(svc *services.ProfileService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DeleteProfileRequest)
		err := svc.DeleteProfile(ctx, req.PartnerID)
		if err != nil {
			return DeleteProfileResponse{Error: err.Error()}, nil
		}
		return DeleteProfileResponse{Message: "Profile deleted successfully"}, nil
	}
}

// ProfileEndpoints collects all profile-related endpoints
type ProfileEndpoints struct {
	ImportProfileEndpoint endpoint.Endpoint
	ListProfilesEndpoint  endpoint.Endpoint
	GetProfileEndpoint    endpoint.Endpoint
	DeleteProfileEndpoint endpoint.Endpoint
}

// NewProfileEndpoints returns a ProfileEndpoints struct
func NewProfileEndpoints(svc *services.ProfileService) ProfileEndpoints {
	return ProfileEndpoints{
		ImportProfileEndpoint: MakeImportProfileEndpoint(svc),
		ListProfilesEndpoint:  MakeListProfilesEndpoint(svc),
		GetProfileEndpoint:    MakeGetProfileEndpoint(svc),
		DeleteProfileEndpoint: MakeDeleteProfileEndpoint(svc),
	}
}

// Chain applies middlewares to all profile endpoints
func (e ProfileEndpoints) Chain(mw ...endpoint.Middleware) ProfileEndpoints {
	for _, m := range mw {
		e.ImportProfileEndpoint = m(e.ImportProfileEndpoint)
		e.ListProfilesEndpoint = m(e.ListProfilesEndpoint)
		e.GetProfileEndpoint = m(e.GetProfileEndpoint)
		e.DeleteProfileEndpoint = m(e.DeleteProfileEndpoint)
	}
	return e
}