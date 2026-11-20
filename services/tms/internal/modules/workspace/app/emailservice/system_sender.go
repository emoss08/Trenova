package emailservice

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/remedy"
	"github.com/emoss08/trenova/shared/mailtext"
	"go.uber.org/zap"
)

const (
	trenovaInstanceKey      = "email:trenova-sender:instance"
	trenovaInvitationPrefix = "email:trenova-sender:invitations:"
	trenovaOrganizationKey  = "email:trenova-sender:organization:"
)

type trenovaBudget int

const (
	budgetInstance = trenovaBudget(iota)
	budgetOrganization
	budgetInvitations
)

var (
	trenovaCredentialKeys = map[email.Provider]string{
		email.ProviderResend:   "apiKey",
		email.ProviderPostmark: "serverToken",
	}
	errNoInstanceID = errors.New(
		"the Trenova system sender cannot be used until this instance knows its own ID",
	)
	trenovaInstancePolicy = kernelports.RateLimitPolicy{
		Rate:   3000,
		Period: time.Hour,
		Burst:  300,
	}
	trenovaInvitationPolicy = kernelports.RateLimitPolicy{
		Rate:   30,
		Period: time.Hour,
		Burst:  30,
	}
	errTrenovaSenderGone = errors.New(
		"the Trenova system sender is no longer set on this instance",
	)
)

type sendRoute struct {
	profile  *email.Profile
	trenova  *systememail.Credential
	fromName string
}

func (r *sendRoute) stamp(msg *email.Message, req *emailports.SendEmailRequest) {
	msg.Type = req.Type
	msg.Kind = req.Type.Kind()
	if r.trenova != nil {
		msg.Sender = email.SenderTrenova
		msg.Provider = r.trenova.Provider
		msg.FromEmail = r.trenova.FromAddress
		msg.FromName = r.fromName
		return
	}
	msg.Sender = email.SenderOrganization
	msg.ProfileID = r.profile.ID
	msg.Provider = r.profile.Provider
	msg.FromEmail = senderEmail(req, r.profile)
	msg.FromName = r.profile.SenderName
	msg.ReplyToEmail = r.profile.ReplyToEmail
}

func (s *Service) route(
	ctx context.Context,
	req *emailports.SendEmailRequest,
) (*sendRoute, error) {
	profile, err := s.resolveProfile(ctx, req)
	if err == nil {
		return &sendRoute{profile: profile}, nil
	}
	if !errors.Is(err, emailports.ErrNoEmailProfile) || !req.ProfileID.IsNil() ||
		!req.Type.IsSystem() {
		return nil, err
	}
	trenova, sErr := s.systemSenders.Current(ctx)
	if sErr != nil {
		return nil, sErr
	}
	if trenova == nil {
		return nil, err
	}
	name, nErr := s.trenovaDisplayName(ctx, req.TenantInfo, trenova)
	if nErr != nil {
		return nil, nErr
	}
	return &sendRoute{trenova: trenova, fromName: name}, nil
}

func (s *Service) trenovaDisplayName(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	trenova *systememail.Credential,
) (string, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return "", err
	}
	return systememail.DisplayName(org.Name, trenova.FromName), nil
}

func (s *Service) admitTrenovaSend(ctx context.Context, req *emailports.SendEmailRequest) error {
	if !req.Type.IsSystem() {
		return fmt.Errorf("%w: %s email may not be sent by Trenova", ErrNonRetryableSend, req.Type)
	}
	recipients := mailtext.NormalizeEmailList(slices.Concat(req.To, req.CC, req.BCC))
	if !req.Type.IsInvitation() {
		members, err := s.directory.MemberAddresses(ctx, req.TenantInfo, recipients)
		if err != nil {
			return err
		}
		if outside := notIn(recipients, members); len(outside) > 0 {
			return remedy.Mark(errortypes.NewBusinessError(
				"Email Trenova sends for this organization only reaches its own people, and "+
					"{0} is not one of them. An administrator can add an email profile in the "+
					"email settings to email anyone else.",
				strings.Join(outside, ", "),
			), remedy.SetUpEmailProfile)
		}
	}
	return s.admitTrenovaRate(ctx, req, len(recipients))
}

func (s *Service) admitTrenovaRate(
	ctx context.Context,
	req *emailports.SendEmailRequest,
	recipients int,
) error {
	cost := max(recipients, 1)
	requests := []kernelports.RateLimitRequest{{
		Key:    trenovaInstanceKey,
		Policy: trenovaInstancePolicy,
		Cost:   min(cost, trenovaInstancePolicy.Burst),
	}}
	budgets := []trenovaBudget{budgetInstance}
	if s.organizationPerHour > 0 {
		policy := kernelports.RateLimitPolicy{
			Rate:   s.organizationPerHour,
			Period: time.Hour,
			Burst:  min(s.organizationPerHour, trenovaInstancePolicy.Burst),
		}
		requests = append(requests, kernelports.RateLimitRequest{
			Key:    trenovaOrganizationKey + req.TenantInfo.OrgID.String(),
			Policy: policy,
			Cost:   min(cost, policy.Burst),
		})
		budgets = append(budgets, budgetOrganization)
	}
	if req.Type.IsInvitation() {
		requests = append(requests, kernelports.RateLimitRequest{
			Key:    trenovaInvitationPrefix + req.TenantInfo.OrgID.String(),
			Policy: trenovaInvitationPolicy,
			Cost:   min(cost, trenovaInvitationPolicy.Burst),
		})
		budgets = append(budgets, budgetInvitations)
	}
	decisions, err := s.rateLimits.Check(ctx, requests)
	if err != nil {
		return fmt.Errorf("check the Trenova sender's rate limit: %w", err)
	}
	for i, decision := range decisions {
		if decision.Allowed {
			continue
		}
		return s.overBudget(budgets[i], decision.RetryAfter)
	}
	return nil
}

