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

import { ChoiceProps } from "@/types";

/** Type for fuel method choices */
export type fuelMethodChoicesProps = "D" | "F" | "P";

export const fuelMethodChoices: ChoiceProps[] = [
  { value: "D", label: "Distance" },
  { value: "F", label: "Flat" },
  { value: "P", label: "Percentage" },
];

/** Type for Auto Billing Criteria Choices */
export type AutoBillingCriteriaChoicesProps =
  | "ORDER_DELIVERY"
  | "TRANSFERRED_TO_BILL"
  | "MARKED_READY";

export const autoBillingCriteriaChoices: ChoiceProps[] = [
  { value: "ORDER_DELIVERY", label: "Auto Bill when order is delivered" },
  {
    value: "TRANSFERRED_TO_BILL",
    label: "Auto Bill when order are transferred to billing",
  },
  {
    value: "MARKED_READY",
    label: "Auto Bill when order is marked ready to bill in Billing Queue",
  },
];

/** Type for order transfer criteria */
export type OrderTransferCriteriaChoicesProps =
  | "READY_AND_COMPLETED"
  | "COMPLETED"
  | "READY_TO_BILL";

export const orderTransferCriteriaChoices: ChoiceProps[] = [
  { value: "READY_AND_COMPLETED", label: "Ready to bill & Completed" },
  { value: "COMPLETED", label: "Completed" },
  { value: "READY_TO_BILL", label: "Ready to bill" },
];

/** Type for Bill Type choices */
export type billTypeChoicesProps =
  | "INVOICE"
  | "CREDIT"
  | "DEBIT"
  | "PREPAID"
  | "OTHER";

export const billTypeChoices: ChoiceProps[] = [
  { value: "INVOICE", label: "Invoice" },
  { value: "CREDIT", label: "Credit" },
  { value: "DEBIT", label: "Debit" },
  { value: "PREPAID", label: "Prepaid" },
  { value: "OTHER", label: "Other" },
];

/** Type for Billing Exception choices */
export type billingExceptionChoicesProps =
  | "PAPERWORK"
  | "CHARGE"
  | "CREDIT"
  | "DEBIT"
  | "OTHER";

export const billingExceptionChoices: ChoiceProps[] = [
  { value: "PAPERWORK", label: "Paperwork" },
  { value: "CHARGE", label: "Charge" },
  { value: "CREDIT", label: "Credit" },
  { value: "DEBIT", label: "Debit" },
  { value: "OTHER", label: "OTHER" },
];
