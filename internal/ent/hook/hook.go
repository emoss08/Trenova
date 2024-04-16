// Code generated by entc, DO NOT EDIT.

package hook

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/ent"
)

// The AccessorialChargeFunc type is an adapter to allow the use of ordinary
// function as AccessorialCharge mutator.
type AccessorialChargeFunc func(context.Context, *ent.AccessorialChargeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f AccessorialChargeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.AccessorialChargeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.AccessorialChargeMutation", m)
}

// The AccountingControlFunc type is an adapter to allow the use of ordinary
// function as AccountingControl mutator.
type AccountingControlFunc func(context.Context, *ent.AccountingControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f AccountingControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.AccountingControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.AccountingControlMutation", m)
}

// The BillingControlFunc type is an adapter to allow the use of ordinary
// function as BillingControl mutator.
type BillingControlFunc func(context.Context, *ent.BillingControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f BillingControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.BillingControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.BillingControlMutation", m)
}

// The BusinessUnitFunc type is an adapter to allow the use of ordinary
// function as BusinessUnit mutator.
type BusinessUnitFunc func(context.Context, *ent.BusinessUnitMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f BusinessUnitFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.BusinessUnitMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.BusinessUnitMutation", m)
}

// The ChargeTypeFunc type is an adapter to allow the use of ordinary
// function as ChargeType mutator.
type ChargeTypeFunc func(context.Context, *ent.ChargeTypeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ChargeTypeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ChargeTypeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ChargeTypeMutation", m)
}

// The CommentTypeFunc type is an adapter to allow the use of ordinary
// function as CommentType mutator.
type CommentTypeFunc func(context.Context, *ent.CommentTypeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f CommentTypeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.CommentTypeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.CommentTypeMutation", m)
}

// The CommodityFunc type is an adapter to allow the use of ordinary
// function as Commodity mutator.
type CommodityFunc func(context.Context, *ent.CommodityMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f CommodityFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.CommodityMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.CommodityMutation", m)
}

// The CustomReportFunc type is an adapter to allow the use of ordinary
// function as CustomReport mutator.
type CustomReportFunc func(context.Context, *ent.CustomReportMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f CustomReportFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.CustomReportMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.CustomReportMutation", m)
}

// The CustomerFunc type is an adapter to allow the use of ordinary
// function as Customer mutator.
type CustomerFunc func(context.Context, *ent.CustomerMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f CustomerFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.CustomerMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.CustomerMutation", m)
}

// The DelayCodeFunc type is an adapter to allow the use of ordinary
// function as DelayCode mutator.
type DelayCodeFunc func(context.Context, *ent.DelayCodeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f DelayCodeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.DelayCodeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.DelayCodeMutation", m)
}

// The DispatchControlFunc type is an adapter to allow the use of ordinary
// function as DispatchControl mutator.
type DispatchControlFunc func(context.Context, *ent.DispatchControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f DispatchControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.DispatchControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.DispatchControlMutation", m)
}

// The DivisionCodeFunc type is an adapter to allow the use of ordinary
// function as DivisionCode mutator.
type DivisionCodeFunc func(context.Context, *ent.DivisionCodeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f DivisionCodeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.DivisionCodeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.DivisionCodeMutation", m)
}

// The DocumentClassificationFunc type is an adapter to allow the use of ordinary
// function as DocumentClassification mutator.
type DocumentClassificationFunc func(context.Context, *ent.DocumentClassificationMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f DocumentClassificationFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.DocumentClassificationMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.DocumentClassificationMutation", m)
}

// The EmailControlFunc type is an adapter to allow the use of ordinary
// function as EmailControl mutator.
type EmailControlFunc func(context.Context, *ent.EmailControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f EmailControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.EmailControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.EmailControlMutation", m)
}

// The EmailProfileFunc type is an adapter to allow the use of ordinary
// function as EmailProfile mutator.
type EmailProfileFunc func(context.Context, *ent.EmailProfileMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f EmailProfileFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.EmailProfileMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.EmailProfileMutation", m)
}

