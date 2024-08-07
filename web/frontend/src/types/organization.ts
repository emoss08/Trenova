/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import type {
  EmailProtocolChoiceProps,
  EnumDatabaseAction,
  EnumDeliveryMethod,
  RouteDistanceUnitProps,
  RouteModelChoiceProps,
  TimezoneChoices,
} from "@/lib/choices";
import { IChoiceProps, type StatusChoiceProps } from ".";

export type Organization = {
  id: string;
  name: string;
  scacCode: string;
  dotNumber: string;
  orgType: string;
  timezone: TimezoneChoices;
  logoUrl?: string | null;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  state?: {
    abbreviation: string;
    name: string;
  };
  version: string;
  postalCode?: string;
};

export type OrganizationFormValues = Omit<Organization, "id">;

export interface TableChangeAlert extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  databaseAction: EnumDatabaseAction;
  topicName: string;
  deliveryMethod: EnumDeliveryMethod;
  description?: string;
  emailRecipients?: string;
  // conditionalLogic?: object | null;
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

/** Types for EmailControl */
export interface EmailControl extends BaseModel {
  id: string;
  billingEmailProfileId?: string | null;
  rateExpirationEmailProfileId?: string | null;
}

export type EmailControlFormValues = Omit<
  EmailControl,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

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

export type AuditLogAction = "CREATE" | "UPDATE" | "DELETE";

export type AuditLogStatus = "ATTEMPTED" | "SUCCEEDED" | "FAILED";

export const enum EnumAuditLogStatus {
  ATTEMPTED = "ATTEMPTED",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
}

export type AuditLog = {
  id: string;
  status: AuditLogStatus;
  tableName: string;
  entityID: string;
  action: AuditLogAction;
  description?: string;
  errorMessage?: string;
  changes: { [key: string]: any };
  timestamp: string;
  userId: string;
  username: string;
  organizationId: string;
  businessUnitId: string;
};

/**
 * Returns status choices for a select input.
 * @returns An array of status choices.
 */
export const auditLogStatusChoices = [
  { value: "ATTEMPTED", label: "Attempted", color: "#15803d" },
  { value: "SUCCEEDED", label: "Succeeded", color: "#b91c1c" },
  { value: "FAILED", label: "Failed", color: "#b91c1c" },
] satisfies ReadonlyArray<IChoiceProps<AuditLogStatus>>;

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
