import {
  AccessorialChargeMethod,
  BillingCycleType,
  BillingExceptionHandling,
  DocumentCategory,
  DocumentClassification,
  PaymentTerm,
  TransferSchedule,
} from "@/types/billing";
import { type ChoiceProps, Gender, Status } from "@/types/common";
import { EquipmentClass } from "@/types/equipment-type";
import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
} from "@/types/hazardous-material";
import {
  SegregationDistanceUnit,
  SegregationType,
} from "@/types/hazmat-segregation-rule";
import { FacilityType, LocationCategoryType } from "@/types/location-category";
import { MoveStatus } from "@/types/move";
import {
  RatingMethod,
  ShipmentDocumentType,
  ShipmentStatus,
} from "@/types/shipment";
import { StopStatus, StopType } from "@/types/stop";
import { Visibility } from "@/types/table-configuration";
import { EquipmentStatus } from "@/types/tractor";
import { Endorsement, PTOStatus, PTOType, WorkerType } from "@/types/worker";

/**
 * Returns status choices for a select input.
 * @returns An array of status choices.
 */
export const statusChoices = [
  { value: Status.Active, label: "Active", color: "#15803d" },
  { value: Status.Inactive, label: "Inactive", color: "#b91c1c" },
] satisfies ReadonlyArray<ChoiceProps<Status>>;

export const shipmentStatusChoices = [
  { value: ShipmentStatus.New, label: "New", color: "#15803d" },
  {
    value: ShipmentStatus.PartiallyAssigned,
    label: "Partially Assigned",
    color: "#7e22ce",
  },
  { value: ShipmentStatus.Assigned, label: "Assigned", color: "#b91c1c" },
  { value: ShipmentStatus.InTransit, label: "In Transit", color: "#f59e0b" },
  { value: ShipmentStatus.Delayed, label: "Delayed", color: "#0369a1" },
  {
    value: ShipmentStatus.PartiallyCompleted,
    label: "Partially Completed",
    color: "#10b981",
  },
  { value: ShipmentStatus.Completed, label: "Completed", color: "#10b981" },
  { value: ShipmentStatus.Billed, label: "Billed", color: "#ec4899" },
  { value: ShipmentStatus.Canceled, label: "Canceled", color: "#b91c1c" },
] satisfies ReadonlyArray<ChoiceProps<ShipmentStatus>>;

export const ratingMethodChoices = [
  { value: RatingMethod.FlatRate, label: "Flat Rate", color: "#15803d" },
  { value: RatingMethod.PerMile, label: "Per Mile", color: "#7e22ce" },
  { value: RatingMethod.PerStop, label: "Per Stop", color: "#b91c1c" },
  { value: RatingMethod.PerPound, label: "Per Pound", color: "#f59e0b" },
  { value: RatingMethod.PerPallet, label: "Per Pallet", color: "#0369a1" },
  {
    value: RatingMethod.PerLinearFoot,
    label: "Per Linear Foot",
    color: "#10b981",
  },
  { value: RatingMethod.Other, label: "Other", color: "#ec4899" },
] satisfies ReadonlyArray<ChoiceProps<RatingMethod>>;

export const equipmentStatusChoices = [
  { value: EquipmentStatus.Available, label: "Available", color: "#15803d" },
  {
    value: EquipmentStatus.OOS,
    label: "Out of Service",
    color: "#b91c1c",
  },
  {
    value: EquipmentStatus.AtMaintenance,
    label: "At Maintenance",
    color: "#7e22ce",
  },
  { value: EquipmentStatus.Sold, label: "Sold", color: "#f59e0b" },
] satisfies ReadonlyArray<ChoiceProps<EquipmentStatus>>;

export const stopStatusChoices = [
  { value: StopStatus.New, label: "New", color: "#7e22ce" },
  { value: StopStatus.InTransit, label: "In Transit", color: "#1d4ed8" },
  { value: StopStatus.Completed, label: "Completed", color: "#15803d" },
  { value: StopStatus.Canceled, label: "Canceled", color: "#b91c1c" },
] satisfies ReadonlyArray<ChoiceProps<StopStatus>>;

