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
        key: Resource.Document,
        label: "Document Studio",
        link: "/billing/documents",
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
            key: Resource.CommentType,
            label: "Comment Types",
            link: "/dispatch/configurations/comment-types",
            supportsModal: true,
          },
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
            key: Resource.Route,
            label: "Routes",
            link: "/dispatch/configurations/routes",
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
        key: Resource.SystemLog,
        label: "System Logs",
        link: "/organization/system-logs/",
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
        key: Resource.AuditEntries,
        label: "Audit Entries",
        link: "/organization/audit-entries/",
        supportsModal: false,
      },
    ],
  },
];

populateResourcePathMap(routes);
