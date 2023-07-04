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

import {
  AutoBillingCriteriaChoicesProps,
  fuelMethodChoicesProps,
  OrderTransferCriteriaChoicesProps,
} from "@/utils/apps/billing";

/** Types for Division Codes */
export type BillingControl = {
  id: string;
  organization: string;
  remove_billing_history: boolean;
  auto_bill_orders: boolean;
  auto_mark_ready_to_bill: boolean;
  validate_customer_rates: boolean;
  auto_bill_criteria: AutoBillingCriteriaChoicesProps;
  order_transfer_criteria: OrderTransferCriteriaChoicesProps;
  enforce_customer_billing: boolean;
};

export interface BillingControlFormValues {
  remove_billing_history: boolean;
  auto_bill_orders: boolean;
  auto_mark_ready_to_bill: boolean;
  validate_customer_rates: boolean;
  auto_bill_criteria: AutoBillingCriteriaChoicesProps;
  order_transfer_criteria: OrderTransferCriteriaChoicesProps;
  enforce_customer_billing: boolean;
}

/** Types for Division Codes */
export type ChargeType = {
  id: string;
  organization: string;
  name: string;
  description?: string | null;
};

export interface ChargeTypeFormValues
  extends Omit<ChargeType, "id" | "organization"> {}

/** Types for Accessorial Charge */
export type AccessorialCharge = {
  id: string;
  code: string;
  description?: string | null;
  is_detention: boolean;
  charge_amount: number;
  charge_amount_currency: string;
  method: fuelMethodChoicesProps;
};

export interface AccessorialChargeFormValues
  extends Omit<AccessorialCharge, "id" | "charge_amount_currency"> {}
