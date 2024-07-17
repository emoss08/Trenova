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



import type { IChoiceProps, TChoiceProps } from "@/types";

/** Type for fuel method choices */
export type FuelMethodChoicesProps = "Distance" | "Flat" | "Percentage";

export const fuelMethodChoices = [
  { value: "Distance", label: "Distance" },
  { value: "Flat", label: "Flat" },
  { value: "Percentage", label: "Percentage" },
] satisfies ReadonlyArray<IChoiceProps<FuelMethodChoicesProps>>;

/** Type for Auto Billing Criteria Choices */
export type AutoBillingCriteriaChoicesProps =
  | "Delivered"
  | "TransferredToBilling"
  | "MarkedReadyToBill";

export const autoBillingCriteriaChoices = [
  { value: "Delivered", label: "Auto Bill when shipment is delivered" },
  {
    value: "TransferredToBilling",
    label: "Auto Bill when order are transferred to billing",
  },
  {
    value: "MarkedReadyToBill",
    label: "Auto Bill when order is marked ready to bill in Billing Queue",
  },
] satisfies ReadonlyArray<IChoiceProps<AutoBillingCriteriaChoicesProps>>;

/** Type for order transfer criteria */
export type ShipmentTransferCriteriaChoicesProps =
  | "ReadyAndCompleted"
  | "Completed"
  | "ReadyToBill";

export const shipmentTransferCriteriaChoices = [
  { value: "ReadyAndCompleted", label: "Ready to bill & Completed" },
  { value: "Completed", label: "Completed" },
  { value: "ReadyToBill", label: "Ready to bill" },
] satisfies ReadonlyArray<IChoiceProps<ShipmentTransferCriteriaChoicesProps>>;

/** Type for Bill Type choices */
export type billTypeChoicesProps =
  | "INVOICE"
  | "CREDIT"
  | "DEBIT"
  | "PREPAID"
  | "OTHER";

export const billTypeChoices: TChoiceProps[] = [
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

export const billingExceptionChoices: TChoiceProps[] = [
  { value: "PAPERWORK", label: "Paperwork" },
  { value: "CHARGE", label: "Charge" },
  { value: "CREDIT", label: "Credit" },
  { value: "DEBIT", label: "Debit" },
  { value: "OTHER", label: "OTHER" },
];
