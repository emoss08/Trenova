package dbcli

import (
	"context"
	"fmt"

	"github.com/emoss08/keel/keelcli"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
)

func provisionRolesCommand(settings *Settings) keelcli.Command {
	return keelcli.Command{
		Name: "provision-roles",
		Summary: "create or update the application (database.user) and system " +
			"(database.system.user) login roles; run as database.migrator, safe to run again",
		Run: func(ctx context.Context, env *keelcli.Env, args []string) error {
			return runProvisionRoles(ctx, env, args, settings)
		},
	}
}

func runProvisionRoles(
	ctx context.Context,
	env *keelcli.Env,
	args []string,
	settings *Settings,
) error {
	if err := env.Parse(env.FlagSet("provision-roles"), args); err != nil {
		return err
	}

	cfg, err := settings.Config()
	if err != nil {
		return err
	}

	db, err := env.DB(ctx)
	if err != nil {
		return err
	}

	roles, err := postgres.ProvisionConfiguredRLSRoles(ctx, db, &cfg.Database)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(
		env.Stdout,
		"Application role %q is bound by tenant isolation\n"+
			"System role %q serves dbscope.WithSystem code paths\n",
		roles[0],
		roles[1],
	)
	return err
}
