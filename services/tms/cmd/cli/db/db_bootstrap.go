package db

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/bootstrap/infrastructure"
	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/instancebootstrapservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/auditrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/instancebootstraprepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/tenantbootstraprepository"
	redisrepositories "github.com/emoss08/trenova/internal/infrastructure/redis/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/fatih/color"
	goredis "github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/term"
)

const (
	bootstrapEnvPassword  = "TRENOVA_BOOTSTRAP_ADMIN_PASSWORD"
	bootstrapStartTimeout = 30 * time.Second
	maxPasswordInputBytes = 1024
)

var (
	errNoBootstrapPassword = errors.New(
		"no administrator password: set " + bootstrapEnvPassword +
			", pipe it with --password-stdin, or run in a terminal to be prompted",
	)
	errPasswordsDiffer = errors.New("the two passwords did not match")
)

type bootstrapInput struct {
	field string
	flag  string
	env   string
	usage string
	value string
}

var bootstrapInputs = []*bootstrapInput{
	{
		field: instancebootstrap.FieldOrganizationName,
		flag:  "org-name",
		env:   "TRENOVA_BOOTSTRAP_ORG_NAME",
		usage: "name of the first organization (required)",
	},
	{
		field: instancebootstrap.FieldAdminName,
		flag:  "admin-name",
		env:   "TRENOVA_BOOTSTRAP_ADMIN_NAME",
		usage: "full name of the first administrator (required)",
	},
	{
		field: instancebootstrap.FieldAdminEmail,
		flag:  "admin-email",
		env:   "TRENOVA_BOOTSTRAP_ADMIN_EMAIL",
		usage: "email address the first administrator signs in with (required)",
	},
	{
		field: instancebootstrap.FieldTimezone,
		flag:  "timezone",
		env:   "TRENOVA_BOOTSTRAP_TIMEZONE",
		usage: "IANA time zone of the organization (default " + instancebootstrap.DefaultTimezone + ")",
	},
	{
		field: instancebootstrap.FieldState,
		flag:  "state",
		env:   "TRENOVA_BOOTSTRAP_STATE",
		usage: "two-letter US state of the organization's address (default " +
			instancebootstrap.DefaultState + ")",
	},
	{
		field: instancebootstrap.FieldCity,
		flag:  "city",
		env:   "TRENOVA_BOOTSTRAP_CITY",
		usage: "city of the organization's address",
	},
	{
		field: instancebootstrap.FieldPostalCode,
		flag:  "postal-code",
		env:   "TRENOVA_BOOTSTRAP_POSTAL_CODE",
		usage: "ZIP code of the organization's address",
	},
	{
		field: instancebootstrap.FieldAddressLine1,
		flag:  "address-line1",
		env:   "TRENOVA_BOOTSTRAP_ADDRESS_LINE1",
		usage: "street address of the organization",
	},
	{
		field: instancebootstrap.FieldSCAC,
		flag:  "scac",
		env:   "TRENOVA_BOOTSTRAP_SCAC",
		usage: "four-letter SCAC code of the organization",
	},
	{
		field: instancebootstrap.FieldDOTNumber,
		flag:  "dot-number",
		env:   "TRENOVA_BOOTSTRAP_DOT_NUMBER",
		usage: "USDOT number of the organization",
	},
}

var bootstrapPasswordStdin bool

var dbBootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Create the first organization and administrator of a new install",
	Long: `Create the first organization of an empty install and its first administrator,
who holds the Organization Administrator role and can sign in immediately.

Every value can be given as a flag or as an environment variable; a flag wins.
The administrator password is never a flag value. It is read from
TRENOVA_BOOTSTRAP_ADMIN_PASSWORD, from standard input with --password-stdin, or
from a prompt when run in a terminal, and must meet the password policy: at
least 12 characters, not a common password, not containing the email address.

The address, time zone and identifiers of the organization are starting values;
placeholders are used for any left out. Change them later in Organization
settings.

The command is safe to run on every start:
  - on a database with no users it creates the organization and administrator;
  - after it has run, the same organization name, administrator name and email
    are a no-op success, and the password is not read;
  - different values, or a database that already has users, are refused with a
    non-zero exit, and nothing is changed.

Examples:
  trenova db bootstrap --org-name "Acme Freight" --admin-name "Dana Whitfield" \
    --admin-email dana@acme.example
  printf '%s\n' "$PASSWORD" | trenova db bootstrap --password-stdin ...
  trenova db bootstrap --dry-run        # report what would happen`,
	RunE: runBootstrap,
}

func init() {
	for _, input := range bootstrapInputs {
		dbBootstrapCmd.Flags().StringVar(
			&input.value,
			input.flag,
			"",
			input.usage+" [$"+input.env+"]",
		)
	}
	dbBootstrapCmd.Flags().BoolVar(
		&bootstrapPasswordStdin,
		"password-stdin",
		false,
		"read the administrator password from the first line of standard input",
	)

	DbCmd.AddCommand(dbBootstrapCmd)
}

