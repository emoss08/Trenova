import { Outlet } from "react-router";
import { LazyComponent } from "./error-boundary";
import { SidebarLink, SidebarNav } from "./sidebar-nav";

const links: SidebarLink[] = [
  {
    href: "/organization/settings/",
    title: "Organization Settings",
    group: "Organization",
  },
  {
    href: "/organization/accounting-controls/",
    title: "Accounting Controls",
    group: "Organization",
    disabled: true,
  },
  {
    href: "/organization/billing-controls/",
    title: "Billing Controls",
    group: "Organization",
  },
  {
    href: "/organization/dispatch-controls/",
    title: "Dispatch Controls",
    group: "Organization",
    disabled: true,
  },
  {
    href: "/organization/shipment-controls/",
    title: "Shipment Controls",
    group: "Organization",
  },
  {
    href: "/organization/route-controls/",
    title: "Route Controls",
    group: "Organization",
    disabled: true,
  },
  {
    href: "/organization/feasibility-controls/",
    title: "Feasibility Controls",
    group: "Organization",
    disabled: true,
  },
  {
    href: "/organization/integrations/",
    title: "Apps & Integrations",
    group: "Organization",
  },
  {
    href: "/organization/feature-management/",
    title: "Feature Management",
    group: "Organization",
    disabled: true,
  },
  {
    href: "/organization/hazmat-segregation-rules/",
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
    href: "/organization/email-controls/",
    title: "Email Controls",
    group: "Email & SMS",
    disabled: true,
  },
  {
    href: "#",
    title: "Email Logs",
    group: "Email & SMS",
    disabled: true,
  },
  {
    href: "/organization/email-profiles/",
    title: "Email Profile(s)",
    group: "Email & SMS",
    disabled: true,
  },
  {
    href: "#",
    title: "Notification Types",
    group: "Notifications",
    disabled: true,
  },
  {
    href: "/organization/audit-entries/",
    title: "Audit Entries",
    group: "Data & Integrations",
  },
  {
    href: "/organization/system-logs/",
    title: "System Logs",
    group: "Data & Integrations",
  },
  {
    href: "/organization/data-retention/",
    title: "Data Retention",
    group: "Data & Integrations",
  },
  {
    href: "/organization/table-change-alerts/",
    title: "Table Change Alerts",
    group: "Data & Integrations",
    disabled: true,
  },
  {
    href: "/organization/google-api/",
    title: "Google Integration",
    group: "Data & Integrations",
    disabled: true,
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
export function AdminLayout() {
  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[240px_minmax(0,1fr)] lg:gap-10">
      <SidebarNav links={links} />
      <div className="relative lg:gap-10">
        <div className="mx-auto min-w-0">
          <LazyComponent>
            <Outlet />
          </LazyComponent>
        </div>
      </div>
    </div>
  );
}
