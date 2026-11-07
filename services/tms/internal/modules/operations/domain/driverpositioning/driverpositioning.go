package driverpositioning

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
)

type Status string

const (
	StatusActive   = Status("Active")
	StatusCanceled = Status("Canceled")
	StatusReplaced = Status("Replaced")
)

const ExpirySeconds = int64(12 * 3600)

type Positioning struct {
	keel.Table `keel:"driver_positionings"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:dpo_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`
	FromCell       int64    `json:"fromCell"       keel:"from_cell"`
	FromLabel      string   `json:"fromLabel"      keel:"from_label"`
	ToCell         int64    `json:"toCell"         keel:"to_cell"`
	ToLabel        string   `json:"toLabel"        keel:"to_label"`
	ToLatitude     float64  `json:"toLatitude"     keel:"to_latitude"`
	ToLongitude    float64  `json:"toLongitude"    keel:"to_longitude"`
	SlotStart      int64    `json:"slotStart"      keel:"slot_start"`
	FreeAt         int64    `json:"freeAt"         keel:"free_at"`
	ArriveAt       int64    `json:"arriveAt"       keel:"arrive_at"`
	Miles          float64  `json:"miles"          keel:"miles"`
	ValueCents     int64    `json:"valueCents"     keel:"value_cents"`
	Status         Status   `json:"status"         keel:"status"`
	CreatedByID    pulid.ID `json:"createdById"    keel:"created_by_id,nullzero"`
	CanceledByID   pulid.ID `json:"canceledById"   keel:"canceled_by_id,nullzero"`
	CanceledAt     *int64   `json:"canceledAt"     keel:"canceled_at"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (p *Positioning) Target() dispatchmarket.Point {
	return dispatchmarket.Point{Latitude: p.ToLatitude, Longitude: p.ToLongitude}
}

func (p *Positioning) ExpiresAt() int64 {
	return p.SlotStart + dispatchmarket.SlotSeconds + ExpirySeconds
}

func (p *Positioning) Live(now int64) bool {
	return p != nil && p.Status == StatusActive && now < p.ExpiresAt()
}
