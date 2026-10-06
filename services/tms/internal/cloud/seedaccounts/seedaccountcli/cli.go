package seedaccountcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountrepository"
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountservice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/auditrepository"
	redisrepositories "github.com/emoss08/trenova/internal/infrastructure/redis/repositories"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	startupTimeout  = 30 * time.Second
	shutdownTimeout = 15 * time.Second
	outputText      = "text"
	outputJSON      = "json"
)

var (
	errApplyAndDryRun = errors.New("--apply and --dry-run cannot be combined")
	errUnknownOutput  = errors.New("--output must be text or json")
	errStepsFailed    = errors.New("some steps failed after the database changes were committed; " +
		"see Failed steps: running the command again ends the accounts' sessions again, cached " +
		"permissions expire with their TTL, and an audit entry that was logged instead must be filed by hand")
)

var (
	applyFlag  bool
	dryRunFlag bool
	outputFlag string
)

var RetireCmd = &cobra.Command{
	Use:   "retire-seed-accounts",
	Short: "Retire the legacy admin, admin-logistics and admin-transport seed accounts",
	Long: `Retire the accounts the AdminAccount development seed created with the published
password admin123! before seeding was limited to development: admin
(admin@trenova.app), admin-logistics (admin.logistics@trenova.app) and
admin-transport (admin.transport@trenova.app).

An account is retired only when its username and email address both match one of
those pairs exactly and the seed's own record proves it came from the seed: a
seed_created_entities row naming the AdminAccount seed, or a password hash that
still verifies against admin123!. A matching account without that proof is
reported and left alone.

Retiring an account removes its organization memberships, role assignments, MFA
factors and open password reset links, revokes the API keys it created, ends its
sessions and clears its cached permissions, and then deletes it if no row
anywhere references it, or otherwise keeps it disabled, locked and with an
unusable random password. The Trenova Logistics (TRNV) and Trenova
Transportation (TTNV) organizations the seed created are deleted only when they
hold nothing but seed defaults; anything else is reported for manual review.

Every change is written to the security audit trail as a critical entry. The
command is safe to run again: retired accounts are reported and left as they are.

Without --apply the command only reads, inside a read-only transaction, and
prints what --apply would do.

Examples:
  trenova cloud retire-seed-accounts
  trenova cloud retire-seed-accounts --output json
  trenova cloud retire-seed-accounts --apply`,
	Args: cobra.NoArgs,
	RunE: runRetire,
}

func init() {
	RetireCmd.Flags().BoolVar(&applyFlag, "apply", false, "perform the retirement")
	RetireCmd.Flags().BoolVar(
		&dryRunFlag,
		"dry-run",
		true,
		"print what --apply would do without changing anything (the default)",
	)
	RetireCmd.Flags().StringVar(&outputFlag, "output", outputText, "output format: text or json")
}

func runRetire(cmd *cobra.Command, _ []string) error {
	apply, err := resolveMode(applyFlag, cmd.Flags().Changed("dry-run") && dryRunFlag)
	if err != nil {
		return err
	}
	if outputFlag != outputText && outputFlag != outputJSON {
		return errUnknownOutput
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	var service *seedaccountservice.Service
	report, err := withService(
		ctx,
		apply,
		&service,
		func(ctx context.Context) (*seedaccountservice.Report, error) {
			return service.Run(ctx, seedaccountservice.RunRequest{Apply: apply})
		},
	)
	if err != nil {
		return err
	}

	if err = render(cmd.OutOrStdout(), report, outputFlag); err != nil {
		return err
	}
	if len(report.Errors) > 0 {
		return errStepsFailed
	}

	return nil
}

func resolveMode(apply, explicitDryRun bool) (bool, error) {
	if apply && explicitDryRun {
		return false, errApplyAndDryRun
	}

	return apply, nil
}

func withService(
	parent context.Context,
	apply bool,
	target **seedaccountservice.Service,
	run func(ctx context.Context) (*seedaccountservice.Report, error),
) (*seedaccountservice.Report, error) {
	app := fx.New(
		commandOptions(apply),
		fx.Populate(target),
		fx.StartTimeout(startupTimeout),
		fx.StopTimeout(shutdownTimeout),
	)
	if err := app.Err(); err != nil {
		return nil, fmt.Errorf("prepare retire-seed-accounts: %w", err)
	}

	startCtx, cancelStart := context.WithTimeout(parent, startupTimeout)
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("start retire-seed-accounts: %w", err)
	}

	report, runErr := run(parent)

	stopCtx, cancelStop := context.WithTimeout(context.WithoutCancel(parent), shutdownTimeout)
	defer cancelStop()

	return report, errors.Join(runErr, app.Stop(stopCtx))
}

