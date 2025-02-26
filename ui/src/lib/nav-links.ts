import { type routeInfo } from "@/types/nav-links";
import {
  faDashboard,
  faFiles,
  faGear,
  faRoad,
  faScrewdriverWrench,
  faTruck,
  faVault,
} from "@fortawesome/pro-regular-svg-icons";

export const routes: routeInfo[] = [
  {
    key: "dashboard",
    label: "Dashboard",
    icon: faDashboard,
    link: "/",
  },
  {
    key: "billing-management",
    label: "Billing Management",
    icon: faVault,
    tree: [
      {
        key: "billing-client",
        label: "Billing Client",
        link: "/billing/client",
      },
      {
        key: "configuration-files",
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: "charge-types",
            label: "Charge Types",
            link: "/billing/configurations/charge-types",
          },
          {
            key: "division-codes",
            label: "Division Codes",
            link: "/billing/configurations/division-codes",
          },
          {
            key: "gl-accounts",
            label: "GL Accounts",
            link: "/billing/configurations/gl-accounts",
          },
          {
            key: "revenue-codes",
            label: "Revenue Codes",
            link: "/billing/configurations/revenue-codes",
          },
          {
            key: "accessorial-charges",
            label: "Accessorial Charges",
            link: "/billing/configurations/accessorial-charges",
          },
          {
            key: "customers",
            label: "Customers",
            link: "/billing/configurations/customers",
          },
          {
            key: "document-classifications",
            label: "Document Classifications",
            link: "/billing/configurations/document-classifications",
          },
        ],
      },
    ],
  },
  {
    key: "dispatch-management",
    label: "Dispatch Management",
    icon: faRoad,
    tree: [
      {
        key: "rate-management",
        label: "Rate Management",
        link: "/dispatch/rate-management",
      },
      {
        key: "configuration-files",
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: "comment-types",
            label: "Comment Types",
            link: "/dispatch/configurations/comment-types",
          },
          {
            key: "delay-codes",
            label: "Delay Codes",
            link: "/dispatch/configurations/delay-codes",
          },
          {
            key: "fleet-codes",
            label: "Fleet Codes",
            link: "/dispatch/configurations/fleet-codes",
          },
          {
            key: "locations",
            label: "Locations",
            link: "/dispatch/configurations/locations",
          },
          {
            key: "location-categories",
            label: "Location Categories",
            link: "/dispatch/configurations/location-categories",
          },
          {
            key: "routes",
            label: "Routes",
            link: "/dispatch/configurations/routes",
          },
          {
            key: "workers",
            label: "Workers",
            link: "/dispatch/configurations/workers",
          },
        ],
      },
    ],
  },
  {
    key: "shipment-management",
    label: "Shipment Management",
    icon: faTruck,
    tree: [
      {
        key: "shipments",
        label: "Shipment Management",
        link: "/shipments/management",
      },
      {
        key: "configuration-files",
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: "shipment-types",
            label: "Shipment Types",
            link: "/shipments/configurations/shipment-types",
          },
          {
            key: "formula-templates",
            label: "Formula Templates",
            link: "/shipments/configurations/formula-templates",
          },
          {
            key: "service-types",
            label: "Service Types",
            link: "/shipments/configurations/service-types",
          },
          {
            key: "commodities",
            label: "Commodities",
            link: "/shipments/configurations/commodities",
          },
          {
            key: "hazardous-materials",
            label: "Hazardous Materials",
            link: "/shipments/configurations/hazardous-materials",
          },
        ],
      },
    ],
  },
  {
    key: "equipment-management",
    label: "Equipment Management",
    icon: faScrewdriverWrench,
    tree: [
      {
        key: "maintenance",
        label: "Maintenance",
        link: "/equipment/maintenance",
      },
      {
        key: "configuration-files",
        label: "Configuration Files",
        icon: faFiles,
        tree: [
          {
            key: "equipment-types",
            label: "Equipment Types",
            link: "/equipment/configurations/equipment-types",
          },
          {
            key: "equipment-manufacturers",
            label: "Equipment Manufacturers",
            link: "/equipment/configurations/equipment-manufacturers",
          },
          {
            key: "tractors",
            label: "Tractors",
            link: "/equipment/configurations/tractors",
          },
          {
            key: "trailers",
            label: "Trailers",
            link: "/equipment/configurations/trailers",
          },
        ],
      },
    ],
  },
  {
    key: "organization-settings",
    label: "Organization Settings",
    icon: faGear,
    link: "/organization-settings",
  },
];