// The EquipmentManufactuerFunc type is an adapter to allow the use of ordinary
// function as EquipmentManufactuer mutator.
type EquipmentManufactuerFunc func(context.Context, *ent.EquipmentManufactuerMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f EquipmentManufactuerFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.EquipmentManufactuerMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.EquipmentManufactuerMutation", m)
}

// The EquipmentTypeFunc type is an adapter to allow the use of ordinary
// function as EquipmentType mutator.
type EquipmentTypeFunc func(context.Context, *ent.EquipmentTypeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f EquipmentTypeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.EquipmentTypeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.EquipmentTypeMutation", m)
}

// The FeasibilityToolControlFunc type is an adapter to allow the use of ordinary
// function as FeasibilityToolControl mutator.
type FeasibilityToolControlFunc func(context.Context, *ent.FeasibilityToolControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f FeasibilityToolControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.FeasibilityToolControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.FeasibilityToolControlMutation", m)
}

// The FeatureFlagFunc type is an adapter to allow the use of ordinary
// function as FeatureFlag mutator.
type FeatureFlagFunc func(context.Context, *ent.FeatureFlagMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f FeatureFlagFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.FeatureFlagMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.FeatureFlagMutation", m)
}

// The FleetCodeFunc type is an adapter to allow the use of ordinary
// function as FleetCode mutator.
type FleetCodeFunc func(context.Context, *ent.FleetCodeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f FleetCodeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.FleetCodeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.FleetCodeMutation", m)
}

// The FormulaTemplateFunc type is an adapter to allow the use of ordinary
// function as FormulaTemplate mutator.
type FormulaTemplateFunc func(context.Context, *ent.FormulaTemplateMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f FormulaTemplateFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.FormulaTemplateMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.FormulaTemplateMutation", m)
}

// The GeneralLedgerAccountFunc type is an adapter to allow the use of ordinary
// function as GeneralLedgerAccount mutator.
type GeneralLedgerAccountFunc func(context.Context, *ent.GeneralLedgerAccountMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f GeneralLedgerAccountFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.GeneralLedgerAccountMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.GeneralLedgerAccountMutation", m)
}

// The GoogleApiFunc type is an adapter to allow the use of ordinary
// function as GoogleApi mutator.
type GoogleApiFunc func(context.Context, *ent.GoogleApiMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f GoogleApiFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.GoogleApiMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.GoogleApiMutation", m)
}

// The HazardousMaterialFunc type is an adapter to allow the use of ordinary
// function as HazardousMaterial mutator.
type HazardousMaterialFunc func(context.Context, *ent.HazardousMaterialMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f HazardousMaterialFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.HazardousMaterialMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.HazardousMaterialMutation", m)
}

// The HazardousMaterialSegregationFunc type is an adapter to allow the use of ordinary
// function as HazardousMaterialSegregation mutator.
type HazardousMaterialSegregationFunc func(context.Context, *ent.HazardousMaterialSegregationMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f HazardousMaterialSegregationFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.HazardousMaterialSegregationMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.HazardousMaterialSegregationMutation", m)
}

// The InvoiceControlFunc type is an adapter to allow the use of ordinary
// function as InvoiceControl mutator.
type InvoiceControlFunc func(context.Context, *ent.InvoiceControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f InvoiceControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.InvoiceControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.InvoiceControlMutation", m)
}

// The LocationFunc type is an adapter to allow the use of ordinary
// function as Location mutator.
type LocationFunc func(context.Context, *ent.LocationMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f LocationFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.LocationMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.LocationMutation", m)
}

// The LocationCategoryFunc type is an adapter to allow the use of ordinary
// function as LocationCategory mutator.
type LocationCategoryFunc func(context.Context, *ent.LocationCategoryMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f LocationCategoryFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.LocationCategoryMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.LocationCategoryMutation", m)
}

// The LocationCommentFunc type is an adapter to allow the use of ordinary
// function as LocationComment mutator.
type LocationCommentFunc func(context.Context, *ent.LocationCommentMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f LocationCommentFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.LocationCommentMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.LocationCommentMutation", m)
}

// The LocationContactFunc type is an adapter to allow the use of ordinary
// function as LocationContact mutator.
type LocationContactFunc func(context.Context, *ent.LocationContactMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f LocationContactFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.LocationContactMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.LocationContactMutation", m)
}

