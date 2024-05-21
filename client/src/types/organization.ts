import type {
  DatabaseActionChoicesProps,
  EmailProtocolChoiceProps,
  RouteDistanceUnitProps,
  RouteModelChoiceProps,
  SourceChoicesProps,
  TimezoneChoices,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";

export type Organization = {
  id: string;
  name: string;
  scacCode: string;
  dotNumber: string;
  orgType: string;
  timezone: TimezoneChoices;
  logoUrl?: string | null;
};

export type OrganizationFormValues = Omit<Organization, "id">;

export interface TableChangeAlert extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  databaseAction: DatabaseActionChoicesProps;
  tableName?: string;
  source: SourceChoicesProps;
  topicName?: string;
  description?: string;
  emailProfile?: string;
  emailRecipients: string;
  conditionalLogic?: object | null;
  customSubject?: string;
  effectiveDate?: string | null;
  expirationDate?: string | null;
}

export type TableChangeAlertFormValues = Omit<
  TableChangeAlert,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export interface EmailProfile extends BaseModel {
  id: string;
  name: string;
  email: string;
  protocol?: EmailProtocolChoiceProps | null;
  host?: string | null;
  port?: number | null;
  username?: string | null;
  password?: string | null;
  isDefault: boolean;
}

export type EmailProfileFormValues = Omit<
  EmailProfile,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export type Department = {
  id: string;
  name: string;
  organization: string;
  description: string;
  depot: string;
};

/** Types for EmailControl */
export interface EmailControl extends BaseModel {
  id: string;
  billingEmailProfileId?: string | null;
  rateExpirtationEmailProfileId?: string | null;
}

export type EmailControlFormValues = Omit<
  EmailControl,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export type Depot = BaseModel & {
  id: string;
  name: string;
  description?: string;
};

export type FeatureFlag = {
  name: string;
  code: string;
  description: string;
  beta: boolean;
  preview: string;
};

export type OrganizationFeatureFlag = {
  isEnabled: boolean;
  edges: {
    featureFlag: FeatureFlag;
  };
};

export type GoogleAPI = BaseModel & {
  id: string;
  apiKey?: string | null;
  mileageUnit: RouteDistanceUnitProps;
  trafficModel: RouteModelChoiceProps;
  addCustomerLocation: boolean;
  addLocation: boolean;
  autoGeocode: boolean;
};

export type TableName = {
  value: string;
  label: string;
};

export type Topic = {
  value: string;
  label: string;
};

export type GoogleAPIFormValues = Omit<
  GoogleAPI,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

/** Base Trenova Interface
 *
 * @note This interface is used for all Trenova models that have the following fields:
 * - organization
 * - created
 * - modified
 *
 * Please do not put businessUnit in this interface. Add it directly to the interface that
 * extends this interface.
 * */
export type BaseModel = {
  organizationId: string;
  version: number;
  createdAt: string;
  updatedAt: string;
};
