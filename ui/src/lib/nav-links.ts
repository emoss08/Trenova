import { CommandGroupInfo, type routeInfo } from "@/types/nav-links";
import {
  faDashboard,
  faFiles,
  faGear,
  faRoad,
  faScrewdriverWrench,
  faTruck,
  faUsers,
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
        label: "Shipment Entry",
        link: "/shipments/entry",
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

export const commandRoutes: CommandGroupInfo[] = [
  {
    id: "dashboard",
    label: "Dashboard",
    routes: [
      {
        id: "main-dashboard",
        link: "/",
        label: "Dashboard",
        icon: faDashboard,
      },
    ],
  },
  {
    id: "shipment-management",
    label: "Shipment Management",
    routes: [
      {
        id: "shipment-types",
        link: "/shipments/configurations/shipment-types",
        label: "Shipment Types",
        icon: faUsers,
      },
      {
        id: "service-types",
        link: "/shipments/configurations/service-types",
        label: "Service Types",
        icon: faTruck,
      },
      {
        id: "hazardous-materials",
        link: "/shipments/configurations/hazardous-materials",
        label: "Hazardous Materials",
        icon: faTruck,
      },
      {
        id: "commodities",
        link: "/shipments/configurations/commodities",
        label: "Commodities",
        icon: faTruck,
      },
    ],
  },
  {
    id: "dispatch-management",
    label: "Dispatch Management",
    routes: [
      {
        id: "workers",
        link: "/dispatch/configurations/workers",
        label: "Workers",
        icon: faUsers,
      },
      {
        id: "fleet-codes",
        link: "/dispatch/configurations/fleet-codes",
        label: "Fleet Codes",
        icon: faTruck,
      },
      {
        id: "location-categories",
        link: "/dispatch/configurations/location-categories",
        label: "Location Categories",
        icon: faTruck,
      },
      {
        id: "locations",
        link: "/dispatch/configurations/locations",
        label: "Locations",
        icon: faTruck,
      },
    ],
  },
  {
    id: "equipment-management",
    label: "Equipment Management",
    routes: [
      {
        id: "equipment-types",
        link: "/equipment/configurations/equipment-types",
        label: "Equipment Types",
        icon: faTruck,
      },
      {
        id: "equipment-manufacturers",
        link: "/equipment/configurations/equipment-manufacturers",
        label: "Equipment Manufacturers",
        icon: faTruck,
      },
      {
        id: "tractors",
        link: "/equipment/configurations/tractors",
        label: "Tractors",
        icon: faTruck,
      },
    ],
  },
];
