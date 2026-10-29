package capturevolume

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile/captureprofiledb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturerequest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturerequest/capturerequestdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	cvChunk         = 500
	cvAgentVersion  = "1.6.3"
	cvOSVersion     = "Windows 11 Pro 23H2"
	cvTableBatches  = "capture_batches"
	cvTablePages    = "capture_pages"
	cvTableItems    = "capture_items"
	cvTableRequests = "capture_requests"
	cvTableDevices  = "capture_devices"
	cvTableProfiles = "capture_profiles"
	cvTableDocs     = "documents"
)

var cvScanners = []string{
	"Fujitsu fi-8170", "Ricoh fi-7160", "Canon imageFORMULA DR-C240", "Epson DS-870",
	"Brother ADS-4900W", "Kodak S2050",
}

type cvDevice struct {
	entity  *capture.CaptureDevice
	scanner string
}

type cvPhase struct {
	*volumegen.Generator
}

type cvBuilder struct {
	g         *volumegen.Generator
	now       int64
	retention int64
	devices   []*cvDevice
	profiles  []*captureprofile.CaptureProfile
	docTypes  map[string]pulid.ID
	batch     *cvBatch
	counts    map[string]int
	statuses  map[capture.BatchStatus]int
}

type cvBatch struct {
	batches  []*capture.CaptureBatch
	pages    []*capture.CapturePage
	items    []*capture.CaptureItem
	requests []*capturerequest.CaptureRequest
	docs     []*document.Document
	stamps   map[string][]volumegen.Stamp
}

func (b *cvBatch) stamp(table string, id pulid.ID, created, updated int64) {
	b.stamps[table] = append(b.stamps[table], volumegen.Stamp{
		Id:      id,
		Created: created,
		Updated: max(updated, created),
	})
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&cvPhase{Generator: g}).seed(ctx)
}

func (g *cvPhase) seed(ctx context.Context) error {
	control, err := g.DocumentControl(ctx)
	if err != nil {
		return err
	}
	if control != nil && !control.EnableCapture {
		return nil
	}
	days := int64(documentcontrol.DefaultCaptureRetentionDays)
	if control != nil && control.CaptureRetentionDays > 0 {
		days = int64(control.CaptureRetentionDays)
	}

	b := &cvBuilder{
		g:         g.Generator,
		now:       volumegen.Unix(g.Now),
		retention: days * volumegen.Day,
		docTypes:  make(map[string]pulid.ID),
		counts:    make(map[string]int),
		statuses:  make(map[capture.BatchStatus]int),
	}
	if err = b.loadDocTypes(ctx); err != nil {
		return err
	}
	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if err := b.seedProfiles(ctx, tx); err != nil {
			return err
		}
		return b.seedDevices(ctx, tx)
	}); err != nil {
		return err
	}

	stacks := b.planStacks()
	for begin := 0; begin < len(stacks); begin += cvChunk {
		end := min(begin+cvChunk, len(stacks))
		b.batch = &cvBatch{stamps: make(map[string][]volumegen.Stamp)}
		for _, s := range stacks[begin:end] {
			b.build(s)
		}
		if err = g.InTx(ctx, b.write); err != nil {
			return err
		}
		g.Progress("capture batches", end, len(stacks))
	}

	for _, entity := range []struct{ table, label string }{
		{cvTableProfiles, "Capture profiles"},
		{cvTableDevices, "Capture devices"},
		{cvTableRequests, "Capture requests"},
		{cvTableBatches, "Capture batches"},
		{cvTablePages, "Capture pages"},
		{cvTableItems, "Capture items"},
		{cvTableDocs, "Captured documents"},
	} {
		g.Report.Add(entity.label, b.counts[entity.table])
	}
	for _, status := range capture.AllBatchStatuses() {
		if n := b.statuses[status]; n > 0 {
			g.Opts.Logf("  • capture batches %s: %d", status, n)
		}
	}

	return nil
}

func (b *cvBuilder) loadDocTypes(ctx context.Context) error {
	dt := documenttypedb.DocumentTypes
	rows, err := dt.Pick(dt.ID, dt.Code).Where(dt.Code.In("POD", "BOL")).All(ctx, b.g.DB)
	if err != nil {
		return fmt.Errorf("load document types: %w", err)
	}
	for _, row := range rows {
		b.docTypes[row.Code] = row.ID
	}

	return nil
}

