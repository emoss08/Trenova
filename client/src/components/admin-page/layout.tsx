import { SidebarLink } from "@/types/sidebar-nav";
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
  faUserShield,
  faUsers,
  faWebhook,
} from "@fortawesome/pro-duotone-svg-icons";
import { faTriangleExclamation } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import React, { Suspense } from "react";
import { Skeleton } from "../ui/skeleton";
import { SidebarNav } from "../user-settings/sidebar-nav";

const links: SidebarLink[] = [
  {
    href: "/admin/dashboard/",
    title: "General Information",
    icon: (
      <FontAwesomeIcon
        icon={faBuilding}
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Organization",
  },
  {
    href: "/admin/hazardous-rules/",
    title: "Hazmat Seg. Rules",
    icon: (
      <FontAwesomeIcon
        icon={faTriangleExclamation}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Organization",
  },
  {
    href: "#",
    title: "Users",
    icon: (
      <FontAwesomeIcon
        icon={faUsers}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "User Management",
    disabled: true,
  },
  {
    href: "/admin/roles/",
    title: "Roles",
    icon: (
      <FontAwesomeIcon
        icon={faUserShield}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "User Management",
    disabled: false,
  },
  {
    href: "#",
    title: "Custom Reports",
    icon: (
      <FontAwesomeIcon
        icon={faBook}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Reporting & Analytics",
    disabled: true,
  },
  {
    href: "#",
    title: "Scheduled Reports",
    icon: (
      <FontAwesomeIcon
        icon={faArrowsRepeat}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Reporting & Analytics",
    disabled: true,
  },
  {
    href: "/admin/email-controls/",
    title: "Email Controls",
    icon: (
      <FontAwesomeIcon
        icon={faInboxes}
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Email & SMS",
    disabled: true,
  },
  {
    href: "/admin/email-profiles/",
    title: "Email Profile(s)",
    icon: (
      <FontAwesomeIcon
        icon={faPaperPlane}
        className="text-muted-foreground group-hover:text-foreground size-4"
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
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Notifications",
    disabled: true,
  },
  {
    href: "/admin/data-retention/",
    title: "Data Retention",
    icon: (
      <FontAwesomeIcon
        icon={faDatabase}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Data & Integrations",
    disabled: true,
  },
  {
    href: "/admin/table-change-alerts/",
    title: "Table Change Alerts",
    icon: (
      <FontAwesomeIcon
        icon={faTable}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Data & Integrations",
  },
  {
    href: "/admin/google-api/",
    title: "Google Integration",
    icon: (
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 488 512"
        className="fill-muted-foreground text-muted-foreground size-4 text-xs"
      >
        <path d="M488 261.8C488 403.3 391.1 504 248 504 110.8 504 0 393.2 0 256S110.8 8 248 8c66.8 0 123 24.5 166.3 64.9l-67.5 64.9C258.5 52.6 94.3 116.6 94.3 256c0 86.5 69.1 156.6 153.7 156.6 98.2 0 135-70.4 140.8-106.9H248v-85.3h236.1c2.3 12.7 3.9 24.9 3.9 41.4z" />
      </svg>
    ),
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Integration Vendor(s)",
    icon: (
      <FontAwesomeIcon
        icon={faWebhook}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Data & Integrations",
    disabled: true,
  },
  {
    href: "#",
    title: "Document Templates",
    icon: (
      <FontAwesomeIcon
        icon={faFiles}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Document Management",
    disabled: true,
  },
  {
    href: "#",
    title: "Document Themes",
    icon: (
      <FontAwesomeIcon
        icon={faFile}
        className="text-muted-foreground group-hover:text-foreground size-4"
      />
    ),
    group: "Document Management",
    disabled: true,
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
      <div className="relative lg:gap-10">
        <div className="mx-auto min-w-0">
          <Suspense fallback={<Skeleton className="size-full" />}>
            {children}
          </Suspense>
        </div>
      </div>
    </div>
  );
}
