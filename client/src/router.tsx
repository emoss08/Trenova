import { RouteErrorBoundary } from "@/components/error-boundary";
import {
  combineLoaders,
  createAdminOnlyLoader,
  createPermissionLoader,
  createPlatformAdminLoader,
} from "@/lib/route-permission";
import { AppLayout } from "@/routes/app-layout";
import { RootLayout } from "@/routes/root-layout";
import { useAuthStore } from "@/stores/auth-store";
import { Operation, Resource } from "@/types/permission";
import {
  createBrowserRouter,
  redirect,
  type LoaderFunction,
  type RouteObject,
} from "react-router";
import LoadingSkeleton from "./components/loading-skeleton";
import { AdminLayout } from "./routes/admin-layout";

const protectedLoader: LoaderFunction = async () => {
  const { checkAuth } = useAuthStore.getState();
  const isAuthenticated = await checkAuth();

  if (!isAuthenticated) {
    return redirect("/login");
  }

  return null;
};

const guestLoader: LoaderFunction = async () => {
  const { checkAuth } = useAuthStore.getState();

  const isAuthenticated = await checkAuth();
  if (isAuthenticated) {
    return redirect("/");
  }

  return null;
};

const routes: RouteObject[] = [
  {
    element: <RootLayout />,
    errorElement: <RouteErrorBoundary />,
    HydrateFallback: LoadingSkeleton,
    children: [
      {
        element: <AppLayout />,
        loader: protectedLoader,
        children: [
          {
            path: "/",
            async lazy() {
              const { Home } = await import("@/routes/home");
              return { Component: Home };
            },
          },
          {
            path: "/shipment-management/shipments",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Shipment),
            ),
            async lazy() {
              const { ShipmentsPage } = await import("@/routes/shipment/page");
              return { Component: ShipmentsPage };
            },
          },
          {
            path: "/shipment-management/shipments/import",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Shipment, Operation.Create),
            ),
            async lazy() {
              const { ShipmentImportPage } = await import(
                "@/routes/shipment/import-page"
              );
              return { Component: ShipmentImportPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/shipment-types",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.ShipmentType),
            ),
            async lazy() {
              const { ShipmentTypesPage } = await import(
                "@/routes/shipment-type/page"
              );
              return { Component: ShipmentTypesPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/service-types",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.ServiceType),
            ),
            async lazy() {
              const { ServiceTypesPage } = await import(
                "@/routes/service-type/page"
              );
              return { Component: ServiceTypesPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/hazardous-materials",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.HazardousMaterial),
            ),
            async lazy() {
              const { HazardousMaterialsPage } = await import(
                "@/routes/hazardous-material/page"
              );
              return { Component: HazardousMaterialsPage };
            },
          },
          {
            path: "/shipment-management/configuration-files/commodities",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Commodity),
            ),
            async lazy() {
              const { CommoditiesPage } = await import(
                "@/routes/commodity/page"
              );
              return { Component: CommoditiesPage };
            },
          },
          {
            path: "/billing/queue",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.BillingQueue),
            ),
            async lazy() {
              const { BillingQueuePage } = await import(
                "@/routes/billing-queue/page"
              );
              return { Component: BillingQueuePage };
            },
          },
          {
            path: "/billing/invoices",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Invoice),
            ),
            async lazy() {
              const { PlaceholderPage } = await import(
                "@/routes/placeholder-page"
              );
              return { Component: PlaceholderPage };
            },
          },
          {
            path: "/billing/configuration-files/charge-types",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.ChargeType),
            ),
            async lazy() {
              const { PlaceholderPage } = await import(
                "@/routes/placeholder-page"
              );
              return { Component: PlaceholderPage };
            },
          },
          {
            path: "/billing/configuration-files/accessorial-charges",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccessorialCharge),
            ),
            async lazy() {
              const { AccessorialChargesPage } = await import(
                "@/routes/accessorial-charge/page"
              );
              return { Component: AccessorialChargesPage };
            },
          },
          {
            path: "/billing/configuration-files/customers",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Customer),
            ),
            async lazy() {
              const { CustomersPage } = await import("@/routes/customer/page");
              return { Component: CustomersPage };
            },
          },
          {
            path: "/billing/configuration-files/document-types",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.DocumentType),
            ),
            async lazy() {
              const { DocumentTypesPage } = await import(
                "@/routes/document-type/page"
              );
              return { Component: DocumentTypesPage };
            },
          },
          {
            path: "/billing/configuration-files/document-packet-rules",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.DocumentType),
            ),
            async lazy() {
              const { DocumentPacketRulesPage } = await import(
                "@/routes/document-packet-rule/page"
              );
              return { Component: DocumentPacketRulesPage };
            },
          },

          {
            path: "/accounting/configuration-files/account-types",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.AccountType),
            ),
            async lazy() {
              const { AccountTypesPage } = await import(
                "@/routes/account-type/page"
              );
              return { Component: AccountTypesPage };
            },
          },
          {
            path: "/billing/configuration-files/formula-templates",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.FormulaTemplate),
            ),
            async lazy() {
              const { FormulaTemplatesPage } = await import(
                "@/routes/formula-template/page"
              );
              return { Component: FormulaTemplatesPage };
            },
          },
          {
            path: "/equipment/tractors",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Tractor),
            ),
            async lazy() {
              const { TractorsPage } = await import("@/routes/tractor/page");
              return { Component: TractorsPage };
            },
          },
          {
            path: "/equipment/trailers",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Trailer),
            ),
            async lazy() {
              const { TrailersPage } = await import("@/routes/trailer/page");
              return { Component: TrailersPage };
            },
          },
          {
            path: "/equipment/configuration-files/equipment-types",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.EquipmentType),
            ),
            async lazy() {
              const { EquipmentTypesPage } = await import(
                "@/routes/equipment-type/page"
              );
              return { Component: EquipmentTypesPage };
            },
          },

          {
            path: "/equipment/configuration-files/equipment-manufacturers",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.EquipmentManufacturer),
            ),
            async lazy() {
              const { EquipmentManufacturersPage } = await import(
                "@/routes/equipment-manufacturer/page"
              );
              return { Component: EquipmentManufacturersPage };
            },
          },
          {
            path: "/dispatch/configuration-files/location-categories",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.LocationCategory),
            ),
            async lazy() {
              const { LocationCategoriesPage } = await import(
                "@/routes/location-category/page"
              );
              return { Component: LocationCategoriesPage };
            },
          },
          {
            path: "/dispatch/configuration-files/fleet-codes",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.FleetCode, Operation.Read),
            ),
            async lazy() {
              const { FleetCodesPage } = await import(
                "@/routes/fleet-code/page"
              );
              return { Component: FleetCodesPage };
            },
          },
          {
            path: "/dispatch/locations",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Location),
            ),
            async lazy() {
              const { LocationsPage } = await import("@/routes/location/page");
              return { Component: LocationsPage };
            },
          },
          {
            path: "/dispatch/workers",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.Worker),
            ),
            async lazy() {
              const { WorkersPage } = await import("@/routes/worker/page");
              return { Component: WorkersPage };
            },
          },
          {
            path: "/accounting/configuration-files/fiscal-years",
            loader: combineLoaders(
              protectedLoader,
              createPermissionLoader(Resource.FiscalYear),
            ),
            async lazy() {
              const { FiscalYearsPage } = await import(
                "@/routes/fiscal-year/page"
              );
              return { Component: FiscalYearsPage };
            },
          },

          {
            path: "admin",
            Component: AdminLayout,
            HydrateFallback: LoadingSkeleton,
            loader: combineLoaders(protectedLoader, createAdminOnlyLoader()),
            children: [
              {
                path: "billing-controls",
                async lazy() {
                  const { BillingControlPage } = await import(
                    "@/routes/billing-control/page"
                  );
                  return { Component: BillingControlPage };
                },
              },
              {
                path: "organization-settings",
                loader: createPermissionLoader(
                  Resource.Organization,
                  Operation.Read,
                ),
                async lazy() {
                  const { OrganizationSettingsPage } = await import(
                    "@/routes/admin/organization-settings/page"
                  );
                  return { Component: OrganizationSettingsPage };
                },
              },
              {
                path: "accounting-control",
                loader: createPermissionLoader(Resource.AccountingControl),
                async lazy() {
                  const { AccountingControlPage } = await import(
                    "@/routes/accounting-control/page"
                  );
                  return { Component: AccountingControlPage };
                },
              },
              {
                path: "table-change-alerts",
                loader: createPlatformAdminLoader(),
                async lazy() {
                  const { TableChangeAlertPage } = await import(
                    "@/routes/table-change-alert/page"
                  );
                  return { Component: TableChangeAlertPage };
                },
              },
              {
                path: "data-entry-controls",
                loader: createPermissionLoader(
                  Resource.DataEntryControl,
                  Operation.Read,
                ),
                async lazy() {
                  const { DataEntryControlPage } = await import(
                    "@/routes/data-entry-control/page"
                  );
                  return { Component: DataEntryControlPage };
                },
              },
              {
                path: "dispatch-controls",
                loader: createPermissionLoader(
                  Resource.DispatchControl,
                  Operation.Read,
                ),
                async lazy() {
                  const { DispatchControlPage } = await import(
                    "@/routes/dispatch-control/page"
                  );
                  return { Component: DispatchControlPage };
                },
              },
              {
                path: "distance-overrides",
                loader: createPermissionLoader(Resource.DistanceOverride),
                async lazy() {
                  const { DistanceOverridesPage } = await import(
                    "@/routes/distance-override/page"
                  );
                  return { Component: DistanceOverridesPage };
                },
              },
              {
                path: "shipment-controls",
                loader: createPermissionLoader(
                  Resource.ShipmentControl,
                  Operation.Read,
                ),
                async lazy() {
                  const { ShipmentControlPage } = await import(
                    "@/routes/shipment-control/page"
                  );
                  return { Component: ShipmentControlPage };
                },
              },
              {
                path: "document-intelligence",
                loader: createPermissionLoader(
                  Resource.DocumentControl,
                  Operation.Read,
                ),
                async lazy() {
                  const { DocumentIntelligencePage } = await import(
                    "@/routes/admin/document-intelligence/page"
                  );
                  return { Component: DocumentIntelligencePage };
                },
              },
              {
                path: "document-parsing-rules",
                loader: createPermissionLoader(
                  Resource.DocumentParsingRule,
                  Operation.Read,
                ),
                async lazy() {
                  const { DocumentParsingRulesPage } = await import(
                    "@/routes/admin/document-parsing-rules/page"
                  );
                  return { Component: DocumentParsingRulesPage };
                },
              },
              {
                path: "sequence-configs",
                loader: createPermissionLoader(
                  Resource.SequenceConfig,
                  Operation.Read,
                ),
                async lazy() {
                  const { SequenceConfigPage } = await import(
                    "@/routes/admin/sequence-config/page"
                  );
                  return { Component: SequenceConfigPage };
                },
              },
              {
                path: "hold-reasons",
                loader: combineLoaders(
                  protectedLoader,
                  createPermissionLoader(Resource.HoldReason),
                ),
                async lazy() {
                  const { HoldReasonsPage } = await import(
                    "@/routes/hold-reason/page"
                  );
                  return { Component: HoldReasonsPage };
                },
              },
              {
                path: "hazmat-segregation-rules",
                loader: createPermissionLoader(Resource.HazmatSegregationRule),
                async lazy() {
                  const { HazmatSegregationRulesPage } = await import(
                    "@/routes/hazmat-segregation-rule/page"
                  );
                  return { Component: HazmatSegregationRulesPage };
                },
              },
              {
                path: "roles",
                loader: createPermissionLoader(Resource.Role, Operation.Read),
                async lazy() {
                  const { RolesPage } = await import(
                    "@/routes/admin/roles/page"
                  );
                  return { Component: RolesPage };
                },
              },
              {
                path: "roles/new",
                loader: createPermissionLoader(Resource.Role, Operation.Create),
                async lazy() {
                  const { RoleCreatePage } = await import(
                    "@/routes/admin/roles/new/page"
                  );
                  return { Component: RoleCreatePage };
                },
              },
              {
                path: "roles/:id/edit",
                loader: createPermissionLoader(Resource.Role, Operation.Update),
                async lazy() {
                  const { RoleEditPage } = await import(
                    "@/routes/admin/roles/[id]/edit/page"
                  );
                  return { Component: RoleEditPage };
                },
              },
              {
                path: "users",
                loader: createPermissionLoader(Resource.User, Operation.Read),
                async lazy() {
                  const { UsersPage } = await import(
                    "@/routes/admin/users/page"
                  );
                  return { Component: UsersPage };
                },
              },
              {
                path: "audit-logs",
                loader: createPermissionLoader(
                  Resource.AuditLog,
                  Operation.Read,
                ),
                async lazy() {
                  const { AuditLogsPage } = await import(
                    "@/routes/admin/audit-logs/page"
                  );
                  return { Component: AuditLogsPage };
                },
              },
              {
                path: "database-sessions",
                loader: createPlatformAdminLoader(),
                async lazy() {
                  const { DatabaseSessionsPage } = await import(
                    "@/routes/admin/database-sessions/page"
                  );
                  return { Component: DatabaseSessionsPage };
                },
              },
              {
                path: "integrations",
                loader: createPermissionLoader(
                  Resource.Integration,
                  Operation.Read,
                ),
                async lazy() {
                  const { IntegrationsPage } = await import(
                    "@/routes/admin/integrations/page"
                  );
                  return { Component: IntegrationsPage };
                },
              },
              {
                path: "api-keys",
                loader: createPermissionLoader(
                  Resource.Integration,
                  Operation.Read,
                ),
                async lazy() {
                  const { APIKeysPage } = await import(
                    "@/routes/admin/api-keys/page"
                  );
                  return { Component: APIKeysPage };
                },
              },
              {
                path: "document-operations",
                loader: createPlatformAdminLoader(),
                async lazy() {
                  const { DocumentOperationsPage } = await import(
                    "@/routes/admin/document-operations/page"
                  );
                  return { Component: DocumentOperationsPage };
                },
              },
              {
                path: "custom-fields",
                loader: createPermissionLoader(
                  Resource.CustomFieldDefinition,
                  Operation.Read,
                ),
                async lazy() {
                  const { CustomFieldDefinitionsPage } = await import(
                    "@/routes/admin/custom-fields/page"
                  );
                  return { Component: CustomFieldDefinitionsPage };
                },
              },
            ],
          },
        ],
      },
      {
        loader: guestLoader,
        children: [
          {
            path: "/login",
            async lazy() {
              const { AuthPage } = await import("@/routes/auth/page");
              return { Component: AuthPage };
            },
          },
          {
            path: "/login/:orgSlug",
            async lazy() {
              const { AuthPage } = await import("@/routes/auth/page");
              return { Component: AuthPage };
            },
          },
        ],
      },
    ],
  },
];

export const router = createBrowserRouter(routes);
