/*
 * COPYRIGHT(c) 2024 MONTA
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

import { faGoogle } from "@fortawesome/free-brands-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  AlignHorizontalDistributeCenterIcon,
  BellDotIcon,
  BookIcon,
  BuildingIcon,
  CheckCircle,
  CircleDollarSignIcon,
  ConstructionIcon,
  ContainerIcon,
  DatabaseBackupIcon,
  DatabaseZapIcon,
  FileIcon,
  FilesIcon,
  FlagIcon,
  InboxIcon,
  LandmarkIcon,
  MailIcon,
  ReceiptIcon,
  Repeat2Icon,
  SendIcon,
  TruckIcon,
} from "lucide-react";
import React, { Suspense } from "react";
import { ScrollArea } from "../ui/scroll-area";
import { Skeleton } from "../ui/skeleton";
import { SidebarNav } from "../user-settings/sidebar-nav";

const links = [
  {
    href: "/admin/dashboard/",
    title: "General Information",
    icon: (
      <BuildingIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/accounting-controls/",
    title: "Accounting Controls",
    icon: (
      <LandmarkIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/billing-controls/",
    title: "Billing Controls",
    icon: (
      <ReceiptIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/invoice-controls/",
    title: "Invoice Controls",
    icon: (
      <CircleDollarSignIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/dispatch-controls/",
    title: "Dispatch Controls",
    icon: (
      <TruckIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/shipment-controls/",
    title: "Shipment Controls",
    icon: (
      <ContainerIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/route-controls/",
    title: "Route Controls",
    icon: (
      <ConstructionIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/feasibility-controls/",
    title: "Feasibility Controls",
    icon: (
      <CheckCircle className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/feature-management/",
    title: "Feature Management",
    icon: (
      <FlagIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Custom Reports",
    icon: (
      <BookIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Reporting & Analytics",
  },
  {
    href: "#",
    title: "Scheduled Reports",
    icon: (
      <Repeat2Icon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Reporting & Analytics",
  },
  {
    href: "/admin/email-controls/",
    title: "Email Controls",
    icon: (
      <InboxIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Email Logs",
    icon: (
      <MailIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Email Profile(s)",
    icon: (
      <SendIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Notification Types",
    icon: (
      <BellDotIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Notifications",
  },
  {
    href: "#",
    title: "Data Retention",
    icon: (
      <DatabaseBackupIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Table Change Alerts",
    icon: (
      <DatabaseZapIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
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
      <AlignHorizontalDistributeCenterIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Document Templates",
    icon: (
      <FilesIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
    ),
    group: "Document Management",
  },
  {
    href: "#",
    title: "Document Themes",
    icon: (
      <FileIcon className="text-muted-foreground group-hover:text-foreground h-4 w-4" />
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
    <div className="bg-background flex">
      <div className="sticky top-0 h-screen w-64">
        {/* Ensure there's no extra padding or margin that could cut off the content */}
        <ScrollArea className="m-0 h-full overflow-y-auto p-0">
          <SidebarNav links={links} />
        </ScrollArea>
      </div>

      {/* Scrollable Main Content */}
      <div className="mx-6 flex-1 overflow-auto">
        <Suspense fallback={<Skeleton className="h-full w-full" />}>
          {children}
        </Suspense>
      </div>
    </div>
  );
}
