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

export const fuelMethodChoices = [
  { value: "D", label: "Distance" },
  { value: "F", label: "Flat" },
  { value: "P", label: "Percentage" },
];

export const autoBillingCriteriaChoices = [
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
export const orderTransferCriteriaChoices = [
  { value: "READY_AND_COMPLETED", label: "Ready to bill & Completed" },
  { value: "COMPLETED", label: "Completed" },
  { value: "READY_TO_BILL", label: "Ready to bill" },
];
export const billTypeChoices = [
  { value: "INVOICE", label: "Invoice" },
  { value: "CREDIT", label: "Credit" },
  { value: "DEBIT", label: "Debit" },
  { value: "PREPAID", label: "Prepaid" },
  { value: "OTHER", label: "Other" },
];
export const billingExceptionChoices = [
  { value: "PAPERWORK", label: "Paperwork" },
  { value: "CHARGE", label: "Charge" },
  { value: "CREDIT", label: "Credit" },
  { value: "DEBIT", label: "Debit" },
  { value: "OTHER", label: "OTHER" },
];
