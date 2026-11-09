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

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/instancebootstraprepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/tenantbootstraprepository"
	"github.com/emoss08/trenova/internal/modules/identity/app/instancebootstrapservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

const (
	bootstrapEnvPassword   = "TRENOVA_BOOTSTRAP_ADMIN_PASSWORD"
	bootstrapEnvAdminSetup = "TRENOVA_BOOTSTRAP_ADMIN_SETUP"
	bootstrapFlagSetup     = "admin-setup"
	bootstrapStartTimeout  = 30 * time.Second
	maxPasswordInputBytes  = 1024
)

var (
	errNoBootstrapPassword = errors.New(
		"no administrator password: set " + bootstrapEnvPassword +
			", pipe it with --password-stdin, or run in a terminal to be prompted",
	)
	errPasswordsDiffer = errors.New("the two passwords did not match")
	errAdminSetup      = errors.New(
		"--" + bootstrapFlagSetup + " (" + bootstrapEnvAdminSetup + ") must be password or link",
	)
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

var (
	bootstrapPasswordStdin bool
	bootstrapAdminSetup    string
)

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

With --admin-setup link the administrator gets no usable password and no
password is read: they choose one from a one-time account setup link, the way
an invited user does. Trenova Cloud installs this way and delivers the link.

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
	dbBootstrapCmd.Flags().StringVar(
		&bootstrapAdminSetup,
		bootstrapFlagSetup,
		string(instancebootstrap.AdminSetupPassword),
		"how the administrator gets a password: password (set now) or link (a one-time setup "+
			"link sets it) [$"+bootstrapEnvAdminSetup+"]",
	)

	DbCmd.AddCommand(dbBootstrapCmd)
}

type passwordSource func() (string, error)

type bootstrapDeps struct {
	conn    *postgres.Connection
	auditor func(
		ctx context.Context,
		conn *postgres.Connection,
	) (permissionports.SecurityAuditor, func(), error)
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
	setup, err := resolveAdminSetup(cmd)
	if err != nil {
		return err
	}

	logger, _, err := config.ProvideLogger(&cfg.Logging)
	if err != nil {
		return err
	}

	maint, err := openDB(context.Background())
	if err != nil {
		return fmt.Errorf("open the database: %w", err)
	}
	defer maint.Close()

	deps := bootstrapDeps{
		conn:    postgres.WrapDB(maint),
		auditor: cliSecurityAuditor,
		cfg:     cfg,
		logger:  logger,
		out:     cmd.OutOrStdout(),
	}

	return bootstrapInstance(ctx, deps, &bootstrapRun{
		inputs:   &inputs,
		setup:    setup,
		password: passwordFrom(cmd.InOrStdin()),
		preview:  dryRun,
	})
}

func resolveAdminSetup(cmd *cobra.Command) (instancebootstrap.AdminSetup, error) {
	raw := bootstrapAdminSetup
	if !cmd.Flags().Changed(bootstrapFlagSetup) {
		raw = os.Getenv(bootstrapEnvAdminSetup)
	}
	setup, ok := instancebootstrap.ParseAdminSetup(raw)
	if !ok {
		return "", errAdminSetup
	}
	return setup, nil
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

type bootstrapRun struct {
	inputs   *instancebootstrap.Inputs
	setup    instancebootstrap.AdminSetup
	password passwordSource
	preview  bool
}

func bootstrapInstance(ctx context.Context, deps bootstrapDeps, run *bootstrapRun) error {
	planner := newInstanceBootstrapService(deps, nil)
	plan, err := planner.Plan(ctx, run.inputs)
	if err != nil {
		return describeBootstrapError(err)
	}

	switch {
	case plan.Status == instancebootstrapports.InstanceBootstrapCompleted:
		printBootstrapCompleted(deps.out, &plan.Inputs, plan.Record.CompletedAt)
		return nil
	case run.preview:
		fmt.Fprintf(
			deps.out,
			"Dry run: would create organization %q and administrator %s\n",
			plan.Inputs.OrganizationName,
			plan.Inputs.AdminEmail,
		)
		return nil
	}

	var secret string
	if run.setup != instancebootstrap.AdminSetupLink {
		if secret, err = run.password(); err != nil {
			return err
		}
	}

	auditor, closeAuditor, err := deps.auditor(ctx, deps.conn)
	if err != nil {
		return err
	}
	defer closeAuditor()

	result, err := newInstanceBootstrapService(deps, auditor).Bootstrap(
		ctx,
		&instancebootstrapports.InstanceBootstrapRequest{
			Inputs:     plan.Inputs,
			AdminSetup: run.setup,
			Password:   secret,
		},
	)
	if err != nil {
		return describeBootstrapError(err)
	}

	if result.Status == instancebootstrapports.InstanceBootstrapCompleted {
		printBootstrapCompleted(deps.out, &plan.Inputs, result.CompletedAt)
		return nil
	}

	printBootstrapCreated(deps.out, &deps.cfg.App, &plan.Inputs, result, run.setup)
	return nil
}

func newInstanceBootstrapService(
	deps bootstrapDeps,
	auditor permissionports.SecurityAuditor,
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
		Auditor:      auditor,
		SystemConfig: &deps.cfg.System,
		Logger:       deps.logger,
	})
}

func cliSecurityAuditor(
	_ context.Context,
	conn *postgres.Connection,
) (permissionports.SecurityAuditor, func(), error) {
	logger := zap.L()

	registry, err := metrics.NewRegistry(&cfg.Monitoring, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("create metrics registry: %w", err)
	}

	audit := auditservice.New(auditservice.Params{
		AuditRepository: auditrepository.New(auditrepository.Params{DB: conn, Logger: logger}),
		Logger:          logger,
		Config:          &cfg.Audit,
		Metrics:         registry,
	})

	auditor := auditservice.NewSecurityAuditor(auditservice.SecurityAuditorParams{
		AuditService: audit,
		Metrics:      registry,
		Logger:       logger,
	})

	return auditor, func() {}, nil
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
		if source := bootstrapInputSource(fieldErr.Field()); source != "" {
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
	appCfg *appconfig.Config,
	inputs *instancebootstrap.Inputs,
	result *instancebootstrapports.InstanceBootstrapResult,
	setup instancebootstrap.AdminSetup,
) {
	fmt.Fprintln(out, color.GreenString("Created organization %q", inputs.OrganizationName))
	fmt.Fprintf(out, "  Administrator: %s <%s>\n", inputs.AdminName, result.AdminEmail)
	fmt.Fprintf(out, "  Username:      %s\n", result.AdminUsername)
	fmt.Fprintf(out, "  Sign-in slug:  %s\n", result.LoginSlug)
	switch {
	case setup == instancebootstrap.AdminSetupLink:
		fmt.Fprintln(
			out,
			"The administrator sets their password from a one-time account setup link.",
		)
	case appCfg != nil && appCfg.GetWebBaseURL() != "":
		fmt.Fprintf(
			out,
			"Sign in at %s with that email address and password.\n",
			appCfg.GetWebBaseURL(),
		)
	}
	if result.SystemUserCreated {
		fmt.Fprintln(out, "  Also created the system account background work runs as.")
	}
}