export const moveStatusChoices = [
  { value: MoveStatus.New, label: "New", color: "#7e22ce" },
  { value: MoveStatus.Assigned, label: "Assigned", color: "#1d4ed8" },
  { value: MoveStatus.InTransit, label: "In Transit", color: "#15803d" },
  { value: MoveStatus.Completed, label: "Completed", color: "#15803d" },
  { value: MoveStatus.Canceled, label: "Canceled", color: "#b91c1c" },
] satisfies ReadonlyArray<ChoiceProps<MoveStatus>>;

export const stopTypeChoices = [
  { value: StopType.Pickup, label: "Pickup", color: "#1d4ed8" },
  { value: StopType.Delivery, label: "Delivery", color: "#15803d" },
  { value: StopType.SplitPickup, label: "Split Pickup", color: "#a855f7" },
  { value: StopType.SplitDelivery, label: "Split Delivery", color: "#f59e0b" },
] satisfies ReadonlyArray<ChoiceProps<StopType>>;

export const segregationTypeChoices = [
  { value: SegregationType.Prohibited, label: "Prohibited", color: "#b91c1c" },
  { value: SegregationType.Separated, label: "Separated", color: "#15803d" },
  { value: SegregationType.Distance, label: "Distance", color: "#7e22ce" },
  { value: SegregationType.Barrier, label: "Barrier", color: "#f59e0b" },
] satisfies ReadonlyArray<ChoiceProps<SegregationType>>;

export const segregationDistanceUnitChoices = [
  { value: SegregationDistanceUnit.Feet, label: "Feet", color: "#15803d" },
  { value: SegregationDistanceUnit.Meters, label: "Meters", color: "#7e22ce" },
  { value: SegregationDistanceUnit.Inches, label: "Inches", color: "#f59e0b" },
  {
    value: SegregationDistanceUnit.Centimeters,
    label: "Centimeters",
    color: "#0369a1",
  },
] satisfies ReadonlyArray<ChoiceProps<SegregationDistanceUnit>>;

/**
 * Returns status choices for a select input.
 * @returns An array of status choices.
 */
export const workerTypeChoices = [
  { value: WorkerType.Employee, label: "Employee", color: "#15803d" },
  { value: WorkerType.Contractor, label: "Contractor", color: "#7e22ce" },
] satisfies ReadonlyArray<ChoiceProps<WorkerType>>;

export const endorsementChoices = [
  { value: Endorsement.None, label: "None", color: "#15803d" },
  { value: Endorsement.Tanker, label: "Tanker", color: "#7e22ce" },
  { value: Endorsement.Hazmat, label: "Hazmat", color: "#dc2626" },
  { value: Endorsement.TankerHazmat, label: "Tanker/Hazmat", color: "#f59e0b" },
  { value: Endorsement.Passenger, label: "Passenger", color: "#1d4ed8" },
  {
    value: Endorsement.DoublesTriples,
    label: "Doubles/Triples",
    color: "#0369a1",
  },
] satisfies ReadonlyArray<ChoiceProps<Endorsement>>;

export const equipmentClassChoices = [
  { value: EquipmentClass.Tractor, label: "Tractor", color: "#15803d" },
  { value: EquipmentClass.Trailer, label: "Trailer", color: "#7e22ce" },
  { value: EquipmentClass.Container, label: "Container", color: "#dc2626" },
  { value: EquipmentClass.Other, label: "Other", color: "#f59e0b" },
] satisfies ReadonlyArray<ChoiceProps<EquipmentClass>>;

export const genderChoices = [
  { value: Gender.Male, label: "Male", color: "#1d4ed8" },
  { value: Gender.Female, label: "Female", color: "#ec4899" },
] satisfies ReadonlyArray<ChoiceProps<Gender>>;

export const ptoStatusChoices = [
  { value: PTOStatus.Requested, label: "Requested", color: "#15803d" },
  { value: PTOStatus.Approved, label: "Approved", color: "#7e22ce" },
  { value: PTOStatus.Rejected, label: "Rejected", color: "#b91c1c" },
  { value: PTOStatus.Cancelled, label: "Cancelled", color: "#f59e0b" },
] satisfies ReadonlyArray<ChoiceProps<PTOStatus>>;

