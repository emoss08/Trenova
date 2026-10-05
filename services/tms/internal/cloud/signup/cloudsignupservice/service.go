package cloudsignupservice

import (
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/signup/signupport"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	statusPending     = "pending"
	maxSendAttempts   = 5
	defaultTimezone   = "America/New_York"
	defaultLocale     = "en"
	placeholderState  = "NY"
	signupAuthAAL     = 1
	maxUsernameLength = 20
)

type Params struct {
	fx.In

	DB            ports.DBConnection
	Signups       signupport.CloudSignupRepository
	Subscriptions repositories.SubscriptionRepository
	Onboarding    repositories.OnboardingRepository
	Bootstrap     repositories.TenantBootstrapRepository
	Users         repositories.UserRepository
	Plans         services.PlanService
	Auth          services.AuthService
	AuthEvents    services.AuthEventRecorder
	Security      services.SecurityAuditor `optional:"true"`
	Turnstile     services.TurnstileVerifier
	Email         services.PlatformEmailService
	Config        *config.Config
	Logger        *zap.Logger
}

type Service struct {
	db            ports.DBConnection
	signups       signupport.CloudSignupRepository
	subscriptions repositories.SubscriptionRepository
	onboarding    repositories.OnboardingRepository
	bootstrap     repositories.TenantBootstrapRepository
	users         repositories.UserRepository
	plans         services.PlanService
	auth          services.AuthService
	authEvents    services.AuthEventRecorder
	security      services.SecurityAuditor
	turnstile     services.TurnstileVerifier
	email         services.PlatformEmailService
	platform      *config.PlatformConfig
	cloud         *cloudconfig.PlatformCloudConfig
	l             *zap.Logger
}

//nolint:gocritic // fx parameter objects are passed by value
func New(p Params) signupport.CloudSignupService {
	return &Service{
		db:            p.DB,
		signups:       p.Signups,
		subscriptions: p.Subscriptions,
		onboarding:    p.Onboarding,
		bootstrap:     p.Bootstrap,
		users:         p.Users,
		plans:         p.Plans,
		auth:          p.Auth,
		authEvents:    p.AuthEvents,
		security:      p.Security,
		turnstile:     p.Turnstile,
		email:         p.Email,
		platform:      &p.Config.Platform,
		cloud:         &cloudconfig.From(p.Config).Cloud,
		l:             p.Logger.Named("service.cloud-signup"),
	}
}

func (s *Service) Enabled() bool {
	return s.platform.IsCloud() && s.cloud.Signup.Enabled
}

func accepted() *signupport.CloudSignupAccepted {
	return &signupport.CloudSignupAccepted{Status: statusPending}
}
