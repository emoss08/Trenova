package edisetup

import (
	"context"
	"maps"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/smime"
)

func (s *Service) ListCommunicationProfiles(
	ctx context.Context,
	req *edipartnerports.ListEDICommunicationProfilesRequest,
) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
	result, err := s.profileRepo.ListProfiles(ctx, req)
	if err != nil {
		return nil, err
	}

	for idx := range result.Items {
		result.Items[idx] = ediprofilesecret.WithSecretState(result.Items[idx])
	}
	return result, nil
}

func (s *Service) SelectCommunicationProfileOptions(
	ctx context.Context,
	req *edipartnerports.EDICommunicationProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
	result, err := s.profileRepo.SelectProfileOptions(ctx, req)
	if err != nil {
		return nil, err
	}

	for idx := range result.Items {
		result.Items[idx] = ediprofilesecret.WithSecretState(result.Items[idx])
	}
	return result, nil
}

func (s *Service) GetCommunicationProfile(
	ctx context.Context,
	req edipartnerports.GetEDICommunicationProfileByIDRequest,
) (*edi.EDICommunicationProfile, error) {
	profile, err := s.profileRepo.GetProfileByID(ctx, req)
	if err != nil {
		return nil, err
	}

	return ediprofilesecret.WithSecretState(profile), nil
}

func (s *Service) GetCommunicationProfilesByIDs(
	ctx context.Context,
	req edipartnerports.GetEDICommunicationProfilesByIDsRequest,
) ([]*edi.EDICommunicationProfile, error) {
	profiles, err := s.profileRepo.GetProfilesByIDs(ctx, req)
	if err != nil {
		return nil, err
	}

	for i, profile := range profiles {
		profiles[i] = ediprofilesecret.WithSecretState(profile)
	}

	return profiles, nil
}

