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

import {
  AlignHorizontalDistributeCenterIcon,
  BellDotIcon,
  BookIcon,
  BuildingIcon,
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
import { Skeleton } from "../ui/skeleton";
import { SidebarNav } from "../user-settings/sidebar-nav";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGoogle } from "@fortawesome/free-brands-svg-icons";

const links = [
  {
    href: "/admin/dashboard/",
    title: "General",
    icon: (
      <BuildingIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Accounting Controls",
    icon: (
      <LandmarkIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Billing Controls",
    icon: (
      <ReceiptIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Invoice Controls",
    icon: (
      <CircleDollarSignIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Dispatch Controls",
    icon: (
      <TruckIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Shipment Controls",
    icon: (
      <ContainerIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Route Controls",
    icon: (
      <ConstructionIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "/admin/dashboard/feature-management/",
    title: "Feature Management",
    icon: (
      <FlagIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Custom Reports",
    icon: (
      <BookIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Reporting & Analytics",
  },
  {
    href: "#",
    title: "Scheduled Reports",
    icon: (
      <Repeat2Icon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Reporting & Analytics",
  },
  {
    href: "#",
    title: "Email Controls",
    icon: (
      <InboxIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Email Logs",
    icon: (
      <MailIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Email Profile(s)",
    icon: (
      <SendIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Notification Types",
    icon: (
      <BellDotIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Notifications",
  },
  {
    href: "#",
    title: "Data Retention",
    icon: (
      <DatabaseBackupIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Table Change Alerts",
    icon: (
      <DatabaseZapIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Google Integration",
    icon: (
      <FontAwesomeIcon
        icon={faGoogle}
        className="h-4 w-4 text-muted-foreground group-hover:text-foreground"
      />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Integration Vendor(s)",
    icon: (
      <AlignHorizontalDistributeCenterIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Document Templates",
    icon: (
      <FilesIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
    ),
    group: "Document Management",
  },
  {
    href: "#",
    title: "Document Themes",
    icon: (
      <FileIcon className="h-4 w-4 text-muted-foreground group-hover:text-foreground" />
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
    <div className="flex bg-background">
      <SidebarNav links={links} />
      <div className="mx-12 w-full flex-1">
        <Suspense fallback={<Skeleton className="h-[1000px] w-full" />}>
          {children}
        </Suspense>
      </div>
    </div>
  );
}