export const ptoTypeChoices = [
  { value: PTOType.Vacation, label: "Vacation", color: "#15803d" },
  { value: PTOType.Sick, label: "Sick", color: "#7e22ce" },
  { value: PTOType.Holiday, label: "Holiday", color: "#b91c1c" },
  { value: PTOType.Bereavement, label: "Bereavement", color: "#f59e0b" },
  { value: PTOType.Maternity, label: "Maternity", color: "#0369a1" },
  { value: PTOType.Paternity, label: "Paternity", color: "#0369a1" },
] satisfies ReadonlyArray<ChoiceProps<PTOType>>;

export const visibilityChoices = [
  {
    value: Visibility.Private,
    label: "Private",
    description: "Visible to only the creator",
    color: "#7e22ce",
  },
  {
    value: Visibility.Public,
    label: "Public",
    description: "Visible to all users",
    color: "#15803d",
  },
  {
    value: Visibility.Shared,
    label: "Shared",
    description: "Visible to the creator and those they share with",
    color: "#b91c1c",
  },
] satisfies ReadonlyArray<ChoiceProps<Visibility>>;

export const hazardousClassChoices = [
  {
    value: HazardousClassChoiceProps.HazardClass1And1,
    label: "Division 1.1: Mass Explosive Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And2,
    label: "Division 1.2: Projection Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And3,
    label: "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And4,
    label: "Division 1.4: Minor Explosion Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And5,
    label: "Division 1.5: Very Insensitive With Mass Explosion Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass1And6,
    label: "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard",
  },
  {
    value: HazardousClassChoiceProps.HazardClass2And1,
    label: "Division 2.1: Flammable Gases",
  },
  {
    value: HazardousClassChoiceProps.HazardClass2And2,
    label: "Division 2.2: Non-Flammable Gases",
  },
  {
    value: HazardousClassChoiceProps.HazardClass2And3,
    label: "Division 2.3: Poisonous Gases",
  },
  {
    value: HazardousClassChoiceProps.HazardClass3,
    label: "Division 3: Flammable Liquids",
  },
  {
    value: HazardousClassChoiceProps.HazardClass4And1,
    label: "Division 4.1: Flammable Solids",
  },
  {
    value: HazardousClassChoiceProps.HazardClass4And2,
    label: "Division 4.2: Spontaneously Combustible Solids",
  },
  {
    value: HazardousClassChoiceProps.HazardClass4And3,
    label: "Division 4.3: Dangerous When Wet",
  },
  {
    value: HazardousClassChoiceProps.HazardClass5And1,
    label: "Division 5.1: Oxidizing Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass5And2,
    label: "Division 5.2: Organic Peroxides",
  },
  {
    value: HazardousClassChoiceProps.HazardClass6And1,
    label: "Division 6.1: Toxic Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass6And2,
    label: "Division 6.2: Infectious Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass7,
    label: "Division 7: Radioactive Material",
  },
  {
    value: HazardousClassChoiceProps.HazardClass8,
    label: "Division 8: Corrosive Substances",
  },
  {
    value: HazardousClassChoiceProps.HazardClass9,
    label: "Division 9: Miscellaneous Hazardous Substances and Articles",
  },
] satisfies ReadonlyArray<ChoiceProps<HazardousClassChoiceProps>>;

export const packingGroupChoices = [
  {
    value: PackingGroupChoiceProps.PackingGroupI,
    label: "I (High Danger)",
    color: "#b91c1c",
  },
  {
    value: PackingGroupChoiceProps.PackingGroupII,
    label: "II (Medium Danger)",
    color: "#ca8a04",
  },
  {
    value: PackingGroupChoiceProps.PackingGroupIII,
    label: "III (Low Danger)",
    color: "#16a34a",
  },
] satisfies ReadonlyArray<ChoiceProps<PackingGroupChoiceProps>>;

