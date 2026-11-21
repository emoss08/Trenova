package archiveengine

import (
	"errors"
	"io"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/archivecrypt"
	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/emoss08/trenova/internal/cloud/copybinary"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	DocumentConcurrency = 4
)

var (
	jsonAPI = sonic.Config{UseNumber: true, SortMapKeys: true}.Froze()

	ErrNoObjectListing = errors.New(
		"the object store cannot list a prefix, so the tenant's documents cannot be archived",
	)
)

type Params struct {
	fx.In

	Rows       archiveports.Rows
	Buckets    archiveports.Buckets
	AuditTrail auditports.TrailVerifier
	AITrail    aiauditports.TrailVerifier
	Storage    kernelports.Client
	Secrets    kernelports.SecretCipher
	Blobs      kernelports.BlobCipher
	Envelopes  *kernelports.EnvelopeSources
	App        *appconfig.Config
	Clock      timeutils.Clock
	Logger     *zap.Logger
}

type Engine struct {
	rows       archiveports.Rows
	buckets    archiveports.Buckets
	auditTrail auditports.TrailVerifier
	aiTrail    aiauditports.TrailVerifier
	storage    kernelports.Client
	secrets    kernelports.SecretCipher
	blobs      kernelports.BlobCipher
	registry   *registry
	release    string
	clock      timeutils.Clock
	l          *zap.Logger
}

//nolint:gocritic // this is dependency injection
func New(p Params) *Engine {
	return &Engine{
		rows:       p.Rows,
		buckets:    p.Buckets,
		auditTrail: p.AuditTrail,
		aiTrail:    p.AITrail,
		storage:    p.Storage,
		secrets:    p.Secrets,
		blobs:      p.Blobs,
		registry:   newRegistry(p.Envelopes),
		release:    p.App.Version,
		clock:      p.Clock,
		l:          p.Logger.Named("cloud.tenant-archive"),
	}
}

type Reporter func(tenantcommand.Progress)

func (r Reporter) report(p *tenantcommand.Progress) {
	if r != nil {
		r(*p)
	}
}

func (e *Engine) prefixes(tenant dbscope.Tenant) []string {
	return objectkey.CarriedPrefixes(tenant.OrganizationID)
}

func (e *Engine) now() time.Time {
	return time.Unix(e.clock(), 0).UTC()
}

func streamOptions(part string, key *archivecrypt.DataKey, compress bool) *archivestream.Options {
	return &archivestream.Options{Part: part, Key: key, Compress: compress}
}

func transform(
	src io.Reader,
	dst io.Writer,
	columns int,
	fn func(*copybinary.Row) error,
) <-chan rewriteResult {
	done := make(chan rewriteResult, 1)
	go func() {
		var (
			rows int64
			err  error
		)
		if fn == nil {
			rows, err = copybinary.Count(dst, src, columns)
		} else {
			rows, err = copybinary.Rewrite(dst, src, columns, fn)
		}
		if closer, ok := src.(*io.PipeReader); ok {
			_ = closer.CloseWithError(err)
		}
		if closer, ok := dst.(*io.PipeWriter); ok {
			_ = closer.CloseWithError(err)
		}
		done <- rewriteResult{rows: rows, err: err}
	}()
	return done
}

type rewriteResult struct {
	rows int64
	err  error
}

func refused(err error) bool {
	_, ok := tenantarchive.RefusalOf(err)
	return ok
}
