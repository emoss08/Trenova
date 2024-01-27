/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 *
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { faGoogle } from "@fortawesome/free-brands-svg-icons";
import {
  faArrowsRepeat,
  faBook,
  faBoxTaped,
  faBuilding,
  faBuildingColumns,
  faCircleCheck,
  faDatabase,
  faFile,
  faFileInvoice,
  faFiles,
  faFlag,
  faInboxes,
  faMailbox,
  faMoneyBillTransfer,
  faPaperPlane,
  faRoad,
  faSatelliteDish,
  faTable,
  faTruck,
  faWebhook,
} from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import React, { Suspense } from "react";
import { Skeleton } from "../ui/skeleton";
import { SidebarNav } from "../user-settings/sidebar-nav";

const links = [
  {
    href: "/admin/dashboard/",
    title: "General Information",
    icon: (
      <FontAwesomeIcon
        icon={faBuilding}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/accounting-controls/",
    title: "Accounting Controls",
    icon: (
      <FontAwesomeIcon
        icon={faBuildingColumns}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/billing-controls/",
    title: "Billing Controls",
    icon: (
      <FontAwesomeIcon
        icon={faMoneyBillTransfer}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/invoice-controls/",
    title: "Invoice Controls",
    icon: (
      <FontAwesomeIcon
        icon={faFileInvoice}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/dispatch-controls/",
    title: "Dispatch Controls",
    icon: (
      <FontAwesomeIcon
        icon={faTruck}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/shipment-controls/",
    title: "Shipment Controls",
    icon: (
      <FontAwesomeIcon
        icon={faBoxTaped}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/route-controls/",
    title: "Route Controls",
    icon: (
      <FontAwesomeIcon
        icon={faRoad}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/feasibility-controls/",
    title: "Feasibility Controls",
    icon: (
      <FontAwesomeIcon
        icon={faCircleCheck}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/feature-management/",
    title: "Feature Management",
    icon: (
      <FontAwesomeIcon
        icon={faFlag}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Custom Reports",
    icon: (
      <FontAwesomeIcon
        icon={faBook}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Reporting & Analytics",
  },
  {
    href: "#",
    title: "Scheduled Reports",
    icon: (
      <FontAwesomeIcon
        icon={faArrowsRepeat}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Reporting & Analytics",
  },
  {
    href: "/admin/email-controls/",
    title: "Email Controls",
    icon: (
      <FontAwesomeIcon
        icon={faInboxes}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Email Logs",
    icon: (
      <FontAwesomeIcon
        icon={faMailbox}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Email & SMS",
  },
  {
    href: "/admin/email-profiles/",
    title: "Email Profile(s)",
    icon: (
      <FontAwesomeIcon
        icon={faPaperPlane}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Notification Types",
    icon: (
      <FontAwesomeIcon
        icon={faSatelliteDish}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Notifications",
  },
  {
    href: "/admin/data-retention/",
    title: "Data Retention",
    icon: (
      <FontAwesomeIcon
        icon={faDatabase}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Data & Integrations",
  },
  {
    href: "/admin/table-change-alerts/",
    title: "Table Change Alerts",
    icon: (
      <FontAwesomeIcon
        icon={faTable}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Data & Integrations",
  },
  {
    href: "/admin/google-api/",
    title: "Google Integration",
    icon: (
      <FontAwesomeIcon
        icon={faGoogle}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Integration Vendor(s)",
    icon: (
      <FontAwesomeIcon
        icon={faWebhook}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Document Templates",
    icon: (
      <FontAwesomeIcon
        icon={faFiles}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Document Management",
  },
  {
    href: "#",
    title: "Document Themes",
    icon: (
      <FontAwesomeIcon
        icon={faFile}
        className="size-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Document Management",
  },
];

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[240px_minmax(0,1fr)] lg:gap-10">
      <SidebarNav links={links} />
      <main className="relative lg:gap-10">
        <div className="mx-auto min-w-0">
          <Suspense fallback={<Skeleton className="size-full" />}>
            {children}
          </Suspense>
        </div>
      </main>
    </div>
  );
}
