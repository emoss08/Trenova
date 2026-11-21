package archiveimport

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/adapters/objectstore/archivebucket"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/cloudcommandrepository"
	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantcopyrepository"
	"github.com/emoss08/trenova/internal/cloud/app/archiveengine"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/minio"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	envSecret = "TRENOVA_IMPORT_SECRET_ACCESS_KEY" // #nosec G101 -- the name of an environment variable, not a credential.
	envKeys   = "TRENOVA_IMPORT_ARCHIVE_KEYS"

	maxReason = 500

	ExitFailed  = 1
	ExitRefused = 2

	StateImporting = "importing"
	StateDone      = "done"
	StateRefused   = "refused"
)

var errBootstrapped = errors.New(
	"the instance was already bootstrapped or holds an organization; an archive is imported only into an empty install",
)

type Finalizer func(
	ctx context.Context,
	conn *postgres.Connection,
	req *instancebootstrapports.FinalizeInstanceBootstrapRequest,
) error

type Deps struct {
	Config    *config.Config
	DB        *keel.DB
	Secrets   kernelports.SecretCipher
	Blobs     kernelports.BlobCipher
	Envelopes *kernelports.EnvelopeSources
	Finalize  Finalizer
	Logger    *zap.Logger
	Close     func()
}

type Status struct {
	ID     string    `json:"id"`
	State  string    `json:"state"`
	At     time.Time `json:"at"`
	Reason string    `json:"reason,omitempty"`
}

type refusal struct {
	reason string
}

func (e *refusal) Error() string {
	return e.reason
}

type flags struct {
	id         string
	instanceID string
	endpoint   string
	region     string
	bucket     string
	accessKey  string
	expect     string
	owner      string
	statusFile string
}

func Command(build func(ctx context.Context) (*Deps, error)) *cobra.Command {
	f := new(flags)
	cmd := &cobra.Command{
		Use:   "import-archive",
		Short: "Make a new dedicated instance from a tenant archive instead of bootstrapping it",
		Long: `Import a company's tenant archive, taken on a shared server, into this empty
dedicated install in place of 'trenova db bootstrap' (Move to Managed). The
organization, its business unit, its users with their passwords and second
factors, every row and every document come from the archive; the archive's
owner becomes the instance's first administrator and no invitation is sent.

The archive is read from the instance's own backups bucket under imports/<id>/
(the operator's InstanceSpec.import). Its manifest must have the SHA-256 given
with --expect-sha256, every part must match the manifest, and the archive's
schema version must be this release's.

Secrets are never flags: the bucket's secret key comes from
TRENOVA_IMPORT_SECRET_ACCESS_KEY and the region's archive keys from
TRENOVA_IMPORT_ARCHIVE_KEYS, a JSON array of {"id", "key" (base64 of 32 bytes),
"current"}. Both are cleared from the environment once read.

It prints one JSON line on standard output, and writes the same to
--status-file when given, shaped as the operator's ImportStatus:
{"id", "state": "importing" | "done" | "refused", "at", "reason"}.

Exit codes:
  0  the archive is imported, or was already imported under this id (a no-op)
  1  the import could not complete (storage or database unavailable); nothing
     is left behind and running it again starts clean. The status line then
     says refused with why, for the operator to report if it gives up
  2  refused: the archive does not match, or this install cannot take it; the
     install is left empty and is never bootstrapped`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd, f, build)
		},
	}
	fl := cmd.Flags()
	fl.StringVar(&f.id, "id", "", "the import's id, InstanceSpec.import.id (required)")
	fl.StringVar(&f.instanceID, "instance-id", "",
		"the tenant's instance id the archive names (default managed.instanceId)")
	fl.StringVar(
		&f.endpoint,
		"endpoint",
		"",
		"the S3 endpoint of the instance's backups bucket (required)",
	)
	fl.StringVar(&f.region, "region", "", "the region requests are signed for")
	fl.StringVar(&f.bucket, "bucket", "", "the instance's backups bucket (required)")
	fl.StringVar(&f.accessKey, "access-key-id", "", "the bucket's access key id (required)")
	fl.StringVar(
		&f.expect,
		"expect-sha256",
		"",
		"the SHA-256 of the archive's manifest.json (required)",
	)
	fl.StringVar(&f.owner, "owner-email", "",
		"the first administrator's email; default the archive's earliest user")
	fl.StringVar(&f.statusFile, "status-file", "", "write the ImportStatus JSON here as well")
	return cmd
}