type passwordSource func() (string, error)

type bootstrapDeps struct {
	conn    *postgres.Connection
	auditor func(
		ctx context.Context,
		conn *postgres.Connection,
	) (services.SecurityAuditor, func(), error)
	cfg    *config.Config
	logger *zap.Logger
	out    io.Writer
}

func runBootstrap(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	inputs := resolveBootstrapInputs(cmd)

	logger, _, err := config.ProvideLogger(cfg)
	if err != nil {
		return err
	}

	manager, err := createManager()
	if err != nil {
		return fmt.Errorf("failed to create database manager: %w", err)
	}
	defer manager.Close()

	deps := bootstrapDeps{
		conn:    postgres.WrapDB(manager.GetDB()),
		auditor: redisSecurityAuditor,
		cfg:     cfg,
		logger:  logger,
		out:     cmd.OutOrStdout(),
	}

	return bootstrapInstance(ctx, deps, &inputs, passwordFrom(cmd.InOrStdin()), dryRun)
}

func resolveBootstrapInputs(cmd *cobra.Command) instancebootstrap.Inputs {
	values := make(map[string]string, len(bootstrapInputs))
	for _, input := range bootstrapInputs {
		value := input.value
		if !cmd.Flags().Changed(input.flag) {
			value = os.Getenv(input.env)
		}
		values[input.field] = value
	}

	return instancebootstrap.Inputs{
		OrganizationName: values[instancebootstrap.FieldOrganizationName],
		AdminName:        values[instancebootstrap.FieldAdminName],
		AdminEmail:       values[instancebootstrap.FieldAdminEmail],
		Timezone:         values[instancebootstrap.FieldTimezone],
		State:            values[instancebootstrap.FieldState],
		City:             values[instancebootstrap.FieldCity],
		PostalCode:       values[instancebootstrap.FieldPostalCode],
		AddressLine1:     values[instancebootstrap.FieldAddressLine1],
		SCAC:             values[instancebootstrap.FieldSCAC],
		DOTNumber:        values[instancebootstrap.FieldDOTNumber],
	}
}

func bootstrapInstance(
	ctx context.Context,
	deps bootstrapDeps,
	inputs *instancebootstrap.Inputs,
	password passwordSource,
	preview bool,
) error {
	planner := newInstanceBootstrapService(deps, nil)
	plan, err := planner.Plan(ctx, inputs)
	if err != nil {
		return describeBootstrapError(err)
	}

	switch {
	case plan.Status == services.InstanceBootstrapCompleted:
		printBootstrapCompleted(deps.out, &plan.Inputs, plan.Record.CompletedAt)
		return nil
	case preview:
		fmt.Fprintf(
			deps.out,
			"Dry run: would create organization %q and administrator %s\n",
			plan.Inputs.OrganizationName,
			plan.Inputs.AdminEmail,
		)
		return nil
	}

	secret, err := password()
	if err != nil {
		return err
	}

	auditor, closeAuditor, err := deps.auditor(ctx, deps.conn)
	if err != nil {
		return err
	}
	defer closeAuditor()

	result, err := newInstanceBootstrapService(deps, auditor).Bootstrap(
		ctx,
		&services.InstanceBootstrapRequest{Inputs: plan.Inputs, Password: secret},
	)
	if err != nil {
		return describeBootstrapError(err)
	}

	if result.Status == services.InstanceBootstrapCompleted {
		printBootstrapCompleted(deps.out, &plan.Inputs, result.CompletedAt)
		return nil
	}

	printBootstrapCreated(deps.out, deps.cfg, &plan.Inputs, result)
	return nil
}

func newInstanceBootstrapService(
	deps bootstrapDeps,
	auditor services.SecurityAuditor,
) *instancebootstrapservice.Service {
	return instancebootstrapservice.New(instancebootstrapservice.Params{
		DB: deps.conn,
		Tenants: tenantbootstraprepository.New(tenantbootstraprepository.Params{
			DB:     deps.conn,
			Logger: deps.logger,
		}),
		Repo: instancebootstraprepository.New(instancebootstraprepository.Params{
			DB:     deps.conn,
			Logger: deps.logger,
		}),
		Auditor: auditor,
		Config:  deps.cfg,
		Logger:  deps.logger,
	})
}

