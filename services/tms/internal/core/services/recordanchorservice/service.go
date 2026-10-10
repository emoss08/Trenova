package recordanchorservice

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/orgzone"
	"github.com/emoss08/trenova/internal/core/services/subjectaccess"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	noteWithheld   = "you can no longer read this record, so nothing about it is shown"
	noteMissing    = "it no longer exists, or it is not this organization's"
	noteUnreadable = "it could not be read just now; read it with its get tool before relying on it"
)

type clockPricer interface {
	PriceOpenClock(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		occurrence *detention.DetentionOccurrence,
	) (*detentionservice.ClockPrice, error)
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Permissions services.PermissionEngine
	Tracking    services.ShipmentTrackingReader
	Invoices    repositories.InvoiceRepository
	Tractors    repositories.TractorRepository
	Trailers    repositories.TrailerRepository
	Telematics  repositories.TelematicsRepository
	Detentions  repositories.DetentionOccurrenceRepository
	Pricer      *detentionservice.Service
	Orgs        repositories.OrganizationRepository
}

type Dependencies struct {
	Logger      *zap.Logger
	Permissions services.PermissionEngine
	Tracking    services.ShipmentTrackingReader
	Invoices    repositories.InvoiceRepository
	Tractors    repositories.TractorRepository
	Trailers    repositories.TrailerRepository
	Telematics  repositories.TelematicsRepository
	Detentions  repositories.DetentionOccurrenceRepository
	Pricer      clockPricer
	Orgs        orgzone.OrganizationReader
	Now         func() int64
}

type kindRead struct {
	tenant   pagination.TenantInfo
	ids      []pulid.ID
	labels   map[pulid.ID]string
	timezone string
	zone     *time.Location
	now      int64
}

type kindReader func(ctx context.Context, read *kindRead) (map[pulid.ID]*agentdefinition.RuntimeAnchor, error)

type Service struct {
	logger      *zap.Logger
	orgs        orgzone.OrganizationReader
	permissions services.PermissionEngine
	readers     map[permission.RecordKind]kindReader
	now         func() int64
}

var _ services.RecordAnchorReader = (*Service)(nil)