export const locationCategoryTypeChoices = [
  { value: LocationCategoryType.Terminal, label: "Terminal", color: "#15803d" },
  {
    value: LocationCategoryType.Warehouse,
    label: "Warehouse",
    color: "#7e22ce",
  },
  {
    value: LocationCategoryType.DistributionCenter,
    label: "Distribution Center",
    color: "#dc2626",
  },
  {
    value: LocationCategoryType.TruckStop,
    label: "Truck Stop",
    color: "#f59e0b",
  },
  {
    value: LocationCategoryType.RestArea,
    label: "Rest Area",
    color: "#0369a1",
  },
  {
    value: LocationCategoryType.CustomerLocation,
    label: "Customer Location",
    color: "#10b981",
  },
  { value: LocationCategoryType.Port, label: "Port", color: "#6366f1" },
  {
    value: LocationCategoryType.RailYard,
    label: "Rail Yard",
    color: "#ec4899",
  },
  {
    value: LocationCategoryType.MaintenanceFacility,
    label: "Maintenance Facility",
    color: "#14b8a6",
  },
] satisfies ReadonlyArray<ChoiceProps<LocationCategoryType>>;

export const facilityTypeChoices = [
  { value: FacilityType.CrossDock, label: "Cross Dock", color: "#7e22ce" },
  {
    value: FacilityType.StorageWarehouse,
    label: "Storage Warehouse",
    color: "#dc2626",
  },
  { value: FacilityType.ColdStorage, label: "Cold Storage", color: "#f59e0b" },
  {
    value: FacilityType.HazmatFacility,
    label: "Hazmat Facility",
    color: "#0369a1",
  },
  {
    value: FacilityType.IntermodalFacility,
    label: "Intermodal Facility",
    color: "#10b981",
  },
] satisfies ReadonlyArray<ChoiceProps<FacilityType>>;

export const shipmentDocumentTypes = [
  {
    value: ShipmentDocumentType.BillOfLading,
    label: "Bill of Lading",
    color: "#15803d",
  },
  {
    value: ShipmentDocumentType.ProofOfDelivery,
    label: "Proof of Delivery",
    color: "#7e22ce",
  },
  { value: ShipmentDocumentType.Invoice, label: "Invoice", color: "#b91c1c" },
  {
    value: ShipmentDocumentType.DeliveryReceipt,
    label: "Delivery Receipt",
    color: "#f59e0b",
  },
  { value: ShipmentDocumentType.Other, label: "Other", color: "#0369a1" },
] satisfies ReadonlyArray<ChoiceProps<ShipmentDocumentType>>;

export const billingExceptionHandlingChoices = [
  {
    value: BillingExceptionHandling.Queue,
    label: "Queue",
    description: "Queue the shipment for billing when an exception occurs.",
    color: "#15803d",
  },
  {
    value: BillingExceptionHandling.Notify,
    label: "Notify",
    description: "Notify the user when an exception occurs.",
    color: "#7e22ce",
  },
  {
    value: BillingExceptionHandling.AutoResolve,
    label: "Auto Resolve",
    description: "Automatically resolve the exception.",
    color: "#b91c1c",
  },
  {
    value: BillingExceptionHandling.Reject,
    label: "Reject",
    description: "Reject the shipment when an exception occurs.",
    color: "#f59e0b",
  },
] satisfies ReadonlyArray<ChoiceProps<BillingExceptionHandling>>;

export const transferScheduleChoices = [
  {
    value: TransferSchedule.Continuous,
    label: "Continuous",
    description: "Transfers occur continuously as new shipments are processed.",
    color: "#15803d",
  },
  {
    value: TransferSchedule.Hourly,
    label: "Hourly",
    description: "Transfers occur hourly based on the configured batch size.",
    color: "#7e22ce",
  },
  {
    value: TransferSchedule.Daily,
    label: "Daily",
    description: "Transfers occur daily based on the configured batch size.",
    color: "#b91c1c",
  },
  {
    value: TransferSchedule.Weekly,
    label: "Weekly",
    description: "Transfers occur weekly based on the configured batch size.",
    color: "#f59e0b",
  },
] satisfies ReadonlyArray<ChoiceProps<TransferSchedule>>;

