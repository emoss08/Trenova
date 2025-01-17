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
              const { Dashboard } = await import("@/app/dashboard/page");
              return { Component: Dashboard };
            },
          },
          // Shipment Links
          {
            path: "/shipments/configurations/shipment-types",
            async lazy() {
              const { ShipmentTypes } = await import(
                "@/app/shipment-types/page"
              );
              return { Component: ShipmentTypes };
            },
          },
          {
            path: "/shipments/configurations/service-types",
            async lazy() {
              const { ServiceTypes } = await import("@/app/service-types/page");
              return { Component: ServiceTypes };
            },
          },
          {
            path: "/shipments/configurations/hazardous-materials",
            async lazy() {
              const { HazardousMaterials } = await import(
                "@/app/hazardous-materials/page"
              );
              return { Component: HazardousMaterials };
            },
          },
          // Billing Links
          {
            path: "/billing/client",
            async lazy() {
              const { BillingClient } = await import(
                "@/app/billing-client/page"
              );
              return { Component: BillingClient };
            },
          },
          {
            path: "/billing/configurations/charge-types",
            async lazy() {
              const { ChargeTypes } = await import("@/app/charge-types/page");
              return { Component: ChargeTypes };
            },
          },
          // Dispatch Links
          {
            path: "/dispatch/configurations/workers",
            async lazy() {
              const { Workers } = await import("@/app/workers/page");
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
              const { FleetCodes } = await import("@/app/fleet-codes/page");
              return { Component: FleetCodes };
            },
            handle: {
              crumb: "Fleet Codes",
              title: "Fleet Codes",
            },
          },
          // Equipment Links
          {
            path: "/equipment/configurations/equipment-types",
            async lazy() {
              const { EquipmentTypes } = await import(
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
              const { EquipmentManufacturers } = await import(
                "@/app/equipment-manufacturers/page"
              );
              return { Component: EquipmentManufacturers };
            },
            handle: {
              crumb: "Equipment Manufacturers",
              title: "Equipment Manufacturers",
            },
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
                  const { LoginPage } = await import("@/app/auth/login-page");
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
