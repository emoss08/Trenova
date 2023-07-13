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

import { TNavigationLink } from "@/types";
import {
  faBox,
  faChartSimple,
  faDatabase,
  faFileInvoiceDollar,
  faFolders,
  faHandHoldingDollar,
  faHeartPulse,
  faInbox,
  faServer,
  faTruckFast,
  faUsers,
  faUserSecret,
  faUserTie,
} from "@fortawesome/pro-duotone-svg-icons";
import { faRoad } from "@fortawesome/pro-duotone-svg-icons/faRoad";

export const adminNavLinks: Record<string, TNavigationLink[]> = {
  "System Health": [
    {
      icon: faHeartPulse,
      title: "System Health",
      description: "Montior the overall health of your system",
      href: "#",
      permission: "admin.view_systemhealth",
    },
    {
      icon: faUserSecret,
      title: "Active Sessions",
      description: "Control & Monitor user sessions",
      href: "#",
      permission: "admin.view_activesessions",
    },
    {
      icon: faChartSimple,
      title: "Active Threads",
      description: "Monitor system processes & threads",
      href: "#",
      permission: "admin.view_activesessions",
    },
    {
      icon: faServer,
      title: "Active DB Triggers",
      description: "Monitor & management database triggers",
      href: "#",
      permission: "admin.view_activetriggers",
    },
    {
      icon: faDatabase,
      title: "Cache Manager",
      description: "Manage & Monitor system cache management",
      href: "#",
      permission: "admin.view_cachemanager",
    },
  ],
  "Configuration Files": [
    {
      icon: faUsers,
      title: "User Management",
      description: "Manage users & their permissions",
      href: "/admin/users",
      permission: "admin.users.view",
    },
    {
      icon: faUserTie,
      title: "Job Titles",
      description: "Manage your organization's job titles & their permissions",
      href: "/accounts/job-titles",
      permission: "view_jobtitles",
    },
  ],
  "Control Files": [
    {
      icon: faFolders,
      title: "Control Files",
      description: "Control & Monitor your organization's billing processes",
      permission: "admin.can_view_all_controls",
      subLinks: [
        {
          icon: faHandHoldingDollar,
          title: "Billing Controls",
          description:
            "Control & Monitor your organization's billing processes",
          href: "/admin/control-files#billing-controls",
          permission: "view_billingcontrol",
        },
        {
          icon: faTruckFast,
          title: "Dispatch Controls",
          description:
            "Control & Oversee your organization's dispatch operations",
          href: "/admin/control-files#dispatch-controls",
          permission: "view_dispatchcontrol",
        },
        {
          icon: faFileInvoiceDollar,
          title: "Invoice Controls",
          description: "Control & Oversee your organization's invoicing",
          href: "/admin/control-files#invoice-controls",
          permission: "view_invoicecontrol",
        },
        {
          icon: faBox,
          title: "Order Controls",
          description:
            "Administer & Manage your organization's order procedures",
          href: "/admin/control-files#order-controls",
          permission: "view_ordercontrol",
        },
        {
          icon: faInbox,
          title: "Email Controls",
          description: "Supervise & Modify your organization's email settings",
          href: "/admin/control-files#email-controls",
          permission: "view_emailcontrol",
        },
        {
          icon: faRoad,
          title: "Route Controls",
          description: "Manage & Optimize your organization's route setting",
          href: "/admin/control-files#route-controls",
          permission: "view_routecontrol",
        },
      ],
    },
  ],
};