export const paymentTermChoices = [
  {
    value: PaymentTerm.Net15,
    label: "Net 15",
    description: "15 days from the invoice date",
    color: "#15803d",
  },
  {
    value: PaymentTerm.Net30,
    label: "Net 30",
    description: "30 days from the invoice date",
    color: "#7e22ce",
  },
  {
    value: PaymentTerm.Net45,
    label: "Net 45",
    description: "45 days from the invoice date",
    color: "#f59e0b",
  },
  {
    value: PaymentTerm.Net60,
    label: "Net 60",
    description: "60 days from the invoice date",
    color: "#0369a1",
  },
  {
    value: PaymentTerm.Net90,
    label: "Net 90",
    description: "90 days from the invoice date",
    color: "#10b981",
  },
  {
    value: PaymentTerm.DueOnReceipt,
    label: "Due on Receipt",
    description: "Due on receipt of the invoice(s)",
    color: "#ec4899",
  },
] satisfies ReadonlyArray<ChoiceProps<PaymentTerm>>;

export const accessorialChargeMethodChoices = [
  {
    value: AccessorialChargeMethod.Flat,
    label: "Flat",
    color: "#15803d",
    description: "One-time fixed fee regardless of shipment details",
  },
  {
    value: AccessorialChargeMethod.Distance,
    label: "Distance",
    color: "#7e22ce",
    description: "Rate calculated per mile or zone traveled",
  },
  {
    value: AccessorialChargeMethod.Percentage,
    label: "Percentage",
    color: "#f59e0b",
    description: "Fee calculated as a percentage of the base linehaul rate",
  },
] satisfies ReadonlyArray<ChoiceProps<AccessorialChargeMethod>>;

export const documentClassificationChoices = [
  {
    value: DocumentClassification.Public,
    label: "Public",
    color: "#15803d",
    description: "Documents that are publicly available",
  },
  {
    value: DocumentClassification.Private,
    label: "Private",
    color: "#7e22ce",
    description: "Documents for internal use only",
  },
  {
    value: DocumentClassification.Sensitive,
    label: "Sensitive",
    color: "#b91c1c",
    description: "Documents containing sensitive information",
  },
  {
    value: DocumentClassification.Regulatory,
    label: "Regulatory",
    color: "#f59e0b",
    description: "Documents containing regulatory information",
  },
] satisfies ReadonlyArray<ChoiceProps<DocumentClassification>>;

export const documentCategoryChoices = [
  {
    value: DocumentCategory.Shipment,
    label: "Shipment",
    color: "#15803d",
    description: "Documents related to shipments",
  },
  {
    value: DocumentCategory.Worker,
    label: "Worker",
    color: "#7e22ce",
    description: "Documents related to workers",
  },
  {
    value: DocumentCategory.Regulatory,
    label: "Regulatory",
    color: "#f59e0b",
    description: "Documents containing regulatory information",
  },
  {
    value: DocumentCategory.Profile,
    label: "Profile",
    color: "#0369a1",
    description: "Documents related to profiles",
  },
  {
    value: DocumentCategory.Branding,
    label: "Branding",
    color: "#10b981",
    description: "Documents related to branding",
  },
  {
    value: DocumentCategory.Invoice,
    label: "Invoice",
    color: "#6495ed",
    description: "Documents related to invoices",
  },
  {
    value: DocumentCategory.Contract,
    label: "Contract",
    color: "#0369a1",
    description: "Documents related to contracts",
  },
  {
    value: DocumentCategory.Other,
    label: "Other",
    color: "#ec4899",
    description: "Other documents",
  },
] satisfies ReadonlyArray<ChoiceProps<DocumentCategory>>;

export const billingCycleTypeChoices = [
  {
    value: BillingCycleType.Immediate,
    label: "Immediate",
    description: "Billing occurs immediately after the shipment is delivered",
    color: "#15803d",
  },
  {
    value: BillingCycleType.Daily,
    label: "Daily",
    description: "Billing occurs daily",
    color: "#7e22ce",
  },
  {
    value: BillingCycleType.Weekly,
    label: "Weekly",
    description: "Billing occurs weekly",
    color: "#b91c1c",
  },
  {
    value: BillingCycleType.Monthly,
    label: "Monthly",
    description: "Billing occurs monthly",
    color: "#f59e0b",
  },
  {
    value: BillingCycleType.Quarterly,
    label: "Quarterly",
    description: "Billing occurs quarterly",
    color: "#0369a1",
  },
] satisfies ReadonlyArray<ChoiceProps<BillingCycleType>>;
