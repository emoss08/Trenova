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
import { ScrollArea } from "../ui/scroll-area";
import { Skeleton } from "../ui/skeleton";
import { SidebarNav } from "../user-settings/sidebar-nav";

const links = [
  {
    href: "/admin/dashboard/",
    title: "General Information",
    icon: (
      <FontAwesomeIcon
        icon={faBuilding}
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
      />
    ),
    group: "Notifications",
  },
  {
    href: "#",
    title: "Data Retention",
    icon: (
      <FontAwesomeIcon
        icon={faDatabase}
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
      />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Table Change Alerts",
    icon: (
      <FontAwesomeIcon
        icon={faTable}
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
        className="text-muted-foreground group-hover:text-foreground h-4 w-4"
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
    <div className="bg-card flex flex-col rounded-md border p-5 md:flex-row">
      <div className="bg-card sticky top-0 z-10 h-36 overflow-y-auto border-b md:h-screen md:w-64 md:border-none">
        <ScrollArea className="m-0 h-full overflow-y-auto p-0">
          <SidebarNav links={links} />
        </ScrollArea>
      </div>

      <div className="flex-1 overflow-auto md:border-l md:pl-4">
        <Suspense fallback={<Skeleton className="h-full w-full" />}>
          {children}
        </Suspense>
      </div>
    </div>
  );
}
