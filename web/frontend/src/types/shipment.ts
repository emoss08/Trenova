import type {
  CodeTypeProps,
  HazardousClassChoiceProps,
  SegregationTypeChoiceProps,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "./organization";

export interface ShipmentControl extends BaseModel {
  id: string;
  autoRateShipment: boolean;
  calculateDistance: boolean;
  enforceRevCode: boolean;
  enforceVoidedComm: boolean;
  generateRoutes: boolean;
  enforceCommodity: boolean;
  autoSequenceStops: boolean;
  enforceOriginDestination: boolean;
  autoShipmentTotal: boolean;
  checkForDuplicateBol: boolean;
  sendPlacardInfo: boolean;
  enforceHazmatSegRules: boolean;
}

export type ShipmentControlFormValues = Omit<
  ShipmentControl,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export interface ShipmentType extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description?: string;
  color?: string;
}

export type ShipmentTypeFormValues = Omit<
  ShipmentType,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export interface ServiceType extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description?: string;
}

export type ServiceTypeFormValues = Omit<
  ServiceType,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export interface ReasonCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  codeType: CodeTypeProps;
  description: string;
}

export type ReasonCodeFormValues = Omit<
  ReasonCode,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export type ShipmentStatus =
  | "New"
  | "InProgress"
  | "Completed"
  | "Hold"
  | "Billed"
  | "Voided";

export interface Shipment extends BaseModel {
  id: string;
  proNumber: string;
  status: ShipmentStatus;
  shipmentTypeId: string;
  revenueCodeId: string;
  serviceTypeId: string;
  ratingUnit: number;
  ratingMethod: "FlatRate" | "PerMile";
  otherChargeAmount: string;
  freightChargeAmount: string;
  totalChargeAmount: string;
  pieces: number | null;
  weight: number | null;
  readyToBill: boolean;
  billDate: string | null;
  shipDate: string | null;
  billed: boolean;
  transferredToBilling: boolean;
  transferredToBillingDate: string | null;
  trailerTypeId: string;
  tractorTypeId: string;
  temperatureMin: number;
  temperatureMax: number;
  billOfLading: string;
  voidedComment: string;
  autoRated: boolean;
  entryMethod: string;
  createdById: string | null;
  isHarzardous: boolean;
  estimatedDeliveryDate: string | null;
  actualDeliveryDate: string | null;
  originLocationId: string;
  destinationLocationId: string;
  customerId: string;
  priority: number;
  specialInstructions: string;
  trackingNumber: string;
  totalDistance: number | null;
  moves: ShipmentMove[];
}

export type ShipmentFormValues = Omit<
  Shipment,
  | "id"
  | "organizationId"
  | "billDate"
  | "shipDate"
  | "billed"
  | "transferredToBilling"
  | "billingTransferDate"
  | "currentSuffix"
  | "createdAt"
  | "updatedAt"
  | "version"
>;

export type ShipmentMove = {
  createdAt: string;
  updatedAt: string;
  status: ShipmentStatus;
  isLoaded: boolean;
  sequenceNumber: number;
  estimatedDistance: number | null;
  actualDistance: number | null;
  estimatedCost: number | null;
  actualCost: number | null;
  notes: string;
  id: string;
  shipmentId: string;
  tractorId: string;
  trailerId: string;
  primaryWorkerId: string;
  secondaryWorkerId: string | null;
  businessUnitId: string;
  organizationId: string;
  stops: Stop[];
};

export type Stop = {
  status: ShipmentStatus;
  type: "Pickup" | "Delivery";
  addressLine: string;
  notes: string;
  sequenceNumber: number;
  pieces: number | null;
  weight: number | null;
  plannedArrival: string;
  plannedDeparture: string;
  actualArrival: string | null;
  actualDeparture: string | null;
  createdAt: string;
  updatedAt: string;
  id: string;
  businessUnitId: string;
  organizationId: string;
  shipmentMoveId: string;
  locationId: string;
};

export type ShipmentSearchForm = {
  searchQuery: string;
  statusFilter: string;
};

export interface FormulaTemplate extends BaseModel {
  id: string;
  name: string;
  formulaText: string;
  description?: string;
  templateType: string;
  customer?: string;
  shipmentType?: string | null;
  autoApply: boolean;
}

export type ShipmentPageTab = {
  name: string;
  component: React.ComponentType;
  icon: JSX.Element;
  description: string;
};

export interface HazardousMaterialSegregationRule extends BaseModel {
  id: string;
  classA: HazardousClassChoiceProps;
  classB: HazardousClassChoiceProps;
  segregationType: SegregationTypeChoiceProps;
}

export type HazardousMaterialSegregationRuleFormValues = Omit<
  HazardousMaterialSegregationRule,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;
