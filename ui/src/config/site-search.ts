import { CommandGroupInfo } from "@/types/nav-links";
import { type SiteSearchQuickOptionProps } from "@/types/search";
import {
  faBoxes,
  faContainerStorage,
  faDashboard,
  faGrid5,
  faTruck,
  faUserHelmetSafety,
  faUsers,
} from "@fortawesome/pro-regular-svg-icons";

export const tabConfig: Record<
  string,
  {
    icon: any;
    label: string;
    filters: string[];
    color: string;
  }
> = {
  all: {
    icon: faGrid5,
    label: "All",
    filters: [],
    color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200",
  },
  shipments: {
    icon: faBoxes,
    label: "Shipments",
    filters: ["status", "priority", "date", "customer"],
    color: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  },
  workers: {
    icon: faUserHelmetSafety,
    label: "Workers",
    filters: ["status", "availability", "type", "license"],
    color:
      "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  },
  equipment: {
    icon: faTruck,
    label: "Tractors",
    filters: ["status", "type", "maintenance", "ownership"],
    color:
      "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300",
  },
};

// Quick actions are shown when the search query is empty
export const quickActions: Record<string, SiteSearchQuickOptionProps> = {
  // TODO(Wolfred): Add some point we need to change these based on user permissions.
  createShipment: {
    icon: faBoxes,
    label: "Create Shipment",
    description: "Add a new shipment from scratch to the system",
    link: "/shipments/management?modal=create",
  },
  createWorker: {
    icon: faUserHelmetSafety,
    label: "Create Worker",
    description: "Add a new worker to the system",
    link: "/dispatch/configurations/workers?modal=create",
  },
  createTractor: {
    icon: faTruck,
    label: "Create Tractor",
    description: "Add a new tractor to the system",
    link: "/equipment/configurations/tractors?modal=create",
  },
  createTrailer: {
    icon: faContainerStorage,
    label: "Create Trailer",
    description: "Add a new trailer to the system",
    link: "/equipment/configurations/trailers?modal=create",
  },
};

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
    id: "billing-management",
    label: "Billing Management",
    routes: [
      {
        id: "customers",
        link: "/billing/configurations/customers",
        label: "Customers",
        icon: faUsers,
      },
    ],
  },
  {
    id: "shipment-management",
    label: "Shipment Management",
    routes: [
      {
        id: "shipments",
        link: "/shipments/management",
        label: "Shipments",
        icon: faTruck,
      },

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
      {
        id: "trailers",
        link: "/equipment/configurations/trailers",
        label: "Trailers",
        icon: faTruck,
      },
    ],
  },
];