func commandOptions(apply bool) fx.Option {
	base := fx.Options(
		fx.NopLogger,
		config.SectionsOption(cloudconfig.Section()),
		config.Module,
		infrastructure.ObservabilityModule,
		infrastructure.DatabaseModule,
		fx.Provide(
			seedaccountrepository.New,
			seedaccountservice.New,
		),
	)
	if !apply {
		return base
	}

	return fx.Options(
		base,
		infrastructure.RedisModule,
		fx.Provide(
			redisrepositories.NewSessionRepository,
			redisrepositories.NewPermissionCacheRepository,
			redisrepositories.NewAuditBufferRepository,
			auditrepository.New,
			newSecurityAuditor,
		),
	)
}

type securityAuditorParams struct {
	fx.In

	LC          fx.Lifecycle
	Audit       repositories.AuditRepository
	AuditBuffer repositories.AuditBufferRepository
	Config      *config.Config
	Logger      *zap.Logger
	Metrics     *metrics.Registry
}

func newSecurityAuditor(p securityAuditorParams) services.SecurityAuditor {
	audit := auditservice.New(auditservice.Params{
		LC:                    p.LC,
		AuditRepository:       p.Audit,
		AuditBufferRepository: p.AuditBuffer,
		Logger:                p.Logger,
		Config:                p.Config,
		Metrics:               p.Metrics,
	})

	return auditservice.NewSecurityAuditor(auditservice.SecurityAuditorParams{
		AuditService: audit,
		Metrics:      p.Metrics,
		Logger:       p.Logger,
	})
}

func render(w io.Writer, report *seedaccountservice.Report, format string) error {
	if format == outputJSON {
		encoded, err := sonic.ConfigStd.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("encode the report: %w", err)
		}
		_, err = fmt.Fprintln(w, string(encoded))
		return err
	}

	renderText(w, report)
	return nil
}

func renderText(w io.Writer, report *seedaccountservice.Report) {
	if report.Applied {
		fmt.Fprintln(w, "Applied. Changes:")
	} else {
		fmt.Fprintln(w, "Dry run. Nothing was changed; run again with --apply to perform:")
	}

	if report.SystemUser == nil {
		fmt.Fprintln(
			w,
			"\nNo instance system user was found; audit entries for deleted organizations "+
				"have nowhere to be recorded and are logged instead.",
		)
	}

	fmt.Fprintln(w, "\nAccounts")
	if len(report.Users) == 0 {
		fmt.Fprintln(w, "  none of the legacy seeded accounts exist")
	}
	for _, user := range report.Users {
		renderUser(w, user, report.Applied)
	}

	fmt.Fprintln(w, "\nOrganizations")
	if len(report.Organizations) == 0 {
		fmt.Fprintln(w, "  none of the seed's demo organizations exist")
	}
	for _, org := range report.Organizations {
		renderOrganization(w, org, report.Applied)
	}

	if report.Applied {
		fmt.Fprintf(w, "\nSecurity audit entries recorded: %d\n", report.AuditEntries)
	}
	if len(report.Errors) > 0 {
		fmt.Fprintln(w, "\nFailed steps")
		for _, failure := range report.Errors {
			fmt.Fprintln(w, "  - "+failure)
		}
	}
}

