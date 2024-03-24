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

import { StatusChoiceProps } from "@/types/index";
import { BaseModel } from "@/types/organization";
import {
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
  "id" | "organizationId" | "createdAt" | "updatedAt"
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
  "id" | "organizationId" | "createdAt" | "updatedAt"
>;

/** Types for Accessorial Charge */
export interface AccessorialCharge extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  code: string;
  description?: string;
  isDetention: boolean;
  amount: number;
  method: FuelMethodChoicesProps;
}

export type AccessorialChargeFormValues = Omit<
  AccessorialCharge,
  "id" | "organizationId" | "createdAt" | "updatedAt"
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
  name: string;
  description?: string | null;
}

/** Types for Document Classification */
export type DocumentClassificationFormValues = Omit<
  DocumentClassification,
  "id" | "organizationId" | "createdAt" | "updatedAt"
>;
