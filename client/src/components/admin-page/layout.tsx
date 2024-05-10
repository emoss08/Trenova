import { SidebarLink } from "@/types/sidebar-nav";
import React, { Suspense } from "react";
import { ComponentLoader } from "../ui/component-loader";
import { SidebarNav } from "../user-settings/sidebar-nav";

const links: SidebarLink[] = [
  {
    href: "/admin/dashboard/",
    title: "General Information",
    group: "Organization",
  },
  {
    href: "/admin/accounting-controls/",
    title: "Accounting Controls",
    group: "Organization",
  },
  {
    href: "/admin/billing-controls/",
    title: "Billing Controls",
    group: "Organization",
  },
  {
    href: "/admin/invoice-controls/",
    title: "Invoice Controls",
    group: "Organization",
  },
  {
    href: "/admin/dispatch-controls/",
    title: "Dispatch Controls",
    group: "Organization",
  },
  {
    href: "/admin/shipment-controls/",
    title: "Shipment Controls",
    group: "Organization",
  },
  {
    href: "/admin/route-controls/",
    title: "Route Controls",
    group: "Organization",
  },
  {
    href: "/admin/feasibility-controls/",
    title: "Feasibility Controls",
    group: "Organization",
  },
  {
    href: "/admin/feature-management/",
    title: "Feature Management",
    group: "Organization",
  },
  {
    href: "/admin/hazardous-rules/",
    title: "Hazmat Seg. Rules",
    group: "Organization",
  },
  {
    href: "#",
    title: "Users & Roles",
    group: "Organization",
    disabled: true,
  },

  {
    href: "#",
    title: "Custom Reports",
    group: "Reporting & Analytics",
    disabled: true,
  },
  {
    href: "#",
    title: "Scheduled Reports",
    group: "Reporting & Analytics",
    disabled: true,
  },
  {
    href: "/admin/email-controls/",
    title: "Email Controls",
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Email Logs",
    group: "Email & SMS",
    disabled: true,
  },
  {
    href: "/admin/email-profiles/",
    title: "Email Profile(s)",
    group: "Email & SMS",
  },
  {
    href: "#",
    title: "Notification Types",
    group: "Notifications",
    disabled: true,
  },
  {
    href: "/admin/data-retention/",
    title: "Data Retention",
    group: "Data & Integrations",
    disabled: true,
  },
  {
    href: "/admin/table-change-alerts/",
    title: "Table Change Alerts",
    group: "Data & Integrations",
  },
  {
    href: "/admin/google-api/",
    title: "Google Integration",
    group: "Data & Integrations",
  },
  {
    href: "#",
    title: "Integration Vendor(s)",
    group: "Data & Integrations",
    disabled: true,
  },
  {
    href: "#",
    title: "Document Templates",
    group: "Document Management",
    disabled: true,
  },
  {
    href: "#",
    title: "Document Themes",
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
          <Suspense fallback={<ComponentLoader className="h-[60vh]" />}>
            {children}
          </Suspense>
        </div>
      </div>
    </div>
  );
}