// The OrganizationFunc type is an adapter to allow the use of ordinary
// function as Organization mutator.
type OrganizationFunc func(context.Context, *ent.OrganizationMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f OrganizationFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.OrganizationMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.OrganizationMutation", m)
}

// The OrganizationFeatureFlagFunc type is an adapter to allow the use of ordinary
// function as OrganizationFeatureFlag mutator.
type OrganizationFeatureFlagFunc func(context.Context, *ent.OrganizationFeatureFlagMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f OrganizationFeatureFlagFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.OrganizationFeatureFlagMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.OrganizationFeatureFlagMutation", m)
}

// The QualifierCodeFunc type is an adapter to allow the use of ordinary
// function as QualifierCode mutator.
type QualifierCodeFunc func(context.Context, *ent.QualifierCodeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f QualifierCodeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.QualifierCodeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.QualifierCodeMutation", m)
}

// The ReasonCodeFunc type is an adapter to allow the use of ordinary
// function as ReasonCode mutator.
type ReasonCodeFunc func(context.Context, *ent.ReasonCodeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ReasonCodeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ReasonCodeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ReasonCodeMutation", m)
}

// The RevenueCodeFunc type is an adapter to allow the use of ordinary
// function as RevenueCode mutator.
type RevenueCodeFunc func(context.Context, *ent.RevenueCodeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f RevenueCodeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.RevenueCodeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.RevenueCodeMutation", m)
}

// The RouteControlFunc type is an adapter to allow the use of ordinary
// function as RouteControl mutator.
type RouteControlFunc func(context.Context, *ent.RouteControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f RouteControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.RouteControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.RouteControlMutation", m)
}

// The ServiceTypeFunc type is an adapter to allow the use of ordinary
// function as ServiceType mutator.
type ServiceTypeFunc func(context.Context, *ent.ServiceTypeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ServiceTypeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ServiceTypeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ServiceTypeMutation", m)
}

// The SessionFunc type is an adapter to allow the use of ordinary
// function as Session mutator.
type SessionFunc func(context.Context, *ent.SessionMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f SessionFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.SessionMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.SessionMutation", m)
}

// The ShipmentFunc type is an adapter to allow the use of ordinary
// function as Shipment mutator.
type ShipmentFunc func(context.Context, *ent.ShipmentMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentMutation", m)
}

// The ShipmentChargesFunc type is an adapter to allow the use of ordinary
// function as ShipmentCharges mutator.
type ShipmentChargesFunc func(context.Context, *ent.ShipmentChargesMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentChargesFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentChargesMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentChargesMutation", m)
}

// The ShipmentCommentFunc type is an adapter to allow the use of ordinary
// function as ShipmentComment mutator.
type ShipmentCommentFunc func(context.Context, *ent.ShipmentCommentMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentCommentFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentCommentMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentCommentMutation", m)
}

// The ShipmentCommodityFunc type is an adapter to allow the use of ordinary
// function as ShipmentCommodity mutator.
type ShipmentCommodityFunc func(context.Context, *ent.ShipmentCommodityMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentCommodityFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentCommodityMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentCommodityMutation", m)
}

// The ShipmentControlFunc type is an adapter to allow the use of ordinary
// function as ShipmentControl mutator.
type ShipmentControlFunc func(context.Context, *ent.ShipmentControlMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentControlFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentControlMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentControlMutation", m)
}

// The ShipmentDocumentationFunc type is an adapter to allow the use of ordinary
// function as ShipmentDocumentation mutator.
type ShipmentDocumentationFunc func(context.Context, *ent.ShipmentDocumentationMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentDocumentationFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentDocumentationMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentDocumentationMutation", m)
}

// The ShipmentMoveFunc type is an adapter to allow the use of ordinary
// function as ShipmentMove mutator.
type ShipmentMoveFunc func(context.Context, *ent.ShipmentMoveMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentMoveFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentMoveMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentMoveMutation", m)
}

// The ShipmentTypeFunc type is an adapter to allow the use of ordinary
// function as ShipmentType mutator.
type ShipmentTypeFunc func(context.Context, *ent.ShipmentTypeMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f ShipmentTypeFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.ShipmentTypeMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.ShipmentTypeMutation", m)
}

