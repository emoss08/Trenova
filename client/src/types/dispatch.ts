import type {
  FeasibilityOperatorChoiceProps,
  RatingMethodChoiceProps,
  ServiceIncidentControlEnum,
  SeverityChoiceProps,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { Customer } from "./customer";
import { type BaseModel } from "./organization";

export interface DispatchControl extends BaseModel {
  id: string;
  organizationId: string;
  recordServiceIncident: ServiceIncidentControlEnum;
  gracePeriod: number;
  deadheadTarget: number;
  maxShipmentWeightLimit: number;
  maintenanceCompliance: boolean;
  enforceWorkerAssign: boolean;
  trailerContinuity: boolean;
  dupeTrailerCheck: boolean;
  regulatoryCheck: boolean;
  prevShipmentOnHold: boolean;
  workerTimeAwayRestriction: boolean;
  tractorWorkerFleetConstraint: boolean;
}

export type DispatchControlFormValues = Omit<
  DispatchControl,
  | "organizationId"
  | "businessUnit"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "version"
>;

export interface DelayCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  fCarrierOrDriver: boolean;
  color?: string;
}

export type DelayCodeFormValues = Omit<
  DelayCode,
  | "organizationId"
  | "businessUnit"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "version"
>;

export interface FleetCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  revenueGoal?: number;
  deadheadGoal?: number;
  mileageGoal?: number;
  color?: string;
  managerId?: string | null;
}

export type FleetCodeFormValues = Omit<
  FleetCode,
  | "organizationId"
  | "businessUnit"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "version"
>;

export interface CommentType extends BaseModel {
  id: string;
  name: string;
  status: StatusChoiceProps;
  description: string;
  severity: SeverityChoiceProps;
}

export type CommentTypeFormValues = Omit<
  CommentType,
  | "organizationId"
  | "businessUnit"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "version"
>;

export interface Rate extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  rateNumber: string;
  customerId?: string | null;
  effectiveDate: string;
  expirationDate: string;
  commodityId?: string | null;
  shipmentTypeId?: string | null;
  // equipmentType?: string | null;
  originLocationId?: string | null;
  destinationLocationId?: string | null;
  rateMethod: RatingMethodChoiceProps;
  rateAmount: number;
  comments?: string | null;
  approvedById?: string | null;
  approvedDate?: string;
  usage_count?: number;
  minimumCharge?: number;
  maximumCharge?: number;
  edges?: {
    customer: Customer;
  };
}

export interface RateBillingTable extends BaseModel {
  id: string;
  rate: string;
  accessorialChargeId: string;
  description?: string;
  unit: number;
  chargeAmount: number;
  subTotal: number;
}

export type RateBillingTableFormValues = Omit<
  RateBillingTable,
  | "organizationId"
  | "businessUnit"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "rate"
  | "accessorialChargeId"
>;

export type RateFormValues = Omit<
  Rate,
  | "organization"
  | "businessUnit"
  | "created"
  | "modified"
  | "id"
  | "rateBillingTables"
> & {
  rateBillingTables?: Array<RateBillingTableFormValues> | null;
};

export type FeasibilityToolControl = {
  id: string;
  mpwOperator: FeasibilityOperatorChoiceProps;
  mpwValue: number;
  mpdOperator: FeasibilityOperatorChoiceProps;
  mpdValue: number;
  mpgOperator: FeasibilityOperatorChoiceProps;
  mpgValue: number;
  otpOperator: FeasibilityOperatorChoiceProps;
  otpValue: number;
};

export type FeasibilityToolControlFormValues = Omit<
  FeasibilityToolControl,
  "organizationId" | "businessUnit" | "createdAt" | "updatedAt" | "id"
>;