func redisSecurityAuditor(
	ctx context.Context,
	conn *postgres.Connection,
) (services.SecurityAuditor, func(), error) {
	logger := zap.L()

	var client *goredis.Client
	app := fx.New(
		fx.NopLogger,
		fx.Supply(cfg),
		fx.Supply(logger),
		infrastructure.RedisModule,
		fx.Populate(&client),
	)

	startCtx, cancel := context.WithTimeout(ctx, bootstrapStartTimeout)
	defer cancel()
	if err := app.Start(startCtx); err != nil {
		return nil, nil, fmt.Errorf("connect to Redis for the audit buffer: %w", err)
	}

	stop := func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), bootstrapStartTimeout)
		defer stopCancel()
		if err := app.Stop(stopCtx); err != nil {
			logger.Warn("failed to close the Redis connection", zap.Error(err))
		}
	}

	registry, err := metrics.NewRegistry(cfg, logger)
	if err != nil {
		stop()
		return nil, nil, fmt.Errorf("create metrics registry: %w", err)
	}

	audit := auditservice.New(auditservice.Params{
		AuditRepository: auditrepository.New(auditrepository.Params{DB: conn, Logger: logger}),
		AuditBufferRepository: redisrepositories.NewAuditBufferRepository(
			redisrepositories.AuditBufferRepositoryParams{Client: client, Logger: logger},
		),
		Logger:  logger,
		Config:  cfg,
		Metrics: registry,
	})

	auditor := auditservice.NewSecurityAuditor(auditservice.SecurityAuditorParams{
		AuditService: audit,
		Metrics:      registry,
		Logger:       logger,
	})

	return auditor, stop, nil
}

func passwordFrom(stdin io.Reader) passwordSource {
	return func() (string, error) {
		if bootstrapPasswordStdin {
			return readPasswordLine(stdin)
		}

		if value, ok := os.LookupEnv(bootstrapEnvPassword); ok && value != "" {
			if err := os.Unsetenv(bootstrapEnvPassword); err != nil {
				return "", fmt.Errorf("clear %s: %w", bootstrapEnvPassword, err)
			}
			return value, nil
		}

		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			return "", errNoBootstrapPassword
		}

		return promptPassword(fd)
	}
}

func readPasswordLine(stdin io.Reader) (string, error) {
	line, err := bufio.NewReaderSize(io.LimitReader(stdin, maxPasswordInputBytes), maxPasswordInputBytes).
		ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read the administrator password from standard input: %w", err)
	}

	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", errNoBootstrapPassword
	}

	return line, nil
}

func promptPassword(fd int) (string, error) {
	fmt.Fprint(os.Stderr, "Administrator password: ")
	first, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read the administrator password: %w", err)
	}

	fmt.Fprint(os.Stderr, "Repeat the password: ")
	second, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read the administrator password: %w", err)
	}

	if string(first) != string(second) {
		return "", errPasswordsDiffer
	}
	if len(first) == 0 {
		return "", errNoBootstrapPassword
	}

	return string(first), nil
}

func describeBootstrapError(err error) error {
	var multiErr *errortypes.MultiError
	if !errors.As(err, &multiErr) || len(multiErr.Errors) == 0 {
		return fmt.Errorf("bootstrap refused: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("bootstrap inputs are not valid:")
	for _, fieldErr := range multiErr.Errors {
		builder.WriteString("\n  - ")
		builder.WriteString(fieldErr.Message)
		if source := bootstrapInputSource(fieldErr.Field); source != "" {
			builder.WriteString(" (")
			builder.WriteString(source)
			builder.WriteString(")")
		}
	}

	return errors.New(builder.String())
}

func bootstrapInputSource(field string) string {
	if field == "password" {
		return bootstrapEnvPassword + " or --password-stdin"
	}
	for _, input := range bootstrapInputs {
		if input.field == field {
			return "--" + input.flag + " or " + input.env
		}
	}

	return ""
}

func printBootstrapCompleted(out io.Writer, inputs *instancebootstrap.Inputs, completedAt int64) {
	fmt.Fprintf(
		out,
		"Already bootstrapped on %s for organization %q and administrator %s; nothing to do.\n",
		time.Unix(completedAt, 0).UTC().Format(time.RFC3339),
		inputs.OrganizationName,
		inputs.AdminEmail,
	)
}

func printBootstrapCreated(
	out io.Writer,
	appCfg *config.Config,
	inputs *instancebootstrap.Inputs,
	result *services.InstanceBootstrapResult,
) {
	fmt.Fprintln(out, color.GreenString("Created organization %q", inputs.OrganizationName))
	fmt.Fprintf(out, "  Administrator: %s <%s>\n", inputs.AdminName, result.AdminEmail)
	fmt.Fprintf(out, "  Username:      %s\n", result.AdminUsername)
	fmt.Fprintf(out, "  Sign-in slug:  %s\n", result.LoginSlug)
	if appCfg != nil && appCfg.App.GetWebBaseURL() != "" {
		fmt.Fprintf(
			out,
			"Sign in at %s with that email address and password.\n",
			appCfg.App.GetWebBaseURL(),
		)
	}
	if result.SystemUserCreated {
		fmt.Fprintln(out, "  Also created the system account background work runs as.")
	}
}