func renderUser(w io.Writer, user *seedaccountservice.UserReport, applied bool) {
	fp := user.Footprint
	fmt.Fprintf(
		w,
		"  %s <%s> %s (%s, locked %t)\n",
		fp.Username,
		fp.EmailAddress,
		fp.ID,
		fp.Status,
		fp.IsLocked,
	)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "    provenance\t%s\n", provenance(user.Provenance))

	switch user.Action {
	case seedaccountservice.UserActionUnproven:
		fmt.Fprintf(
			tw,
			"    action\tskipped: nothing proves the seed created it; review it by hand\n",
		)
	case seedaccountservice.UserActionAlreadyRetired:
		fmt.Fprintf(tw, "    action\talready retired; sessions are ended again\n")
	case seedaccountservice.UserActionRetire:
		memberships, assignments, keys, factors, tokens := counts(user, applied)
		fmt.Fprintf(tw, "    memberships removed\t%d\n", memberships)
		fmt.Fprintf(tw, "    role assignments removed\t%d\n", assignments)
		fmt.Fprintf(tw, "    API keys revoked\t%d\n", keys)
		fmt.Fprintf(tw, "    MFA factors removed\t%d\n", factors)
		fmt.Fprintf(tw, "    reset links invalidated\t%d\n", tokens)
		if user.Delete {
			fmt.Fprintf(tw, "    account\tdeleted (nothing references it)\n")
		} else {
			fmt.Fprintf(
				tw,
				"    account\tdisabled, locked, unusable password; kept because of %s\n",
				seedaccountservice.DescribeReferences(fp.References),
			)
		}
		if user.Removed != nil && user.Removed.RetainedReason != "" {
			fmt.Fprintf(tw, "    delete refused\t%s\n", user.Removed.RetainedReason)
		}
	}
	if applied && user.Action != seedaccountservice.UserActionUnproven {
		fmt.Fprintf(tw, "    sessions ended\t%t\n", user.SessionsRevoked)
	}
	_ = tw.Flush()
}

func counts(
	user *seedaccountservice.UserReport,
	applied bool,
) (memberships, assignments, keys, factors int, tokens int64) {
	if applied && user.Stripped != nil {
		s := user.Stripped
		return len(
				s.Memberships,
			), len(
				s.RoleAssignments,
			), len(
				s.APIKeys,
			), len(
				s.MFAAuthenticators,
			), s.ResetTokens
	}

	fp := user.Footprint
	return len(fp.Memberships), len(fp.RoleAssignments), len(fp.ActiveAPIKeys),
		len(fp.MFAAuthenticators), fp.OpenResetTokens
}

func provenance(values []seedaccountservice.Provenance) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		switch value {
		case seedaccountservice.ProvenanceSeedTracking:
			parts = append(parts, "seed_created_entities row for "+seedaccountport.SeedName)
		case seedaccountservice.ProvenanceSeededPassword:
			parts = append(parts, "password is still the seeded one")
		}
	}

	return strings.Join(parts, "; ")
}

func renderOrganization(w io.Writer, org *seedaccountservice.OrganizationReport, applied bool) {
	fp := org.Footprint
	fmt.Fprintf(w, "  %s (%s) %s\n", fp.Name, fp.ScacCode, fp.ID)

	switch {
	case org.Removed:
		fmt.Fprintln(w, "    deleted")
	case org.Action == seedaccountservice.OrganizationActionRemove && !applied:
		fmt.Fprintln(w, "    would be deleted: it holds nothing but seed defaults")
	default:
		fmt.Fprintln(w, "    kept for manual review:")
		for _, reason := range org.Reasons {
			fmt.Fprintln(w, "      - "+reason)
		}
	}
}