func (s *Service) overBudget(budget trenovaBudget, retryAfter time.Duration) error {
	minutes := retryMinutes(retryAfter)
	if budget == budgetInvitations {
		return errortypes.NewRateLimitError(
			"to",
			"This organization has sent as many invitations as it may in an hour. "+
				"Try again in {0} minutes, or add an email profile in the email settings.",
			minutes,
		).WithRetryAfter(retryAfter)
	}
	if budget == budgetOrganization {
		s.tenancy.RecordLimitRefusal("system_email", platformmetrics.LimitScopeOrganization)
		return errortypes.NewRateLimitError(
			"to",
			"This organization has sent as much email through Trenova as it may in an hour. "+
				"Try again in {0} minutes, or add an email profile in the email settings.",
			minutes,
		).WithRetryAfter(retryAfter)
	}
	s.tenancy.RecordLimitRefusal("system_email", platformmetrics.LimitScopeInstance)
	return errortypes.NewRateLimitError(
		"to",
		"Email Trenova sends for this instance has reached its hourly limit. "+
			"Try again in {0} minutes, or add an email profile in the email settings.",
		minutes,
	).WithRetryAfter(retryAfter)
}

func retryMinutes(d time.Duration) string {
	return strconv.Itoa(int(math.Max(1, math.Ceil(d.Minutes()))))
}

func notIn(addresses, members []string) []string {
	known := make(map[string]struct{}, len(members))
	for _, member := range members {
		known[mailtext.NormalizeEmailAddress(member)] = struct{}{}
	}
	outside := make([]string, 0)
	for _, address := range addresses {
		if _, ok := known[address]; !ok {
			outside = append(outside, address)
		}
	}
	return outside
}

func (s *Service) trenovaProvider(
	ctx context.Context,
	msg *email.Message,
) (*providerCall, error) {
	if msg.Kind != email.KindSystem || !msg.Type.IsSystem() {
		return nil, fmt.Errorf(
			"%w: %s email may not be sent by Trenova",
			ErrNonRetryableSend,
			msg.Type,
		)
	}
	instanceID, err := s.instanceTags.InstanceTag(ctx, pagination.TenantInfo{
		OrgID: msg.OrganizationID,
		BuID:  msg.BusinessUnitID,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"%w: name the instance the message is sent for: %w",
			ErrRetryableSend,
			err,
		)
	}
	if instanceID == "" {
		return nil, fmt.Errorf("%w: %w", ErrNonRetryableSend, errNoInstanceID)
	}
	trenova, err := s.systemSenders.Current(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: read the Trenova sender: %w", ErrRetryableSend, err)
	}
	if trenova == nil {
		return nil, fmt.Errorf("%w: %w", ErrNonRetryableSend, errTrenovaSenderGone)
	}
	sender, ok := s.providerSenders[trenova.Provider]
	credentialKey, known := trenovaCredentialKeys[trenova.Provider]
	if !ok || !known {
		return nil, fmt.Errorf(
			"%w: no sender registered for email provider %s",
			ErrNonRetryableSend,
			trenova.Provider,
		)
	}
	msg.Provider = trenova.Provider
	msg.FromEmail = trenova.FromAddress
	return &providerCall{
		sender:   sender,
		config:   map[string]string{credentialKey: trenova.APIKey},
		metadata: map[string]string{systememail.InstanceTag: instanceID},
	}, nil
}

func (s *Service) trenovaSendFailure(
	msg *email.Message,
	config map[string]string,
	err error,
) error {
	reason := err.Error()
	for _, secret := range config {
		if secret != "" {
			reason = strings.ReplaceAll(reason, secret, "[redacted]")
		}
	}
	s.l.Warn(
		"the Trenova sender could not deliver an email",
		zap.String("messageId", msg.ID.String()),
		zap.String("provider", string(msg.Provider)),
		zap.String("reason", reason),
	)
	class := ErrNonRetryableSend
	if errors.Is(err, ErrRetryableSend) {
		class = ErrRetryableSend
	}
	return fmt.Errorf("%w: Trenova's system sender could not deliver this email", class)
}

func (s *Service) Routing(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignments []*email.ProfileAssignment,
) (*emailports.EmailRouting, error) {
	inUse, from, err := s.SystemEmailInUse(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if assignments == nil {
		assignments = []*email.ProfileAssignment{}
	}
	return &emailports.EmailRouting{
		Assignments:            assignments,
		SystemEmailInUse:       inUse,
		SystemEmailFromAddress: from,
	}, nil
}

func (s *Service) SystemEmailInUse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, string, error) {
	trenova, err := s.systemSenders.Current(ctx)
	if err != nil || trenova == nil {
		return false, "", err
	}
	_, err = s.repo.GetAssignedProfile(ctx, tenantInfo, email.PurposeGeneral)
	switch {
	case err == nil:
		return false, "", nil
	case errortypes.IsNotFoundError(err):
		return true, trenova.FromAddress, nil
	default:
		return false, "", err
	}
}

func (s *Service) TenantOfSystemMessage(
	ctx context.Context,
	provider email.Provider,
	providerMessageID string,
) (pagination.TenantInfo, bool, error) {
	if strings.TrimSpace(providerMessageID) == "" {
		return pagination.TenantInfo{}, false, nil
	}
	tenantInfo, err := s.systemMessages.TenantOfSystemMessage(ctx, provider, providerMessageID)
	switch {
	case err == nil:
		return tenantInfo, true, nil
	case errortypes.IsNotFoundError(err):
		return pagination.TenantInfo{}, false, nil
	default:
		return pagination.TenantInfo{}, false, err
	}
}