func run(cmd *cobra.Command, f *flags, build func(ctx context.Context) (*Deps, error)) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	out := cmd.OutOrStdout()
	report := func(state, reason string) {
		line, _ := sonic.ConfigStd.Marshal(
			Status{ID: f.id, State: state, At: time.Now().UTC(), Reason: clipReason(reason)},
		)
		fmt.Fprintln(out, string(line))
		if f.statusFile != "" {
			_ = os.WriteFile(f.statusFile, append(line, '\n'), 0o600)
		}
	}
	err := importArchive(ctx, f, build, report)
	var refused *refusal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &refused):
		report(StateRefused, refused.reason)
		os.Exit(ExitRefused)
	default:
		fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
		report(StateRefused, err.Error())
		os.Exit(ExitFailed)
	}
	return err
}

func importArchive(
	ctx context.Context,
	f *flags,
	build func(ctx context.Context) (*Deps, error),
	report func(state, reason string),
) error {
	access, err := f.access()
	if err != nil {
		return &refusal{reason: err.Error()}
	}
	deps, err := build(ctx)
	if err != nil {
		return err
	}
	defer deps.Close()
	instanceID := f.instanceID
	if instanceID == "" {
		instanceID = deps.Config.Managed.InstanceID
	}
	if !tenantarchive.ValidCommandID(f.id) || !tenantarchive.ValidCommandID(instanceID) ||
		!tenantarchive.ValidSHA256(f.expect) {
		return &refusal{
			reason: "--id, --instance-id (or managed.instanceId) and --expect-sha256 are required",
		}
	}
	conn := postgres.WrapDB(deps.DB)
	commands := cloudcommandrepository.New(cloudcommandrepository.Params{
		DB: conn, Clock: func() int64 { return time.Now().Unix() },
	})
	held, err := commands.Get(ctx, f.id)
	switch {
	case err == nil && held.State == tenantcommand.StateDone:
		report(StateDone, "")
		return nil
	case err != nil && !errors.Is(err, cloudtenantports.ErrCommandNotFound):
		return err
	}
	empty, err := emptyInstall(ctx, deps.DB)
	if err != nil {
		return err
	}
	if !empty {
		return &refusal{reason: errBootstrapped.Error()}
	}
	report(StateImporting, "")

	engine, err := newEngine(conn, deps)
	if err != nil {
		return err
	}
	result, err := engine.Import(ctx, &archiveengine.ImportRequest{
		InstanceID:   instanceID,
		ExpectSHA256: f.expect,
		Access:       access,
		OwnerEmail:   f.owner,
		Root:         tenantarchive.ImportPrefix(f.id),
	})
	if refused, ok := tenantarchive.RefusalOf(err); ok {
		return &refusal{reason: refused.Reason}
	}
	if err != nil {
		return err
	}
	if err = finalize(ctx, conn, deps, result); err != nil {
		return err
	}
	now := time.Now().Unix()
	if _, _, err = commands.Create(ctx, &tenantcommand.Command{
		CommandID:   f.id,
		InstanceID:  instanceID,
		Kind:        tenantcommand.KindImport,
		State:       tenantcommand.StateDone,
		Location:    access.Location.String(),
		Prefix:      access.Location.Prefix,
		SHA256:      result.ManifestSHA256,
		Progress:    tenantcommand.Progress{Phase: tenantcommand.PhaseFinished},
		RequestedAt: now,
		FinishedAt:  now,
		UpdatedAt:   now,
	}); err != nil {
		return err
	}
	report(StateDone, "")
	return nil
}

func clipReason(reason string) string {
	runes := []rune(reason)
	if len(runes) <= maxReason {
		return reason
	}
	return string(runes[:maxReason-1]) + "…"
}

