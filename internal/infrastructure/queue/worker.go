package queue

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/riverqueue/river"
)

// ShipmentAutoAssignArgs contains arguments for shipment auto-assignment
type ShipmentAutoAssignArgs struct {
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
	ShipmentID     string `json:"shipmentId"`
}

func (ShipmentAutoAssignArgs) Kind() string {
	return "shipment:auto_assign"
}

// ShipmentAutoAssignWorker handles automatic assignment of shipments to drivers
type ShipmentAutoAssignWorker struct {
	river.WorkerDefaults[ShipmentAutoAssignArgs]
}

func (w *ShipmentAutoAssignWorker) Work(ctx context.Context, job *river.Job[ShipmentAutoAssignArgs]) error {
	// Logic for auto-assigning shipment
	return nil
}

// ShipmentStatusUpdateArgs contains arguments for updating shipment status
type ShipmentStatusUpdateArgs struct {
	OrganizationID string          `json:"organizationId"`
	BusinessUnitID string          `json:"businessUnitId"`
	ShipmentID     string          `json:"shipmentId"`
	NewStatus      shipment.Status `json:"newStatus"`
}

func (ShipmentStatusUpdateArgs) Kind() string {
	return "shipment:status_update"
}

// ShipmentStatusUpdateWorker handles shipment status updates
type ShipmentStatusUpdateWorker struct {
	river.WorkerDefaults[ShipmentStatusUpdateArgs]
}

func (w *ShipmentStatusUpdateWorker) Work(ctx context.Context, job *river.Job[ShipmentStatusUpdateArgs]) error {
	// Logic for updating shipment status
	return nil
}

// BillingProcessArgs contains arguments for processing billing
type BillingProcessArgs struct {
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
	ShipmentID     string `json:"shipmentId"`
}

func (BillingProcessArgs) Kind() string {
	return "billing:process"
}

// BillingProcessWorker handles shipment billing processing
type BillingProcessWorker struct {
	river.WorkerDefaults[BillingProcessArgs]
}

func (w *BillingProcessWorker) Work(ctx context.Context, job *river.Job[BillingProcessArgs]) error {
	// Logic for processing billing
	return nil
}

// ShipmentComplianceCheckArgs contains arguments for checking shipment compliance
type ShipmentComplianceCheckArgs struct {
	OrganizationID string `json:"organizationId"`
	BusinessUnitID string `json:"businessUnitId"`
	ShipmentID     string `json:"shipmentId"`
}

func (ShipmentComplianceCheckArgs) Kind() string {
	return "shipment:compliance_check"
}

// ShipmentComplianceCheckWorker handles compliance checks for shipments
type ShipmentComplianceCheckWorker struct {
	river.WorkerDefaults[ShipmentComplianceCheckArgs]
}

func (w *ShipmentComplianceCheckWorker) Work(ctx context.Context, job *river.Job[ShipmentComplianceCheckArgs]) error {
	// Logic for checking compliance
	return nil
}

type ScheduledAliveArgs struct {
	Message string `json:"message"`
}

func (ScheduledAliveArgs) Kind() string {
	return "scheduled:alive"
}

// AliveWorker handles the alive job
type ScheduledAliveWorker struct {
	river.WorkerDefaults[ScheduledAliveArgs]
}

func (w *ScheduledAliveWorker) Work(ctx context.Context, job *river.Job[ScheduledAliveArgs]) error {
	fmt.Printf("ScheduledAliveWorker: %s\n", job.Args.Message)
	return nil
}

// RegisterWorkers registers all worker types
func RegisterWorkers(workers *river.Workers) error {
	if err := river.AddWorkerSafely(workers, &ShipmentAutoAssignWorker{}); err != nil {
		return fmt.Errorf("failed to add ShipmentAutoAssignWorker: %w", err)
	}
	if err := river.AddWorkerSafely(workers, &ShipmentStatusUpdateWorker{}); err != nil {
		return fmt.Errorf("failed to add ShipmentStatusUpdateWorker: %w", err)
	}
	if err := river.AddWorkerSafely(workers, &BillingProcessWorker{}); err != nil {
		return fmt.Errorf("failed to add BillingProcessWorker: %w", err)
	}
	if err := river.AddWorkerSafely(workers, &ShipmentComplianceCheckWorker{}); err != nil {
		return fmt.Errorf("failed to add ShipmentComplianceCheckWorker: %w", err)
	}
	if err := river.AddWorkerSafely(workers, &ScheduledAliveWorker{}); err != nil {
		return fmt.Errorf("failed to add ScheduledAliveWorker: %w", err)
	}

	return nil
}
