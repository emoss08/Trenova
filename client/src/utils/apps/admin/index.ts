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

import { faUserCrown } from "@fortawesome/pro-duotone-svg-icons";
import { LinksGroupProps } from "@/components/layout/Navbar/_partials/LinksGroup";

/** Links for System Health Navigation Menu */
export const adminNavLinks = [
  {
    label: "Administrator",
    icon: faUserCrown,
    link: "/",
    permission: "admin.view_systemhealth",
    links: [
      {
        label: "Active Sessions",
        link: "#",
        permission: "admin.view_activesessions",
      },
      {
        label: "Active Threads",
        link: "#",
        permission: "admin.active_threads",
      },
      {
        label: "Active DB Triggers",
        link: "#",
        permission: "admin.view_activetriggers",
      },
      {
        label: "Cache Manager",
        link: "#",
        permission: "admin.view_cachemanager",
      },
      {
        label: "Configuration Files",
        link: "#", // Placeholder, replace with the actual link
        subLinks: [
          {
            label: "User Management",
            link: "/admin/users",
            permission: "admin.users.view",
          },
          {
            label: "Job Titles",
            link: "/accounts/job-titles",
            permission: "view_jobtitles",
          },
        ],
      },
      {
        label: "Control Files",
        link: "#", // Placeholder, replace with the actual link
        subLinks: [
          {
            label: "Billing Controls",
            link: "/admin/control-files#billing-controls",
            permission: "view_billingcontrol",
          },
          {
            label: "Dispatch Controls",
            link: "/admin/control-files#dispatch-controls",
            permission: "view_dispatchcontrol",
          },
          {
            label: "Invoice Controls",
            link: "/admin/control-files#invoice-controls",
            permission: "view_invoicecontrol",
          },
          {
            label: "Order Controls",
            link: "/admin/control-files#order-controls",
            permission: "view_ordercontrol",
          },
          {
            label: "Email Controls",
            link: "/admin/control-files#email-controls",
            permission: "view_emailcontrol",
          },
          {
            label: "Route Controls",
            link: "/admin/control-files#route-controls",
            permission: "view_routecontrol",
          },
        ],
      },
    ],
  },
] as LinksGroupProps[];
