/* eslint-disable prefer-const */
import { AdminLayout } from "@/components/admin-layout";
import { RootErrorBoundary } from "@/components/error-boundary";
import LoadingSkeleton from "@/components/loading";
import { MainLayout } from "@/components/main-layout";
import {
  authLoader,
  createPermissionLoader,
  protectedLoader,
} from "@/lib/loaders";
import { Resource } from "@/types/audit-entry";
import { createBrowserRouter, RouteObject } from "react-router";

const routes: RouteObject[] = [
  {
    errorElement: <RootErrorBoundary />,
    children: [
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
                loader: createPermissionLoader(Resource.BillingClient),
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
                loader: createPermissionLoader(Resource.ChargeType),
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
                loader: createPermissionLoader(Resource.Customer),
                async lazy() {
                  let { Customers } = await import("@/app/customers/page");
                  return { Component: Customers };
                },
                handle: {
                  crumb: "Customers",
                  title: "Customers",
                },
              },
              {
                path: "configurations/document-types",
                loader: createPermissionLoader(Resource.DocumentType),
                async lazy() {
                  let { DocumentTypes } = await import(
                    "@/app/document-types/page"
                  );
                  return { Component: DocumentTypes };
                },
                handle: {
                  crumb: "Document Types",
                  title: "Document Types",
                },
              },
              {
                path: "configurations/fiscal-years",
                loader: createPermissionLoader(Resource.FiscalYear),
                async lazy() {
                  let { FiscalYears } = await import("@/app/fiscal-years/page");
                  return { Component: FiscalYears };
                },
                handle: {
                  crumb: "Fiscal Years",
                  title: "Fiscal Years",
                },
              },
              {
                path: "configurations/account-types",
                loader: createPermissionLoader(Resource.AccountType),
                async lazy() {
                  let { AccountTypes } = await import(
                    "@/app/account-types/page"
                  );
                  return { Component: AccountTypes };
                },
                handle: {
                  crumb: "Account Types",
                  title: "Account Types",
                },
              },
              {
                path: "configurations/accessorial-charges",
                loader: createPermissionLoader(Resource.AccessorialCharge),
                async lazy() {
                  let { AccessorialCharges } = await import(
                    "@/app/accessorial-charges/page"
                  );
                  return { Component: AccessorialCharges };
                },
                handle: {
                  crumb: "Accessorial Charges",
                  title: "Accessorial Charges",
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
                loader: createPermissionLoader(Resource.Shipment),
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
                path: "consolidation-groups",
                loader: createPermissionLoader(Resource.Consolidation),
                async lazy() {
                  let { ConsolidationGroup } = await import(
                    "@/app/consolidation-group/page"
                  );
                  return { Component: ConsolidationGroup };
                },
                handle: {
                  crumb: "Consolidation Groups",
                  title: "Consolidation Groups",
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
                    loader: createPermissionLoader(Resource.DedicatedLane),
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
                    loader: createPermissionLoader(Resource.ShipmentType),
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
                    loader: createPermissionLoader(Resource.ServiceType),
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
                    loader: createPermissionLoader(Resource.HazardousMaterial),
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
                    loader: createPermissionLoader(Resource.Commodity),
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
                    loader: createPermissionLoader(Resource.Worker),
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
                    loader: createPermissionLoader(Resource.FleetCode),
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
                    loader: createPermissionLoader(Resource.LocationCategory),
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
                    loader: createPermissionLoader(Resource.Location),
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
                    loader: createPermissionLoader(Resource.EquipmentType),
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
                    loader: createPermissionLoader(
                      Resource.EquipmentManufacturer,
                    ),
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
                    loader: createPermissionLoader(Resource.Tractor),
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
                    loader: createPermissionLoader(Resource.Trailer),
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
                loader: createPermissionLoader(Resource.Organization),
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
                loader: createPermissionLoader(Resource.ShipmentControl),
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
                path: "dispatch-controls",
                loader: createPermissionLoader(Resource.DispatchControl),
                async lazy() {
                  let { DispatchControl } = await import(
                    "@/app/dispatch-control/page"
                  );
                  return { Component: DispatchControl };
                },
                handle: {
                  crumb: "Dispatch Controls",
                  title: "Dispatch Controls",
                },
              },
              {
                path: "distance-overrides",
                loader: createPermissionLoader(Resource.DistanceOverride),
                async lazy() {
                  let { DistanceOverrides } = await import(
                    "@/app/distance-overrides/page"
                  );
                  return { Component: DistanceOverrides };
                },
                handle: {
                  crumb: "Distance Overrides",
                  title: "Distance Overrides",
                },
              },
              {
                path: "consolidation-settings",
                loader: createPermissionLoader(Resource.ConsolidationSettings),
                async lazy() {
                  let { ConsolidationSetting } = await import(
                    "@/app/consolidation-setting/page"
                  );
                  return { Component: ConsolidationSetting };
                },
                handle: {
                  crumb: "Consolidation Settings",
                  title: "Consolidation Settings",
                },
              },
              {
                path: "billing-controls",
                loader: createPermissionLoader(Resource.BillingControl),
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
                loader: createPermissionLoader(Resource.DataRetention),
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
                loader: createPermissionLoader(Resource.ResourceEditor),
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
                loader: createPermissionLoader(Resource.PatternConfig),
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
                loader: createPermissionLoader(Resource.Integration),
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
                path: "docker",
                loader: createPermissionLoader(Resource.Docker),
                async lazy() {
                  let { DockerManagement } = await import("@/app/docker/page");
                  return { Component: DockerManagement };
                },
                handle: {
                  crumb: "Docker Management",
                  title: "Docker Management",
                },
              },
              {
                path: "users",
                loader: createPermissionLoader(Resource.User),
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
                loader: createPermissionLoader(Resource.HazmatSegregationRule),
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
                path: "hold-reasons",
                loader: createPermissionLoader(Resource.HoldReason),
                async lazy() {
                  let { HoldReasons } = await import("@/app/hold-reason/page");
                  return { Component: HoldReasons };
                },
                handle: {
                  crumb: "Hold Reasons",
                  title: "Hold Reasons",
                },
              },
              {
                path: "audit-entries",
                loader: createPermissionLoader(Resource.AuditEntry),
                async lazy() {
                  let { AuditLogs } = await import("@/app/audit-logs/page");
                  return { Component: AuditLogs };
                },
                handle: {
                  crumb: "Audit Entries",
                  title: "Audit Entries",
                },
              },
              {
                path: "email-profiles",
                loader: createPermissionLoader(Resource.EmailProfile),
                async lazy() {
                  let { EmailProfiles } = await import(
                    "@/app/email-profiles/page"
                  );
                  return { Component: EmailProfiles };
                },
                handle: {
                  crumb: "Email Profiles",
                  title: "Email Profiles",
                },
              },
              {
                path: "ai-logs",
                loader: createPermissionLoader(Resource.AILog),
                async lazy() {
                  let { AILogs } = await import("@/app/ai-logs/page");
                  return { Component: AILogs };
                },
                handle: {
                  crumb: "AI Logs",
                  title: "AI Logs",
                },
              },
              {
                path: "variables",
                loader: createPermissionLoader(Resource.Variable),
                async lazy() {
                  let { Variables } = await import("@/app/variables/page");
                  return { Component: Variables };
                },
                handle: {
                  crumb: "AI Logs",
                  title: "AI Logs",
                },
              },
            ],
          },
        ],
      },
      {
        path: "/permission-denied",
        loader: protectedLoader,
        async lazy() {
          let { PermissionDenied } = await import(
            "@/app/permission-denied/page"
          );
          return { Component: PermissionDenied };
        },
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