type TestCommunicationProfileConnectionRequest struct {
	ProfileID  pulid.ID              `json:"-"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type TestCommunicationProfileConnectionResult struct {
	Success bool                                   `json:"success"`
	Checks  []editransportports.EDIConnectionCheck `json:"checks"`
}

func (s *Service) TestCommunicationProfileConnection(
	ctx context.Context,
	req *TestCommunicationProfileConnectionRequest,
) (*TestCommunicationProfileConnectionResult, error) {
	profile, err := s.profileRepo.GetProfileByID(
		ctx,
		edipartnerports.GetEDICommunicationProfileByIDRequest{
			ID:         req.ProfileID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if profile.Method == edi.ConnectionMethodInternal {
		return nil, errortypes.NewValidationError(
			"method",
			errortypes.ErrInvalidOperation,
			"Internal communication profiles do not have an external connection to test",
		)
	}
	secrets, err := s.secret.TransportSecrets(profile)
	if err != nil {
		return nil, err
	}
	checks, err := s.transport.TestConnection(
		ctx,
		profile.Method,
		&editransportports.EDITransportRequest{Profile: profile, Secrets: secrets},
	)
	if err != nil {
		return nil, errortypes.NewBusinessError(err.Error())
	}
	result := &TestCommunicationProfileConnectionResult{Success: true, Checks: checks}
	for _, check := range checks {
		if check.Status == editransportports.EDIConnectionCheckFailed {
			result.Success = false
			break
		}
	}
	return result, nil
}

func (s *Service) InspectAS2Certificate(pemData string) (*smime.CertificateSummary, error) {
	trimmed := strings.TrimSpace(pemData)
	if trimmed == "" {
		return nil, errortypes.NewValidationError(
			"certificate",
			errortypes.ErrRequired,
			"Certificate PEM is required",
		)
	}
	certificate, err := smime.ParseCertificate([]byte(trimmed))
	if err != nil {
		return nil, errortypes.NewValidationError(
			"certificate",
			errortypes.ErrInvalid,
			"Certificate must be a valid PEM certificate",
		)
	}
	summary := smime.SummarizeCertificate(certificate)
	return &summary, nil
}

func (s *Service) CreateCommunicationProfile(
	ctx context.Context,
	req *UpsertEDICommunicationProfileRequest,
	actor *permissionports.RequestActor,
) (*edi.EDICommunicationProfile, error) {
	entity, err := s.buildCommunicationProfile(ctx, req, nil)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateCommunicationProfile(entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.profileRepo.CreateProfile(ctx, entity)
	if err != nil {
		return nil, mapEDICommunicationProfileConstraint(err)
	}

	s.logAction(
		created,
		actor,
		permission.OpCreate,
		nil,
		created,
		"EDI communication profile created",
	)
	return ediprofilesecret.WithSecretState(created), nil
}

func (s *Service) UpdateCommunicationProfile(
	ctx context.Context,
	req *UpsertEDICommunicationProfileRequest,
	actor *permissionports.RequestActor,
) (*edi.EDICommunicationProfile, error) {
	existing, err := s.profileRepo.GetProfileByID(
		ctx,
		edipartnerports.GetEDICommunicationProfileByIDRequest{
			ID:         req.ProfileID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	entity, err := s.buildCommunicationProfile(ctx, req, existing)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateCommunicationProfile(entity); multiErr != nil {
		return nil, multiErr
	}

	original := *existing
	updated, err := s.profileRepo.UpdateProfile(ctx, entity)
	if err != nil {
		return nil, mapEDICommunicationProfileConstraint(err)
	}

	s.logAction(
		updated,
		actor,
		permission.OpUpdate,
		&original,
		updated,
		"EDI communication profile updated",
	)
	return ediprofilesecret.WithSecretState(updated), nil
}

func (s *Service) buildCommunicationProfile(
	ctx context.Context,
	req *UpsertEDICommunicationProfileRequest,
	existing *edi.EDICommunicationProfile,
) (*edi.EDICommunicationProfile, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"",
			errortypes.ErrRequired,
			"EDI communication profile is required",
		)
	}

	method := req.Method
	if method == "" && existing != nil {
		method = existing.Method
	}
	if method == "" {
		method = edi.ConnectionMethodInternal
	}

	status := domaintypes.Status(req.Status)
	if status == "" {
		status = domaintypes.StatusActive
	}

	config := normalizeProfileConfig(req.Config)
	if existing != nil && len(config) == 0 {
		config = existing.Config
	}

	profileID := req.ProfileID
	if profileID.IsNil() {
		if existing != nil {
			profileID = existing.ID
		} else {
			profileID = pulid.MustNew("edicp_")
		}
	}

	secrets, err := s.resolveProfileSecrets(req, profileID, existing)
	if err != nil {
		return nil, err
	}

	entity := &edi.EDICommunicationProfile{
		ID:               profileID,
		BusinessUnitID:   req.TenantInfo.BuID,
		OrganizationID:   req.TenantInfo.OrgID,
		EDIConnectionID:  req.EDIConnectionID,
		EDIPartnerID:     req.EDIPartnerID,
		Method:           method,
		Status:           status,
		Name:             strings.TrimSpace(req.Name),
		Description:      strings.TrimSpace(req.Description),
		Config:           config,
		EncryptedSecrets: secrets,
		Version:          req.Version,
	}
	if existing != nil {
		entity.ID = existing.ID
		entity.Version = existing.Version
		if entity.EDIConnectionID.IsNil() {
			entity.EDIConnectionID = existing.EDIConnectionID
		}
		if entity.EDIPartnerID.IsNil() {
			entity.EDIPartnerID = existing.EDIPartnerID
		}
		if entity.Name == "" {
			entity.Name = existing.Name
		}
	}

	if method == edi.ConnectionMethodInternal {
		if entity.EDIConnectionID.IsNil() && entity.EDIPartnerID.IsNil() {
			return nil, errortypes.NewValidationError(
				"ediConnectionId",
				errortypes.ErrRequired,
				"Internal profiles require a connection or partner",
			)
		}
	}

	if err = s.ensureProfileReferences(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *Service) ensureProfileReferences(
	ctx context.Context,
	entity *edi.EDICommunicationProfile,
) error {
	if entity.EDIPartnerID.IsNotNil() {
		if _, err := s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
			ID: entity.EDIPartnerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}); err != nil {
			return err
		}
	}

	if entity.EDIConnectionID.IsNotNil() {
		_, err := s.connectionRepo.GetConnectionByID(
			ctx,
			edipartnerports.GetEDIConnectionByIDRequest{
				ID: entity.EDIConnectionID,
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
			},
		)
		return err
	}
	return nil
}

func (s *Service) resolveProfileSecrets(
	req *UpsertEDICommunicationProfileRequest,
	profileID pulid.ID,
	existing *edi.EDICommunicationProfile,
) (map[string]string, error) {
	secrets := map[string]string{}
	if existing != nil {
		maps.Copy(secrets, existing.EncryptedSecrets)
	}

	for key, value := range req.Secrets {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}

		encrypted, err := s.encryptProfileSecret(req, profileID, trimmedKey, trimmedValue)
		if err != nil {
			return nil, err
		}
		secrets[trimmedKey] = encrypted
	}
	return secrets, nil
}

func (s *Service) encryptProfileSecret(
	req *UpsertEDICommunicationProfileRequest,
	profileID pulid.ID,
	key string,
	value string,
) (string, error) {
	if s.encryption == nil {
		return "", errortypes.NewBusinessError(
			"EDI communication profile secrets cannot be saved because the encryption service is not configured",
		)
	}

	encrypted, err := s.encryption.EncryptStringWithAAD(
		value,
		ediprofilesecret.ProfileSecretAAD(req.TenantInfo, profileID, key),
	)
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to encrypt EDI communication profile secret",
		).WithInternal(err)
	}
	return encrypted, nil
}

func normalizeProfileConfig(config map[string]any) map[string]any {
	normalized := make(map[string]any, len(config))
	for key, value := range config {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if stringValue, ok := value.(string); ok {
			normalized[trimmedKey] = strings.TrimSpace(stringValue)
			continue
		}
		normalized[trimmedKey] = value
	}
	return normalized
}
