/*
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

import {
  DatabaseActionChoicesProps,
  EmailProtocolChoiceProps,
  RouteDistanceUnitProps,
  RouteModelChoiceProps,
  SourceChoicesProps,
} from "@/lib/choices";
import { StatusChoiceProps } from "@/types/index";

export type Organization = {
  id: string;
  name: string;
  scacCode: string;
  dotNumber: string;
  orgType: string;
  timezone: string;
  logoUrl?: string | null;
};

export type OrganizationFormValues = Omit<Organization, "id">;

export interface TableChangeAlert extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  databaseAction: DatabaseActionChoicesProps;
  table?: string | null;
  source: SourceChoicesProps;
  topic?: string | null;
  description?: string | null;
  emailProfile?: string | null;
  emailRecipients: string;
  conditionalLogic?: object | null;
  customSubject?: string | null;
  effectiveDate?: string | null;
  expirationDate?: string | null;
}

export type TableChangeAlertFormValues = Omit<
  TableChangeAlert,
  "id" | "organizationId" | "createdAt" | "updatedAt"
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
  defaultProfile: boolean;
}

export type EmailProfileFormValues = Omit<
  EmailProfile,
  "id" | "organizationId" | "createdAt" | "updatedAt"
>;

export type Department = {
  id: string;
  name: string;
  organization: string;
  description: string;
  depot: string;
};

/** Types for EmailControl */
export type EmailControl = {
  id: string;
  organization: string;
  billingEmailProfile?: string | null;
  rateExpirationEmailProfile?: string | null;
};

export type EmailControlFormValues = {
  billingEmailProfile?: string | null;
  rateExpirationEmailProfile?: string | null;
};

export type Depot = BaseModel & {
  id: string;
  name: string;
  description?: string;
};

export type FeatureFlag = {
  name: string;
  code: string;
  description: string;
  enabled: boolean;
  beta: boolean;
  preview: string;
  paidOnly: boolean;
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
  "id" | "organizationId" | "createdAt" | "updatedAt"
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
  createdAt: string;
  updatedAt: string;
};
