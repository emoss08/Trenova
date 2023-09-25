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

import React from "react";
import { faHandHoldingDollar } from "@fortawesome/pro-duotone-svg-icons";
import {
  LinksGroup,
  LinksGroupProps,
} from "@/components/layout/Navbar/_partials/LinksGroup";

/** Links for Billing Navigation Menu */
export const billingNavLinks = [
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
            label: "Accessorial Charges",
            link: "/billing/accessorial-charges/",
            permission: "view_accessorialcharge",
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
] satisfies LinksGroupProps[];

export function BillingLinks() {
  const billingLinks = billingNavLinks.map((item) => (
    <LinksGroup {...item} key={item.label} />
  ));

  return <>{billingLinks}</>;
}
