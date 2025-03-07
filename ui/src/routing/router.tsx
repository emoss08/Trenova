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
            path: "/billing/configurations/customers",
            async lazy() {
              let { Customers } = await import("@/app/customers/page");
              return { Component: Customers };
            },
          },
          // Shipment Links
          {
            path: "/shipments/management",
            async lazy() {
              let { Shipment } = await import("@/app/shipment/page");
              return { Component: Shipment };
            },
          },
          {
            path: "/shipments/configurations/shipment-types",
            async lazy() {
              let { ShipmentTypes } = await import("@/app/shipment-types/page");
              return { Component: ShipmentTypes };
            },
          },
          {
            path: "/shipments/configurations/service-types",
            async lazy() {
              let { ServiceTypes } = await import("@/app/service-types/page");
              return { Component: ServiceTypes };
            },
          },
          {
            path: "/shipments/configurations/hazardous-materials",
            async lazy() {
              let { HazardousMaterials } = await import(
                "@/app/hazardous-materials/page"
              );
              return { Component: HazardousMaterials };
            },
          },
          {
            path: "/shipments/configurations/commodities",
            async lazy() {
              let { Commodities } = await import("@/app/commodities/page");
              return { Component: Commodities };
            },
          },
          // Billing Links
          {
            path: "/billing/client",
            async lazy() {
              let { BillingClient } = await import("@/app/billing-client/page");
              return { Component: BillingClient };
            },
          },
          {
            path: "/billing/configurations/charge-types",
            async lazy() {
              let { ChargeTypes } = await import("@/app/charge-types/page");
              return { Component: ChargeTypes };
            },
            handle: {
              crumb: "Charge Types",
              title: "Charge Types",
            },
          },
          // Dispatch Links
          {
            path: "/dispatch/configurations/workers",
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
            path: "/dispatch/configurations/fleet-codes",
            async lazy() {
              let { FleetCodes } = await import("@/app/fleet-codes/page");
              return { Component: FleetCodes };
            },
            handle: {
              crumb: "Fleet Codes",
              title: "Fleet Codes",
            },
          },
          {
            path: "/dispatch/configurations/location-categories",
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
          // Location Links
          {
            path: "/dispatch/configurations/locations",
            async lazy() {
              let { Locations } = await import("@/app/locations/page");
              return { Component: Locations };
            },
            handle: {
              crumb: "Locations",
              title: "Locations",
            },
          },
          // Equipment Links
          {
            path: "/equipment/configurations/equipment-types",
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
            path: "/equipment/configurations/equipment-manufacturers",
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
            path: "/equipment/configurations/tractors",
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
            path: "/equipment/configurations/trailers",
            async lazy() {
              let { Trailers } = await import("@/app/trailers/page");
              return { Component: Trailers };
            },
            handle: {
              crumb: "Trailers",
              title: "Trailers",
            },
          },

          // Organization Setting Links
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
              },
              {
                path: "system-logs",
                async lazy() {
                  let { LogReader } = await import("@/app/logreader/page");
                  return { Component: LogReader };
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
              },
              {
                path: "data-retention",
                async lazy() {
                  let { DataRetention } = await import(
                    "@/app/data-retention/page"
                  );
                  return { Component: DataRetention };
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

