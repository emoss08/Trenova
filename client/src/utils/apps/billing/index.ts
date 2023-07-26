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

import { TChoiceProps } from "@/types";
import {
  faBuildingColumns,
  faGrid2,
  faHandHoldingDollar,
} from "@fortawesome/pro-duotone-svg-icons";
import { LinksGroupProps } from "@/components/layout/Navbar/_partials/LinksGroup";

/** Type for fuel method choices */
export type fuelMethodChoicesProps = "D" | "F" | "P";

export const fuelMethodChoices: TChoiceProps[] = [
  { value: "D", label: "Distance" },
  { value: "F", label: "Flat" },
  { value: "P", label: "Percentage" },
];

/** Type for Auto Billing Criteria Choices */
export type AutoBillingCriteriaChoicesProps =
  | "ORDER_DELIVERY"
  | "TRANSFERRED_TO_BILL"
  | "MARKED_READY";

export const autoBillingCriteriaChoices: TChoiceProps[] = [
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

export const orderTransferCriteriaChoices: TChoiceProps[] = [
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

/** Links for Billing Navigation Menu */
export const billingNavLinks = [
  {
    label: "Dashboard",
    icon: faGrid2,
    link: "/",
  },
  {
    label: "Billing & Accounting",
    icon: faHandHoldingDollar,
    link: "/billing", // Placeholder, replace with the actual link
    links: [
      {
        label: "Billing Client",
        link: "/billing/client/",
        permission: "billing.use_billing_client",
      },
      {
        label: "Billing Control",
        link: "/admin/control-files#billing-controls/",
        permission: "view_billingcontrol",
      },
      {
        label: "Configuration Files",
        link: "#", // Placeholder, replace with the actual link
        subLinks: [
          {
            label: "Charge Types",
            link: "/billing/charge-types/",
            permission: "view_chargetype",
          },
          {
            label: "Division Codes",
            link: "/accounting/division-codes/",
            permission: "view_divisioncode",
          },
          {
            label: "GL Accounts",
            link: "/accounting/gl-accounts/",
            permission: "view_generalledgeraccount",
          },
          {
            label: "Revenue Codes",
            link: "/accounting/revenue-codes/",
            permission: "view_revenuecode",
          },
          {
            label: "Customers",
            link: "/billing/customers/",
            permission: "view_customer",
          },
        ],
      },
    ],
  },
] as LinksGroupProps[];
