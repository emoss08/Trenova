/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import type { SidebarLink } from "@/components/sidebar-nav";
import { Resource } from "@/types/audit-entry";
import { RouteInfo } from "@/types/nav-links";
import {
  faDashboard,
  faFiles,
  faGear,
  faRoad,
  faScrewdriverWrench,
  faTruck,
  faVault,
} from "@fortawesome/pro-regular-svg-icons";
import { populateResourcePathMap } from "./utils";

export const adminLinks: SidebarLink[] = [
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
    href: "/organization/consolidation-settings/",
    title: "Consolidation Settings",
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
    href: "/organization/hazmat-segregation-rules/",
    title: "Hazmat Seg. Rules",
    group: "Organization",
  },
  {
    href: "/organization/hold-reasons/",
    title: "Hold Reasons",
    group: "Organization",
  },
  {
    href: "/organization/users/",
    title: "Users & Roles",
    group: "Organization",
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
    href: "/organization/integrations/",
    title: "Apps & Integrations",
    group: "Data & Integrations",
  },
  {
    href: "/organization/pattern-config/",
    title: "Pattern Detection",
    group: "Data & Integrations",
  },
  {
    href: "/organization/data-retention/",
    title: "Data Retention",
    group: "Data & Integrations",
  },
  {
    href: "/organization/docker/",
    title: "Docker Management",
    group: "Data & Integrations",
  },
  {
    href: "/organization/resource-editor/",
    title: "Resource Editor",
    group: "Data & Integrations",
    disabled: false,
  },
  {
    href: "/organization/table-change-alerts/",
    title: "Table Change Alerts",
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

/**
 * Main navigation routes
 */
export const routes: RouteInfo[] = [
  {
    key: Resource.Dashboard,
    label: "Dashboard",
    icon: faDashboard,
    link: "/",
    supportsModal: false,
  },
  {
    key: Resource.BillingManagement,
    label: "Billing Management",
    icon: faVault,
    isDefault: true,
    tree: [
      {
        key: Resource.BillingClient,
        label: "Billing Client",
        link: "/billing/client",
        supportsModal: true,
      },
      {
        key: Resource.ConfigurationFiles,
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: Resource.ChargeType,
            label: "Charge Types",
            link: "/billing/configurations/charge-types",
            supportsModal: true,
          },
          {
            key: Resource.DivisionCode,
            label: "Division Codes",
            link: "/billing/configurations/division-codes",
            supportsModal: true,
          },
          {
            key: Resource.GlAccount,
            label: "GL Accounts",
            link: "/billing/configurations/gl-accounts",
            supportsModal: true,
          },
          {
            key: Resource.RevenueCode,
            label: "Revenue Codes",
            link: "/billing/configurations/revenue-codes",
            supportsModal: true,
          },
          {
            key: Resource.AccessorialCharge,
            label: "Accessorial Charges",
            link: "/billing/configurations/accessorial-charges",
            supportsModal: true,
          },
          {
            key: Resource.Customer,
            label: "Customers",
            link: "/billing/configurations/customers",
            supportsModal: true,
          },
          {
            key: Resource.DocumentType,
            label: "Document Types",
            link: "/billing/configurations/document-types",
            supportsModal: true,
          },
        ],
      },
    ],
  },
  {
    key: Resource.Dispatch,
    label: "Dispatch Management",
    icon: faRoad,
    isDefault: true,
    tree: [
      {
        key: Resource.RateManagement,
        label: "Rate Management",
        link: "/dispatch/rate-management",
        supportsModal: false,
      },
      {
        key: Resource.ConfigurationFiles,
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: Resource.DelayCode,
            label: "Delay Codes",
            link: "/dispatch/configurations/delay-codes",
            supportsModal: true,
          },
          {
            key: Resource.FleetCode,
            label: "Fleet Codes",
            link: "/dispatch/configurations/fleet-codes",
            supportsModal: true,
          },
          {
            key: Resource.Location,
            label: "Locations",
            link: "/dispatch/configurations/locations",
            supportsModal: true,
          },
          {
            key: Resource.LocationCategory,
            label: "Location Categories",
            link: "/dispatch/configurations/location-categories",
            supportsModal: true,
          },
          {
            key: Resource.Worker,
            label: "Workers",
            link: "/dispatch/configurations/workers",
            supportsModal: true,
          },
        ],
      },
    ],
  },
  {
    key: Resource.ShipmentManagement,
    label: "Shipment Management",
    icon: faTruck,
    isDefault: true,
    tree: [
      {
        key: Resource.Shipment,
        label: "Shipment Management",
        link: "/shipments/management",
        supportsModal: true,
      },
      {
        key: Resource.Consolidation,
        label: "Consolidation Groups",
        link: "/shipments/consolidation-groups",
        supportsModal: true,
      },
      {
        key: Resource.ConfigurationFiles,
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: Resource.ShipmentType,
            label: "Shipment Types",
            link: "/shipments/configurations/shipment-types",
            supportsModal: true,
          },
          {
            key: Resource.FormulaTemplate,
            label: "Formula Templates",
            link: "/shipments/configurations/formula-templates",
            supportsModal: true,
          },
          {
            key: Resource.DedicatedLane,
            label: "Dedicated Lanes",
            link: "/shipments/configurations/dedicated-lanes",
            supportsModal: true,
          },
          {
            key: Resource.ServiceType,
            label: "Service Types",
            link: "/shipments/configurations/service-types",
            supportsModal: true,
          },
          {
            key: Resource.Commodity,
            label: "Commodities",
            link: "/shipments/configurations/commodities",
            supportsModal: true,
          },
          {
            key: Resource.HazardousMaterial,
            label: "Hazardous Materials",
            link: "/shipments/configurations/hazardous-materials",
            supportsModal: true,
          },
        ],
      },
    ],
  },
  {
    key: Resource.Equipment,
    label: "Equipment Management",
    icon: faScrewdriverWrench,
    isDefault: true,
    tree: [
      {
        key: Resource.Maintenance,
        label: "Maintenance",
        link: "/equipment/maintenance",
        supportsModal: false,
      },
      {
        key: Resource.ConfigurationFiles,
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: Resource.EquipmentType,
            label: "Equipment Types",
            link: "/equipment/configurations/equipment-types",
            supportsModal: true,
          },
          {
            key: Resource.EquipmentManufacturer,
            label: "Equipment Manufacturers",
            link: "/equipment/configurations/equipment-manufacturers",
            supportsModal: true,
          },
          {
            key: Resource.Tractor,
            label: "Tractors",
            link: "/equipment/configurations/tractors",
            supportsModal: true,
          },
          {
            key: Resource.Trailer,
            label: "Trailers",
            link: "/equipment/configurations/trailers",
            supportsModal: true,
          },
        ],
      },
    ],
  },
  {
    key: Resource.Organization,
    label: "Organization Settings",
    icon: faGear,
    supportsModal: false,
    tree: [
      {
        key: Resource.Setting,
        label: "Organization Settings",
        link: "/organization/settings/",
        supportsModal: false,
      },
      {
        key: Resource.HazmatSegregationRule,
        label: "Hazmat Segregation Rules",
        link: "/organization/hazmat-segregation-rules/",
        supportsModal: true,
      },
      {
        key: Resource.HoldReason,
        label: "Hold Reasons",
        link: "/organization/hold-reasons/",
        supportsModal: true,
      },
      {
        key: Resource.Integration,
        label: "Apps & Integrations",
        link: "/organization/integrations/",
        supportsModal: false,
      },
      {
        key: Resource.ConsolidationSettings,
        label: "Consolidation Settings",
        link: "/organization/consolidation-settings/",
        supportsModal: false,
      },
      {
        key: Resource.ShipmentControl,
        label: "Shipment Controls",
        link: "/organization/shipment-controls/",
        supportsModal: false,
      },
      {
        key: Resource.BillingControl,
        label: "Billing Controls",
        link: "/organization/billing-controls/",
        supportsModal: false,
      },
      {
        key: Resource.DataRetention,
        label: "Data Retention",
        link: "/organization/data-retention/",
        supportsModal: false,
      },
      {
        key: Resource.EmailProfile,
        label: "Email Profile(s)",
        link: "/organization/email-profiles/",
        supportsModal: false,
      },
      {
        key: Resource.User,
        label: "Users & Roles",
        link: "/organization/users/",
        supportsModal: false,
      },
      {
        key: Resource.PatternConfig,
        label: "Pattern Detection",
        link: "/organization/pattern-config/",
        supportsModal: false,
      },
      {
        key: Resource.ResourceEditor,
        label: "Resource Editor",
        link: "/organization/resource-editor/",
        supportsModal: false,
      },
      {
        key: Resource.Docker,
        label: "Docker Management",
        link: "/organization/docker/",
        supportsModal: false,
      },
      {
        key: Resource.AuditEntry,
        label: "Audit Entries",
        link: "/organization/audit-entries/",
        supportsModal: false,
      },
    ],
  },
];

populateResourcePathMap(routes);