// The StopFunc type is an adapter to allow the use of ordinary
// function as Stop mutator.
type StopFunc func(context.Context, *ent.StopMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f StopFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.StopMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.StopMutation", m)
}

// The TableChangeAlertFunc type is an adapter to allow the use of ordinary
// function as TableChangeAlert mutator.
type TableChangeAlertFunc func(context.Context, *ent.TableChangeAlertMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f TableChangeAlertFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.TableChangeAlertMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.TableChangeAlertMutation", m)
}

// The TagFunc type is an adapter to allow the use of ordinary
// function as Tag mutator.
type TagFunc func(context.Context, *ent.TagMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f TagFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.TagMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.TagMutation", m)
}

// The TractorFunc type is an adapter to allow the use of ordinary
// function as Tractor mutator.
type TractorFunc func(context.Context, *ent.TractorMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f TractorFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.TractorMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.TractorMutation", m)
}

// The TrailerFunc type is an adapter to allow the use of ordinary
// function as Trailer mutator.
type TrailerFunc func(context.Context, *ent.TrailerMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f TrailerFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.TrailerMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.TrailerMutation", m)
}

// The UsStateFunc type is an adapter to allow the use of ordinary
// function as UsState mutator.
type UsStateFunc func(context.Context, *ent.UsStateMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f UsStateFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.UsStateMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.UsStateMutation", m)
}

// The UserFunc type is an adapter to allow the use of ordinary
// function as User mutator.
type UserFunc func(context.Context, *ent.UserMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f UserFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.UserMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.UserMutation", m)
}

// The UserFavoriteFunc type is an adapter to allow the use of ordinary
// function as UserFavorite mutator.
type UserFavoriteFunc func(context.Context, *ent.UserFavoriteMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f UserFavoriteFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.UserFavoriteMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.UserFavoriteMutation", m)
}

// The UserReportFunc type is an adapter to allow the use of ordinary
// function as UserReport mutator.
type UserReportFunc func(context.Context, *ent.UserReportMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f UserReportFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.UserReportMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.UserReportMutation", m)
}

// The WorkerFunc type is an adapter to allow the use of ordinary
// function as Worker mutator.
type WorkerFunc func(context.Context, *ent.WorkerMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f WorkerFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.WorkerMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.WorkerMutation", m)
}

// The WorkerCommentFunc type is an adapter to allow the use of ordinary
// function as WorkerComment mutator.
type WorkerCommentFunc func(context.Context, *ent.WorkerCommentMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f WorkerCommentFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.WorkerCommentMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.WorkerCommentMutation", m)
}

// The WorkerContactFunc type is an adapter to allow the use of ordinary
// function as WorkerContact mutator.
type WorkerContactFunc func(context.Context, *ent.WorkerContactMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f WorkerContactFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.WorkerContactMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.WorkerContactMutation", m)
}

// The WorkerProfileFunc type is an adapter to allow the use of ordinary
// function as WorkerProfile mutator.
type WorkerProfileFunc func(context.Context, *ent.WorkerProfileMutation) (ent.Value, error)

// Mutate calls f(ctx, m).
func (f WorkerProfileFunc) Mutate(ctx context.Context, m ent.Mutation) (ent.Value, error) {
	if mv, ok := m.(*ent.WorkerProfileMutation); ok {
		return f(ctx, mv)
	}
	return nil, fmt.Errorf("unexpected mutation type %T. expect *ent.WorkerProfileMutation", m)
}

// Condition is a hook condition function.
type Condition func(context.Context, ent.Mutation) bool

// And groups conditions with the AND operator.
func And(first, second Condition, rest ...Condition) Condition {
	return func(ctx context.Context, m ent.Mutation) bool {
		if !first(ctx, m) || !second(ctx, m) {
			return false
		}
		for _, cond := range rest {
			if !cond(ctx, m) {
				return false
			}
		}
		return true
	}
}

// Or groups conditions with the OR operator.
func Or(first, second Condition, rest ...Condition) Condition {
	return func(ctx context.Context, m ent.Mutation) bool {
		if first(ctx, m) || second(ctx, m) {
			return true
		}
		for _, cond := range rest {
			if cond(ctx, m) {
				return true
			}
		}
		return false
	}
}