func (b *cvBuilder) seedProfiles(ctx context.Context, tx *keel.DB) error {
	cp := captureprofiledb.CaptureProfiles
	existing, err := cp.Query().Order(cp.ID.Asc()).All(ctx, tx)
	if err != nil {
		return fmt.Errorf("load capture profiles: %w", err)
	}
	if len(existing) > 0 {
		for _, p := range existing {
			if p.Status == captureprofile.ProfileActive {
				b.profiles = append(b.profiles, p)
			}
		}
		return nil
	}

	created := volumegen.Unix(b.g.Start) - b.g.Fake.Between64(10, 60)*volumegen.Day
	defs := []*captureprofile.CaptureProfile{
		{
			Name:              "Driver paperwork",
			Description:       "Bills of lading, delivery receipts and lumper slips from driver turn-ins",
			IsDefault:         true,
			DPI:               300,
			PixelType:         capture.PixelBlackWhite,
			Duplex:            true,
			UseFeeder:         true,
			DiscardBlankPages: true,
			JPEGQuality:       80,
			SeparatorStrategies: []captureprofile.SeparatorStrategy{
				captureprofile.SeparatorPatchCode,
				captureprofile.SeparatorBlankPage,
			},
		},
		{
			Name:              "Billing packets",
			Description:       "Grayscale scans of signed paperwork for invoice packets",
			DPI:               200,
			PixelType:         capture.PixelGrayscale,
			Duplex:            true,
			UseFeeder:         true,
			DiscardBlankPages: true,
			JPEGQuality:       75,
			SeparatorStrategies: []captureprofile.SeparatorStrategy{
				captureprofile.SeparatorCoverSheet,
			},
		},
		{
			Name:                "Receipts in color",
			Description:         "Fuel, scale and lumper receipts where the stamp color matters",
			DPI:                 300,
			PixelType:           capture.PixelColor,
			JPEGQuality:         85,
			SeparatorStrategies: []captureprofile.SeparatorStrategy{},
		},
	}
	stamps := make([]volumegen.Stamp, 0, len(defs))
	for _, p := range defs {
		p.ID = volumegen.IDAt("cprf_", created)
		p.BusinessUnitID = b.g.Refs.BuID
		p.OrganizationID = b.g.Refs.OrgID
		p.Status = captureprofile.ProfileActive
		p.Version = 1
		stamps = append(stamps, volumegen.Stamp{Id: p.ID, Created: created, Updated: created})
	}
	if err = volumegen.InsertRows(ctx, tx, cp.Handle, defs); err != nil {
		return err
	}
	if err = volumegen.Backdate(ctx, tx, cp.Handle, stamps); err != nil {
		return err
	}
	b.profiles = defs
	b.counts[cvTableProfiles] += len(defs)

	return nil
}

