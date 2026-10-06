package supportcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessrepository"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessservice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/auditrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/userrepository"
	redisrepositories "github.com/emoss08/trenova/internal/infrastructure/redis/repositories"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	startupTimeout  = 30 * time.Second
	shutdownTimeout = 15 * time.Second
	operatorEnv     = "TRENOVA_OPERATOR"
)

var (
	staffRole       string
	staffOperator   string
	includeInactive bool
)

var CloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Trenova Cloud platform operations",
	Long: `Trenova Cloud platform operations run by Trenova operators.

Examples:
  trenova cloud staff add jordan@trenova.app --role support --by "Jordan Lee"
  trenova cloud staff remove jordan@trenova.app --by "Jordan Lee"
  trenova cloud staff list`,
}

var staffCmd = &cobra.Command{
	Use:   "staff",
	Short: "Manage who is Trenova platform staff and may open support sessions",
}

var staffAddCmd = &cobra.Command{
	Use:   "add <email>",
	Short: "Mark an existing user of the staff organization as platform staff",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		role := supportaccess.StaffRole(strings.ToLower(strings.TrimSpace(staffRole)))
		if !role.IsValid() {
			return fmt.Errorf("--role must be one of support or engineer, got %q", staffRole)
		}
		operator, err := resolveOperator()
		if err != nil {
			return err
		}

		return withStaffManager(
			cmd.Context(),
			func(ctx context.Context, m *supportaccessservice.StaffManager) error {
				member, addErr := m.AddStaff(ctx, &supportaccessservice.AddStaffRequest{
					EmailAddress: args[0],
					Role:         role,
					AddedBy:      operator,
				})
				if addErr != nil {
					return addErr
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s (%s) is platform staff with the %s role\n",
					member.UserName, member.UserEmail, member.Role)
				return nil
			},
		)
	},
}

var staffRemoveCmd = &cobra.Command{
	Use:   "remove <email>",
	Short: "Remove a user from platform staff and end their open support sessions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		operator, err := resolveOperator()
		if err != nil {
			return err
		}

		return withStaffManager(
			cmd.Context(),
			func(ctx context.Context, m *supportaccessservice.StaffManager) error {
				result, removeErr := m.RemoveStaff(ctx, &supportaccessservice.RemoveStaffRequest{
					EmailAddress: args[0],
					RemovedBy:    operator,
				})
				if removeErr != nil {
					return removeErr
				}
				if !result.Removed {
					fmt.Fprintf(cmd.OutOrStdout(), "%s was not active platform staff\n", args[0])
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Removed %s from platform staff\n", args[0])
				}
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"Ended %d open support session(s)\n",
					result.EndedSessions,
				)
				return nil
			},
		)
	},
}

var staffListCmd = &cobra.Command{
	Use:   "list",
	Short: "List platform staff",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return withStaffManager(
			cmd.Context(),
			func(ctx context.Context, m *supportaccessservice.StaffManager) error {
				members, err := m.ListStaff(ctx, includeInactive)
				if err != nil {
					return err
				}
				if len(members) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No platform staff yet.")
					return nil
				}

				w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tEMAIL\tROLE\tACTIVE\tADDED BY\tADDED")
				for _, member := range members {
					fmt.Fprintf(
						w,
						"%s\t%s\t%s\t%t\t%s\t%s\n",
						member.UserName,
						member.UserEmail,
						member.Role,
						member.Active,
						member.AddedBy,
						time.Unix(member.CreatedAt, 0).UTC().Format(time.RFC3339),
					)
				}
				return w.Flush()
			},
		)
	},
}

func init() {
	staffAddCmd.Flags().StringVar(&staffRole, "role", string(supportaccess.StaffRoleSupport),
		"Staff role: support or engineer")
	for _, command := range []*cobra.Command{staffAddCmd, staffRemoveCmd} {
		command.Flags().StringVar(&staffOperator, "by", "",
			"Who is making the change, recorded in the audit trail (defaults to "+operatorEnv+
				" or the operating system user)")
	}
	staffListCmd.Flags().BoolVar(&includeInactive, "all", false, "Include removed staff")

	staffCmd.AddCommand(staffAddCmd, staffRemoveCmd, staffListCmd)
	CloudCmd.AddCommand(staffCmd)
}

func resolveOperator() (string, error) {
	if operator := strings.TrimSpace(staffOperator); operator != "" {
		return operator, nil
	}
	if operator := strings.TrimSpace(os.Getenv(operatorEnv)); operator != "" {
		return operator, nil
	}
	if current, err := user.Current(); err == nil && strings.TrimSpace(current.Username) != "" {
		return current.Username, nil
	}

	return "", errors.New("say who is making the change with --by or " + operatorEnv)
}

type auditorParams struct {
	fx.In

	Repository repositories.AuditRepository
	Buffer     repositories.AuditBufferRepository
	Config     *config.Config
	Metrics    *metrics.Registry
	Logger     *zap.Logger
}

func newAuditor(p auditorParams) services.SecurityAuditor {
	audit := auditservice.New(auditservice.Params{
		AuditRepository:       p.Repository,
		AuditBufferRepository: p.Buffer,
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

func CommandOptions() fx.Option {
	return fx.Options(
		fx.NopLogger,
		config.SectionsOption(cloudconfig.Section()),
		config.Module,
		infrastructure.ObservabilityModule,
		infrastructure.DatabaseModule,
		infrastructure.RedisModule,
		fx.Provide(
			userrepository.New,
			auditrepository.New,
			redisrepositories.NewAuditBufferRepository,
			newAuditor,
			supportaccessrepository.New,
			supportaccessservice.NewStaffManager,
		),
	)
}

func withStaffManager(
	parent context.Context,
	run func(ctx context.Context, manager *supportaccessservice.StaffManager) error,
) error {
	if parent == nil {
		parent = context.Background()
	}

	var manager *supportaccessservice.StaffManager
	app := fx.New(
		CommandOptions(),
		fx.Populate(&manager),
		fx.StartTimeout(startupTimeout),
		fx.StopTimeout(shutdownTimeout),
	)
	if err := app.Err(); err != nil {
		return fmt.Errorf("prepare staff command: %w", err)
	}

	startCtx, cancelStart := context.WithTimeout(parent, startupTimeout)
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		return fmt.Errorf("start staff command: %w", err)
	}

	runErr := run(parent, manager)

	stopCtx, cancelStop := context.WithTimeout(context.WithoutCancel(parent), shutdownTimeout)
	defer cancelStop()

	return errors.Join(runErr, app.Stop(stopCtx))
}