func (f *flags) access() (*tenantarchive.Access, error) {
	secret := os.Getenv(envSecret)
	rawKeys := os.Getenv(envKeys)
	_ = os.Unsetenv(envSecret)
	_ = os.Unsetenv(envKeys)
	type archiveKey struct {
		ID      string `json:"id"`
		Key     string `json:"key"`
		Current bool   `json:"current"`
	}
	var keys []archiveKey
	if err := sonic.ConfigStd.UnmarshalFromString(rawKeys, &keys); err != nil {
		return nil, errors.New(envKeys + " must be a JSON array of {id, key, current}")
	}
	access := &tenantarchive.Access{
		Location: tenantarchive.Location{
			Endpoint: f.endpoint,
			Region:   f.region,
			Bucket:   f.bucket,
			Prefix:   tenantarchive.ImportPrefix(f.id),
		},
		Credentials: tenantarchive.Credentials{AccessKey: f.accessKey, SecretKey: secret},
	}
	for _, k := range keys {
		material, err := base64.StdEncoding.DecodeString(k.Key)
		if err != nil || len(material) != tenantarchive.KeySize {
			return nil, errors.New("each archive key is standard base64 of 32 bytes")
		}
		access.Keys = append(
			access.Keys,
			tenantarchive.Key{ID: k.ID, Material: material, Current: k.Current},
		)
	}
	if err := access.CheckReadUnder(tenantarchive.ImportPrefix(f.id)); err != nil {
		return nil, err
	}
	return access, nil
}

func emptyInstall(ctx context.Context, db *keel.DB) (bool, error) {
	held, err := keel.QueryValue[bool](
		ctx,
		db,
		`SELECT EXISTS (SELECT 1 FROM public.instance_bootstraps) OR EXISTS (SELECT 1 FROM public.organizations)`,
	)
	if err != nil {
		return false, fmt.Errorf("check that the install is empty: %w", err)
	}
	return !held, nil
}

func newEngine(conn *postgres.Connection, deps *Deps) (*archiveengine.Engine, error) {
	cfg := deps.Config
	storage, err := minio.New(minio.Params{StorageConfig: &cfg.Storage, Logger: deps.Logger})
	if err != nil {
		return nil, fmt.Errorf("open the instance's object storage: %w", err)
	}
	return archiveengine.New(archiveengine.Params{
		Rows: tenantcopyrepository.New(tenantcopyrepository.Params{
			DB: conn, App: &cfg.App, Database: &cfg.Database, AIAudit: &cfg.AIAudit,
		}),
		Buckets:   archivebucket.New(),
		Storage:   storage,
		Secrets:   deps.Secrets,
		Blobs:     deps.Blobs,
		Envelopes: deps.Envelopes,
		App:       &cfg.App,
		Reporting: &cfg.Reporting,
		Clock:     func() int64 { return time.Now().Unix() },
		Logger:    deps.Logger,
	}), nil
}

type owner struct {
	Name  string `db:"name"`
	Email string `db:"email_address"`
}

func finalize(
	ctx context.Context,
	conn *postgres.Connection,
	deps *Deps,
	result *archiveengine.ImportResult,
) error {
	people, err := keel.QueryAll(
		ctx,
		deps.DB,
		pgx.RowToStructByName[owner],
		`SELECT name::text AS name, email_address::text AS email_address FROM public.users WHERE id = $1`,
		result.Owner.String(),
	)
	if err != nil || len(people) != 1 {
		return fmt.Errorf(
			"read the imported first administrator: %w",
			errors.Join(err, io.ErrUnexpectedEOF),
		)
	}
	inputs := instancebootstrap.Inputs{
		OrganizationName: result.Manifest.OrganizationName,
		AdminName:        people[0].Name,
		AdminEmail:       strings.ToLower(people[0].Email),
	}
	if err = deps.Finalize(ctx, conn, &instancebootstrapports.FinalizeInstanceBootstrapRequest{
		OrganizationID:     result.Tenant.OrganizationID,
		BusinessUnitID:     result.Tenant.BusinessUnitID,
		AdminUserID:        result.Owner,
		Inputs:             inputs.Normalize(),
		SystemUserPassword: deps.Config.System.SystemUserPassword,
		Now:                time.Now().Unix(),
	}); err != nil {
		return fmt.Errorf("record the instance as set up from the archive: %w", err)
	}
	return nil
}
