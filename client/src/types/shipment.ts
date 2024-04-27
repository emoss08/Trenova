import type {
  CodeTypeProps,
  HazardousClassChoiceProps,
  SegregationTypeChoiceProps,
  ShipmentEntryMethodChoices,
  ShipmentStatusChoiceProps,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "./organization";
import { type StopFormValues } from "./stop";

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

export interface Shipment extends BaseModel {
  id: string;
  proNumber: string;
  shipmentType: string;
  serviceType?: string;
  status: ShipmentStatusChoiceProps;
  revenueCode?: string;
  originAddress?: string;
  originLocation?: string;
  originAppointmentWindowStart: string;
  originAppointmentWindowEnd: string;
  destinationLocation?: string;
  destinationAddress?: string;
  destinationAppointmentWindowStart: string;
  destinationAppointmentWindowEnd: string;
  ratingUnits: number;
  rate?: string;
  mileage?: number;
  otherChargeAmount?: string;
  freightChargeAmount?: string;
  rateMethod?: string;
  customer: string;
  pieces?: number;
  weight?: string;
  readyToBill: boolean;
  billDate?: string;
  shipDate?: string;
  billed: boolean;
  transferredToBilling: boolean;
  billingTransferDate?: string;
  subTotal?: string;
  trailer?: string;
  trailerType: string;
  tractorType?: string;
  enteredBy: string;
  temperatureMin?: string;
  temperatureMax?: string;
  bolNumber: string;
  consigneeRefNumber?: string;
  comment?: string;
  voidedComm?: string;
  autoRate: boolean;
  currentSuffix?: string;
  formulaTemplate?: string;
  entryMethod: ShipmentEntryMethodChoices;
  copyAmount?: number;
  stops?: StopFormValues[];
}

export type ShipmentFormValues = Omit<
  Shipment,
  | "id"
  | "organization"
  | "billDate"
  | "shipDate"
  | "billed"
  | "transferredToBilling"
  | "billingTransferDate"
  | "currentSuffix"
  | "created"
  | "modified"
  | "version"
>;

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