func (b *cvBuilder) seedDevices(ctx context.Context, tx *keel.DB) error {
	f := b.g.Fake
	users := b.g.Refs.UserIDs
	n := volumegen.Clamp(len(users)*2/3, 1, 15)
	rows := make([]*capture.CaptureDevice, 0, n+1)
	stamps := make([]volumegen.Stamp, 0, n+1)
	for i := range n + 1 {
		created := volumegen.Unix(b.g.Start) - f.Between64(1, 90)*volumegen.Day
		user := users[i%len(users)]
		scanner := volumegen.Pick(f, cvScanners)
		id := volumegen.IDAt("cdev_", created)
		d := &capture.CaptureDevice{
			ID:             id,
			BusinessUnitID: b.g.Refs.BuID,
			OrganizationID: b.g.Refs.OrgID,
			UserID:         user,
			Name: fmt.Sprintf(
				"%s scanner %d",
				volumegen.Pick(f, []string{"Dispatch", "Billing", "Front desk", "Safety"}),
				i+1,
			),
			MachineName:          fmt.Sprintf("TRN-OPS-%03d", f.Between(1, 400)),
			WindowsUser:          "TRENOVA\\" + strings.ToLower(f.FirstName()),
			AgentVersion:         cvAgentVersion,
			Architecture:         capture.ArchitectureX64,
			OSVersion:            cvOSVersion,
			Status:               capture.DeviceActive,
			RefreshTokenHash:     digest("capture-refresh:" + id.String()),
			AccessTokenHash:      digest("capture-access:" + id.String()),
			AccessTokenExpiresAt: b.now + f.Between64(5*volumegen.Minute, volumegen.Hour),
			Sources: []capture.SourceInfo{
				{
					Name:         scanner,
					Protocol:     capture.ProtocolTWAIN,
					Bitness:      64,
					IsDefault:    true,
					Duplex:       true,
					Feeder:       true,
					PatchCodes:   true,
					Barcodes:     true,
					BlankDiscard: true,
					Resolutions:  []int{150, 200, 300, 600},
					PixelTypes: []capture.PixelType{
						capture.PixelBlackWhite,
						capture.PixelGrayscale,
						capture.PixelColor,
					},
				},
			},
			LastSeenAt: volumegen.Ptr(b.now - f.Between64(10, 3*volumegen.Hour)),
			LastIP:     fmt.Sprintf("10.20.%d.%d", f.Between(0, 30), f.Between(2, 250)),
			Version:    1,
		}
		updated := *d.LastSeenAt
		if i == n {
			revoked := b.now - f.Between64(15, 120)*volumegen.Day
			d.Revoke(volumegen.Ptr(b.g.PickUser(users)), "Workstation replaced", revoked)
			d.LastSeenAt = volumegen.Ptr(revoked - f.Between64(volumegen.Hour, 10*volumegen.Day))
			updated = revoked
		} else {
			b.devices = append(b.devices, &cvDevice{entity: d, scanner: scanner})
		}
		rows = append(rows, d)
		stamps = append(stamps, volumegen.Stamp{Id: id, Created: created, Updated: updated})
	}
	if err := volumegen.InsertRows(ctx, tx, capturedb.CaptureDevices.Handle, rows); err != nil {
		return err
	}
	if err := volumegen.Backdate(ctx, tx, capturedb.CaptureDevices.Handle, stamps); err != nil {
		return err
	}
	b.counts[cvTableDevices] += len(rows)

	return nil
}

func (b *cvBuilder) write(ctx context.Context, tx *keel.DB) error {
	batch := b.batch
	steps := []func() error{
		func() error {
			return volumegen.InsertRows(ctx, tx, documentdb.Documents.Handle, batch.docs)
		},
		func() error {
			return volumegen.InsertRows(ctx, tx, capturedb.CaptureBatches.Handle, batch.batches)
		},
		func() error {
			return volumegen.InsertRows(ctx, tx, capturedb.CapturePages.Handle, batch.pages)
		},
		func() error {
			return volumegen.InsertRows(ctx, tx, capturedb.CaptureItems.Handle, batch.items)
		},
		func() error {
			return volumegen.InsertRows(
				ctx,
				tx,
				capturerequestdb.CaptureRequests.Handle,
				batch.requests,
			)
		},
		func() error {
			return volumegen.Backdate(
				ctx,
				tx,
				documentdb.Documents.Handle,
				batch.stamps[cvTableDocs],
			)
		},
		func() error {
			return volumegen.Backdate(
				ctx,
				tx,
				capturedb.CaptureBatches.Handle,
				batch.stamps[cvTableBatches],
			)
		},
		func() error {
			return volumegen.Backdate(
				ctx,
				tx,
				capturedb.CapturePages.Handle,
				batch.stamps[cvTablePages],
			)
		},
		func() error {
			return volumegen.Backdate(
				ctx,
				tx,
				capturedb.CaptureItems.Handle,
				batch.stamps[cvTableItems],
			)
		},
		func() error {
			return volumegen.Backdate(
				ctx,
				tx,
				capturerequestdb.CaptureRequests.Handle,
				batch.stamps[cvTableRequests],
			)
		},
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	b.counts[cvTableDocs] += len(batch.docs)
	b.counts[cvTableBatches] += len(batch.batches)
	b.counts[cvTablePages] += len(batch.pages)
	b.counts[cvTableItems] += len(batch.items)
	b.counts[cvTableRequests] += len(batch.requests)

	return nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
