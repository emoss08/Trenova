/* eslint-disable prefer-const */
import { AdminLayout } from "@/components/admin-layout";
import { RootErrorBoundary } from "@/components/error-boundary";
import LoadingSkeleton from "@/components/loading";
import { MainLayout } from "@/components/main-layout";
import { authLoader, protectedLoader } from "@/lib/loaders";
import { createBrowserRouter, RouteObject } from "react-router";

const routes: RouteObject[] = [
  {
    errorElement: <RootErrorBoundary />,
    children: [
      // Protected routes with MainLayout
      {
        Component: MainLayout,
        HydrateFallback: LoadingSkeleton,
        loader: protectedLoader,
        children: [
          {
            path: "/",
            index: true,
            async lazy() {
              let { Dashboard } = await import("@/app/dashboard/page");
              return { Component: Dashboard };
            },
          },
          // Billing Links
          {
            path: "/billing",
            handle: {
              crumb: "Billing",
              title: "Billing",
            },
            children: [
              {
                path: "client",
                async lazy() {
                  let { BillingClient } = await import(
                    "@/app/billing-client/page"
                  );
                  return { Component: BillingClient };
                },
                handle: {
                  crumb: "Billing Client",
                  title: "Billing Client",
                },
              },
              {
                path: "configurations/charge-types",
                async lazy() {
                  let { ChargeTypes } = await import("@/app/charge-types/page");
                  return { Component: ChargeTypes };
                },
                handle: {
                  crumb: "Charge Types",
                  title: "Charge Types",
                },
              },
              {
                path: "configurations/customers",
                async lazy() {
                  let { Customers } = await import("@/app/customers/page");
                  return { Component: Customers };
                },
              },
              {
                path: "configurations/document-types",
                async lazy() {
                  let { DocumentTypes } = await import(
                    "@/app/document-types/page"
                  );
                  return { Component: DocumentTypes };
                },
              },
              {
                path: "configurations/accessorial-charges",
                async lazy() {
                  let { AccessorialCharges } = await import(
                    "@/app/accessorial-charges/page"
                  );
                  return { Component: AccessorialCharges };
                },
              },
            ],
          },
          // Shipment Links
          {
            path: "shipments",
            handle: {
              crumb: "Shipment Management",
              title: "Shipment Management",
            },
            children: [
              {
                path: "management",
                async lazy() {
                  let { Shipment } = await import("@/app/shipment/page");
                  return { Component: Shipment };
                },
                handle: {
                  crumb: "Shipment Management",
                  title: "Shipment Management",
                },
              },

              {
                path: "configurations",
                handle: {
                  crumb: "Configurations Files",
                  title: "Configurations Files",
                },
                children: [
                  {
                    path: "dedicated-lanes",
                    async lazy() {
                      let { DedicatedLane } = await import(
                        "@/app/dedicated-lane/page"
                      );
                      return { Component: DedicatedLane };
                    },
                    handle: {
                      crumb: "Dedicated Lanes",
                      title: "Dedicated Lanes",
                    },
                  },
                  {
                    path: "shipment-types",
                    async lazy() {
                      let { ShipmentTypes } = await import(
                        "@/app/shipment-types/page"
                      );
                      return { Component: ShipmentTypes };
                    },
                    handle: {
                      crumb: "Shipment Types",
                      title: "Shipment Types",
                    },
                  },
                  {
                    path: "service-types",
                    async lazy() {
                      let { ServiceTypes } = await import(
                        "@/app/service-types/page"
                      );
                      return { Component: ServiceTypes };
                    },
                    handle: {
                      crumb: "Service Types",
                      title: "Service Types",
                    },
                  },
                  {
                    path: "hazardous-materials",
                    async lazy() {
                      let { HazardousMaterials } = await import(
                        "@/app/hazardous-materials/page"
                      );
                      return { Component: HazardousMaterials };
                    },
                    handle: {
                      crumb: "Hazardous Materials",
                      title: "Hazardous Materials",
                    },
                  },
                  {
                    path: "commodities",
                    async lazy() {
                      let { Commodities } = await import(
                        "@/app/commodities/page"
                      );
                      return { Component: Commodities };
                    },
                    handle: {
                      crumb: "Commodities",
                      title: "Commodities",
                    },
                  },
                ],
              },
            ],
          },
          {
            path: "dispatch",
            handle: {
              crumb: "Dispatch Management",
              title: "Dispatch Management",
            },
            children: [
              {
                path: "configurations",
                handle: {
                  crumb: "Configurations Files",
                  title: "Configurations Files",
                },
                children: [
                  {
                    path: "workers",
                    async lazy() {
                      let { Workers } = await import("@/app/workers/page");
                      return { Component: Workers };
                    },
                    handle: {
                      crumb: "Workers",
                      title: "Workers",
                    },
                  },
                  {
                    path: "fleet-codes",
                    async lazy() {
                      let { FleetCodes } = await import(
                        "@/app/fleet-codes/page"
                      );
                      return { Component: FleetCodes };
                    },
                    handle: {
                      crumb: "Fleet Codes",
                      title: "Fleet Codes",
                    },
                  },
                  {
                    path: "location-categories",
                    async lazy() {
                      let { LocationCategories } = await import(
                        "@/app/location-categories/page"
                      );
                      return { Component: LocationCategories };
                    },
                    handle: {
                      crumb: "Location Categories",
                      title: "Location Categories",
                    },
                  },
                  {
                    path: "locations",
                    async lazy() {
                      let { Locations } = await import("@/app/locations/page");
                      return { Component: Locations };
                    },
                    handle: {
                      crumb: "Locations",
                      title: "Locations",
                    },
                  },
                ],
              },
            ],
          },
          {
            path: "equipment",
            handle: {
              crumb: "Equipment Management",
              title: "Equipment Management",
            },
            children: [
              {
                path: "configurations",
                handle: {
                  crumb: "Configurations Files",
                  title: "Configurations Files",
                },
                children: [
                  {
                    path: "equipment-types",
                    async lazy() {
                      let { EquipmentTypes } = await import(
                        "@/app/equipment-types/page"
                      );
                      return { Component: EquipmentTypes };
                    },
                    handle: {
                      crumb: "Equipment Types",
                      title: "Equipment Types",
                    },
                  },
                  {
                    path: "equipment-manufacturers",
                    async lazy() {
                      let { EquipmentManufacturers } = await import(
                        "@/app/equipment-manufacturers/page"
                      );
                      return { Component: EquipmentManufacturers };
                    },
                    handle: {
                      crumb: "Equipment Manufacturers",
                      title: "Equipment Manufacturers",
                    },
                  },
                  {
                    path: "tractors",
                    async lazy() {
                      let { Tractor } = await import("@/app/tractor/page");
                      return { Component: Tractor };
                    },
                    handle: {
                      crumb: "Tractors",
                      title: "Tractors",
                    },
                  },
                  {
                    path: "trailers",
                    async lazy() {
                      let { Trailers } = await import("@/app/trailers/page");
                      return { Component: Trailers };
                    },
                    handle: {
                      crumb: "Trailers",
                      title: "Trailers",
                    },
                  },
                ],
              },
            ],
          },
          {
            path: "/organization/",
            Component: AdminLayout,
            HydrateFallback: LoadingSkeleton,
            loader: protectedLoader,
            handle: {
              crumb: "Organization Settings",
              title: "Organization Settings",
            },
            children: [
              {
                path: "settings",
                async lazy() {
                  let { OrganizationSettings } = await import(
                    "@/app/organization/page"
                  );
                  return { Component: OrganizationSettings };
                },
                handle: {
                  crumb: "Organization Settings",
                  title: "Organization Settings",
                },
              },
              {
                path: "shipment-controls",
                async lazy() {
                  let { ShipmentControl } = await import(
                    "@/app/shipment-control/page"
                  );
                  return { Component: ShipmentControl };
                },
                handle: {
                  crumb: "Shipment Controls",
                  title: "Shipment Controls",
                },
              },
              {
                path: "billing-controls",
                async lazy() {
                  let { BillingControl } = await import(
                    "@/app/billing-control/page"
                  );
                  return { Component: BillingControl };
                },
                handle: {
                  crumb: "Billing Controls",
                  title: "Billing Controls",
                },
              },
              {
                path: "data-retention",
                async lazy() {
                  let { DataRetention } = await import(
                    "@/app/data-retention/page"
                  );
                  return { Component: DataRetention };
                },
                handle: {
                  crumb: "Data Retention",
                  title: "Data Retention",
                },
              },
              {
                path: "resource-editor",
                async lazy() {
                  let { ResourceEditor } = await import(
                    "@/app/resource-editor/page"
                  );
                  return { Component: ResourceEditor };
                },
                handle: {
                  crumb: "Resource Editor",
                  title: "Resource Editor",
                },
              },
              {
                path: "pattern-config",
                async lazy() {
                  let { PatternConfig } = await import(
                    "@/app/pattern-config/page"
                  );
                  return { Component: PatternConfig };
                },
                handle: {
                  crumb: "Pattern Detection",
                  title: "Pattern Detection",
                },
              },
              {
                path: "integrations",
                async lazy() {
                  let { IntegrationsPage } = await import(
                    "@/app/integrations/page"
                  );
                  return { Component: IntegrationsPage };
                },
                handle: {
                  crumb: "Apps & Integrations",
                  title: "Apps & Integrations",
                },
              },
              {
                path: "users",
                async lazy() {
                  let { Users } = await import("@/app/users/page");
                  return { Component: Users };
                },
                handle: {
                  crumb: "Users & Roles",
                  title: "Users & Roles",
                },
              },
              {
                path: "hazmat-segregation-rules",
                async lazy() {
                  let { HazmatSegregationRules } = await import(
                    "@/app/hazmat-segregation-rules/page"
                  );
                  return { Component: HazmatSegregationRules };
                },
                handle: {
                  crumb: "Hazmat Segregation Rules",
                  title: "Hazmat Segregation Rules",
                },
              },
              {
                path: "audit-entries",
                async lazy() {
                  let { AuditLogs } = await import("@/app/audit-logs/page");
                  return { Component: AuditLogs };
                },
                handle: {
                  crumb: "Audit Entries",
                  title: "Audit Entries",
                },
              },
            ],
          },
        ],
      },
      // Auth routes with AuthLayout
      {
        loader: authLoader,
        children: [
          {
            path: "auth",
            children: [
              {
                index: true,
                async lazy() {
                  let { LoginPage } = await import("@/app/auth/login-page");
                  return { Component: LoginPage };
                },
              },
            ],
          },
        ],
      },
    ],
  },
];

const router = createBrowserRouter(routes);

export { router, routes };

