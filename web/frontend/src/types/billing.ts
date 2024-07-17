/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "@/types/organization";
import type {
  AutoBillingCriteriaChoicesProps,
  FuelMethodChoicesProps,
  ShipmentTransferCriteriaChoicesProps,
} from "@/utils/apps/billing";

/** Types for Billing Control */
export interface BillingControl extends BaseModel {
  id: string;
  removeBillingHistory: boolean;
  autoBillShipment: boolean;
  autoMarkReadyToBill: boolean;
  validateCustomerRates: boolean;
  autoBillCriteria: AutoBillingCriteriaChoicesProps;
  shipmentTransferCriteria: ShipmentTransferCriteriaChoicesProps;
  enforceCustomerBilling: boolean;
}

export type BillingControlFormValues = Omit<
  BillingControl,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

/** Types for Charge Type */
export interface ChargeType extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  description?: string | null;
}

export type ChargeTypeFormValues = Omit<
  ChargeType,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

/** Types for Accessorial Charge */
export interface AccessorialCharge extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description?: string;
  isDetention: boolean;
  amount: string; // Decimal
  method: FuelMethodChoicesProps;
}

export type AccessorialChargeFormValues = Omit<
  AccessorialCharge,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

/** Types for Orders Ready to Bill */
export type OrdersReadyProps = {
  id: string;
  pro_number: string;
  mileage: string;
  other_charge_amount: string;
  freight_charge_amount: string;
  sub_total: string;
  customer_name: string;
  missing_documents: string[];
  is_missing_documents: boolean;
};

/** Types for Billing Queue */
export interface BillingQueue extends BaseModel {
  id: string;
  order_type: string;
  order: string;
  revenue_code: string;
  customer: string;
  invoice_number: string;
  pieces: number;
  weight: number;
  bill_type: string;
  ready_to_bill: boolean;
  bill_date: Date;
  mileage: number;
  worker: string;
  commodity: string;
  commodity_descr: string;
  consignee_ref_number: string;
  other_charge_total: string;
  total_amount: string;
  is_summary: boolean;
  is_cancelled: boolean;
  bol_number: string;
  user: string;
  customer_name: string;
}

/** Types for Billing History */
export interface BillingHistory extends BaseModel {
  id: string;
  organizationId: string;
  orderType: string;
  order: string;
  revenueCode: string;
  customer: string;
  invoiceNumber: string;
  pieces: number;
  weight: number;
  billType: string;
  billDate: string;
  mileage: number;
  worker: string;
  commodity: string;
  commodityDescr: string;
  consigneeRefNumber: string;
  otherChargeTotal: number;
  freightChargeAmount: number;
  totalAmount: number;
  totalAmountCurrency: string;
  isSummary: boolean;
  isCancelled: boolean;
  bolNumber: string;
  user: string;
  created: string;
  modified: string;
}

/** Types for Document Classification */
export interface DocumentClassification extends BaseModel {
  id: string;
  status: StatusChoiceProps;

  code: string;
  description?: string;
  color?: string;
}

/** Types for Document Classification */
export type DocumentClassificationFormValues = Omit<
  DocumentClassification,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;