// Not negates a given condition.
func Not(cond Condition) Condition {
	return func(ctx context.Context, m ent.Mutation) bool {
		return !cond(ctx, m)
	}
}

// HasOp is a condition testing mutation operation.
func HasOp(op ent.Op) Condition {
	return func(_ context.Context, m ent.Mutation) bool {
		return m.Op().Is(op)
	}
}

// HasAddedFields is a condition validating `.AddedField` on fields.
func HasAddedFields(field string, fields ...string) Condition {
	return func(_ context.Context, m ent.Mutation) bool {
		if _, exists := m.AddedField(field); !exists {
			return false
		}
		for _, field := range fields {
			if _, exists := m.AddedField(field); !exists {
				return false
			}
		}
		return true
	}
}

// HasClearedFields is a condition validating `.FieldCleared` on fields.
func HasClearedFields(field string, fields ...string) Condition {
	return func(_ context.Context, m ent.Mutation) bool {
		if exists := m.FieldCleared(field); !exists {
			return false
		}
		for _, field := range fields {
			if exists := m.FieldCleared(field); !exists {
				return false
			}
		}
		return true
	}
}

// HasFields is a condition validating `.Field` on fields.
func HasFields(field string, fields ...string) Condition {
	return func(_ context.Context, m ent.Mutation) bool {
		if _, exists := m.Field(field); !exists {
			return false
		}
		for _, field := range fields {
			if _, exists := m.Field(field); !exists {
				return false
			}
		}
		return true
	}
}

// If executes the given hook under condition.
//
//	hook.If(ComputeAverage, And(HasFields(...), HasAddedFields(...)))
func If(hk ent.Hook, cond Condition) ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if cond(ctx, m) {
				return hk(next).Mutate(ctx, m)
			}
			return next.Mutate(ctx, m)
		})
	}
}

// On executes the given hook only for the given operation.
//
//	hook.On(Log, ent.Delete|ent.Create)
func On(hk ent.Hook, op ent.Op) ent.Hook {
	return If(hk, HasOp(op))
}

// Unless skips the given hook only for the given operation.
//
//	hook.Unless(Log, ent.Update|ent.UpdateOne)
func Unless(hk ent.Hook, op ent.Op) ent.Hook {
	return If(hk, Not(HasOp(op)))
}

// FixedError is a hook returning a fixed error.
func FixedError(err error) ent.Hook {
	return func(ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(context.Context, ent.Mutation) (ent.Value, error) {
			return nil, err
		})
	}
}

// Reject returns a hook that rejects all operations that match op.
//
//	func (T) Hooks() []ent.Hook {
//		return []ent.Hook{
//			Reject(ent.Delete|ent.Update),
//		}
//	}
func Reject(op ent.Op) ent.Hook {
	hk := FixedError(fmt.Errorf("%s operation is not allowed", op))
	return On(hk, op)
}

// Chain acts as a list of hooks and is effectively immutable.
// Once created, it will always hold the same set of hooks in the same order.
type Chain struct {
	hooks []ent.Hook
}

// NewChain creates a new chain of hooks.
func NewChain(hooks ...ent.Hook) Chain {
	return Chain{append([]ent.Hook(nil), hooks...)}
}

// Hook chains the list of hooks and returns the final hook.
func (c Chain) Hook() ent.Hook {
	return func(mutator ent.Mutator) ent.Mutator {
		for i := len(c.hooks) - 1; i >= 0; i-- {
			mutator = c.hooks[i](mutator)
		}
		return mutator
	}
}

// Append extends a chain, adding the specified hook
// as the last ones in the mutation flow.
func (c Chain) Append(hooks ...ent.Hook) Chain {
	newHooks := make([]ent.Hook, 0, len(c.hooks)+len(hooks))
	newHooks = append(newHooks, c.hooks...)
	newHooks = append(newHooks, hooks...)
	return Chain{newHooks}
}

// Extend extends a chain, adding the specified chain
// as the last ones in the mutation flow.
func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.hooks...)
}