func New(p Params) services.RecordAnchorReader {
	return NewWithDependencies(&Dependencies{
		Logger:      p.Logger,
		Permissions: p.Permissions,
		Tracking:    p.Tracking,
		Invoices:    p.Invoices,
		Tractors:    p.Tractors,
		Trailers:    p.Trailers,
		Telematics:  p.Telematics,
		Detentions:  p.Detentions,
		Pricer:      p.Pricer,
		Orgs:        p.Orgs,
		Now:         timeutils.NowUnix,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	now := d.Now
	if now == nil {
		now = timeutils.NowUnix
	}
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	readers := make(map[permission.RecordKind]kindReader, 6)
	if d.Tracking != nil {
		readers[permission.RecordKind(permission.ResourceShipment)] = shipmentReader(d.Tracking)
	}
	if d.Invoices != nil {
		readers[permission.RecordKind(permission.ResourceInvoice)] = invoiceReader(d.Invoices)
	}
	if d.Tractors != nil {
		readers[permission.RecordKind(permission.ResourceTractor)] = tractorReader(d.Tractors, d.Telematics)
	}
	if d.Trailers != nil {
		readers[permission.RecordKind(permission.ResourceTrailer)] = trailerReader(d.Trailers)
	}
	if d.Telematics != nil {
		readers[permission.RecordKind(permission.ResourceWorker)] = workerReader(d.Telematics)
	}
	if d.Detentions != nil {
		readers[permission.KindDetentionOccurrence] = detentionReader(d.Detentions, d.Pricer)
	}

	return &Service{
		logger:      logger.Named("record-anchor-service"),
		orgs:        d.Orgs,
		permissions: d.Permissions,
		readers:     readers,
		now:         now,
	}
}

type kindBatch struct {
	kind   permission.RecordKind
	ids    []pulid.ID
	labels map[pulid.ID]string
}

func (s *Service) Anchor(
	ctx context.Context,
	req *services.RecordAnchorRequest,
) []agentdefinition.RuntimeAnchor {
	if req == nil || len(req.Records) == 0 {
		return nil
	}

	anchors, batches := s.plan(req.Records)
	if len(anchors) == 0 {
		return nil
	}
	timezone := s.timezone(ctx, req)
	zone, _ := timeutils.ResolveZone(timezone)
	now := s.now()

	read := make(map[pulid.ID]*agentdefinition.RuntimeAnchor, len(anchors))
	notes := make(map[pulid.ID]string, len(anchors))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, batch := range batches {
		reader, known := s.readers[batch.kind]
		if !known {
			continue
		}
		wg.Go(func() {
			found, note := s.readBatch(ctx, reader, batch, &kindRead{
				tenant:   req.TenantInfo,
				ids:      batch.ids,
				labels:   batch.labels,
				timezone: timezone,
				zone:     zone,
				now:      now,
			}, req.Actor)
			mu.Lock()
			defer mu.Unlock()
			for _, id := range batch.ids {
				if anchor, ok := found[id]; ok {
					read[id] = anchor
					continue
				}
				notes[id] = note
			}
		})
	}
	wg.Wait()

	for idx := range anchors {
		anchor := &anchors[idx]
		id, _ := pulid.Parse(anchor.ID)
		if fresh, ok := read[id]; ok {
			if fresh.Label != "" {
				anchor.Label = fresh.Label
			}
			anchor.Facts = fresh.Facts
			anchor.Note = fresh.Note
			continue
		}
		if note, ok := notes[id]; ok {
			anchor.Note = note
		}
	}

	return anchors
}

func (s *Service) timezone(ctx context.Context, req *services.RecordAnchorRequest) string {
	if req.Timezone != "" || s.orgs == nil {
		return req.Timezone
	}
	zone, err := orgzone.Location(ctx, s.orgs, req.TenantInfo)
	if err != nil {
		s.logger.Debug("records in play are read in UTC", zap.Error(err))
		return ""
	}

	return zone.String()
}

func (s *Service) readBatch(
	ctx context.Context,
	reader kindReader,
	batch *kindBatch,
	read *kindRead,
	actor *services.RequestActor,
) (map[pulid.ID]*agentdefinition.RuntimeAnchor, string) {
	allowed, err := s.mayRead(ctx, actor, batch.kind)
	if err != nil {
		s.logger.Warn("could not check access to records in play",
			zap.String("kind", string(batch.kind)), zap.Error(err))
		return nil, noteUnreadable
	}
	if !allowed {
		return nil, noteWithheld
	}

	found, err := reader(ctx, read)
	if err != nil {
		s.logger.Warn("could not read records in play",
			zap.String("kind", string(batch.kind)), zap.Error(err))
		return nil, noteUnreadable
	}

	return found, noteMissing
}

func (s *Service) plan(
	records []agent.EntityRef,
) ([]agentdefinition.RuntimeAnchor, []*kindBatch) {
	anchors := make([]agentdefinition.RuntimeAnchor, 0, len(records))
	batches := make([]*kindBatch, 0, 4)
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		id := strings.TrimSpace(record.ID)
		if _, dup := seen[id]; dup {
			continue
		}
		kind, ok := kindOf(id)
		if !ok {
			continue
		}
		parsed, err := pulid.Parse(id)
		if err != nil {
			continue
		}
		seen[id] = struct{}{}
		anchors = append(anchors, agentdefinition.RuntimeAnchor{
			Kind: string(kind), ID: id, Label: record.Label,
		})
		if _, readable := s.readers[kind]; !readable {
			continue
		}
		idx := slices.IndexFunc(batches, func(b *kindBatch) bool { return b.kind == kind })
		if idx < 0 {
			batches = append(batches, &kindBatch{kind: kind, labels: map[pulid.ID]string{}})
			idx = len(batches) - 1
		}
		batches[idx].ids = append(batches[idx].ids, parsed)
		batches[idx].labels[parsed] = record.Label
	}

	return anchors, batches
}

func kindOf(id string) (permission.RecordKind, bool) {
	prefix, _, found := strings.Cut(id, "_")
	if !found || prefix == "" {
		return "", false
	}

	return permission.RecordKindOfIDPrefix(prefix + "_")
}

func (s *Service) mayRead(
	ctx context.Context,
	actor *services.RequestActor,
	kind permission.RecordKind,
) (bool, error) {
	return subjectaccess.MayReadResource(ctx, s.permissions, actor, kind.Resource())
}
