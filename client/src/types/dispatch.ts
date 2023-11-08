/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { TRateMethodChoices } from "@/lib/constants";
import { BaseModel } from "./organization";
import { StatusChoiceProps } from "@/types/index";

export type DispatchControl = {
  id: string;
  organization: string;
  recordServiceIncident: string;
  gracePeriod: number;
  deadheadTarget: number;
  driverAssign: boolean;
  trailerContinuity: boolean;
  dupeTrailerCheck: boolean;
  regulatoryCheck: boolean;
  prevOrdersOnHold: boolean;
  driverTimeAwayRestriction: boolean;
  tractorWorkerFleetConstraint: boolean;
};

export type DispatchControlFormValues = Omit<
  DispatchControl,
  "id" | "organization"
>;

export interface DelayCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  fCarrierOrDriver: boolean;
}

export type DelayCodeFormValues = Omit<
  DelayCode,
  "organization" | "businessUnit" | "created" | "modified"
>;

export interface FleetCode extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description: string;
  revenueGoal?: string | null;
  deadheadGoal?: string | null;
  mileageGoal?: string | null;
  manager?: string | null;
}

export type FleetCodeFormValues = Omit<
  FleetCode,
  "organization" | "businessUnit" | "created" | "modified"
>;

export interface CommentType extends BaseModel {
  id: string;
  name: string;
  status: StatusChoiceProps;
  description: string;
  created: string;
  modified: string;
}

export type CommentTypeFormValues = Omit<
  CommentType,
  "organization" | "businessUnit" | "created" | "modified" | "id"
>;

export interface Rate extends BaseModel {
  id: string;
  isActive: boolean;
  rateNumber: string;
  customer?: string | null;
  effectiveDate: Date;
  expirationDate: Date;
  commodity?: string | null;
  orderType?: string | null;
  equipmentType?: string | null;
  originLocation?: string | null;
  destinationLocation?: string | null;
  rateMethod: TRateMethodChoices;
  rateAmount: number;
  distanceOverride?: number | null;
  comments?: string | null;
  rateBillingTables?: Array<RateBillingTable> | null;
}

export const rateFields: ReadonlyArray<keyof RateFormValues> = [
  "isActive",
  "rateNumber",
  "customer",
  "effectiveDate",
  "expirationDate",
  "effectiveDate",
  "commodity",
  "orderType",
  "equipmentType",
  "originLocation",
  "destinationLocation",
  "rateMethod",
  "rateAmount",
  "distanceOverride",
  "comments",
];

export type RateBillingTable = {
  organization: string;
  businessUnit: string;
  id: string;
  rate: string;
  accessorialCharge: string;
  description?: string | null;
  unit: number;
  chargeAmount: number;
  subTotal: number;
  created: string;
  modified: string;
};

export type RateBillingTableFormValues = Omit<
  RateBillingTable,
  "id" | "rate" | "organization" | "businessUnit" | "created" | "modified"
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
  organization: string;
  businessUnit: string;
  mpwOperator: string;
  mpwCriteria: number;
  mpdOperator: string;
  mpdCriteria: number;
  mpgOperator: string;
  mpgCriteria: number;
  otpOperator: string;
  otpCriteria: number;
  created: string;
  modified: string;
};

export type FeasibilityToolControlFormValues = Omit<
  FeasibilityToolControl,
  "id" | "organization" | "businessUnit" | "created" | "modified"
>;
