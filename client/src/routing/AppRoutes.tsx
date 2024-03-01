/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { lazy } from "react";
import { RouteObject } from "react-router-dom";

const HomePage = lazy(() => import("../pages"));
const LoginPage = lazy(() => import("../pages/login-page"));
const ErrorPage = lazy(() => import("../pages/error-page"));
const DivisionCodesPage = lazy(
  () => import("../pages/accounting/DivisionCodes"),
);
const LocationCategoryPage = lazy(
  () => import("../pages/dispatch/LocationCategories"),
);
const RevenueCodesPage = lazy(() => import("../pages/accounting/RevenueCodes"));
const GLAccountsPage = lazy(() => import("../pages/accounting/GLAccounts"));
const UserSettingsPage = lazy(() => import("../pages/users/UserSettings"));
const UserPreferencesPage = lazy(
  () => import("../pages/users/UserPreferences"),
);
const AddShipmentPage = lazy(() => import("@/pages/shipment/AddShipment"));

/* Admin Pages */
const FeatureManagementPage = lazy(
  () => import("../pages/admin/FeatureManagement"),
);
const AccountingControlPage = lazy(
  () => import("../pages/admin/control-files/AccountingControl"),
);
const BillingControlPage = lazy(
  () => import("../pages/admin/control-files/BillingControl"),
);
const InvoiceControlPage = lazy(
  () => import("../pages/admin/control-files/InvoiceControl"),
);
const DispatchControlPage = lazy(
  () => import("../pages/admin/control-files/DispatchControl"),
);
const ShipmentControlPage = lazy(
  () => import("../pages/admin/control-files/ShipmentControl"),
);
const RouteControlPage = lazy(
  () => import("../pages/admin/control-files/RouteControl"),
);
const FeasibilityControlPage = lazy(
  () => import("../pages/admin/control-files/FeasibilityControl"),
);
const EmailControlPage = lazy(
  () => import("../pages/admin/control-files/EmailControl"),
);
const EmailProfilePage = lazy(() => import("../pages/admin/EmailProfiles"));
const GoogleAPIPage = lazy(() => import("../pages/admin/GoogleAPI"));
const TableChangeAlertPage = lazy(
  () => import("../pages/admin/TableChangeAlerts"),
);
const DataRetentionPage = lazy(() => import("../pages/admin/DataRetention"));

// Commodity Pages
const CommodityPage = lazy(() => import("../pages/commodities/Commodity"));
const HazardousMaterialPage = lazy(
  () => import("../pages/commodities/HazardousMaterial"),
);

// Other Pages
const ResetPasswordPage = lazy(() => import("../pages/reset-password-page"));
const ChargeTypePage = lazy(() => import("../pages/billing/ChargeTypes"));
const AccessorialChargePage = lazy(
  () => import("../pages/billing/AccessorialCharges"),
);
const CustomerPage = lazy(() => import("../pages/customer/Customer"));
const WorkerPage = lazy(() => import("../pages/worker/Worker"));
const DelayCodePage = lazy(() => import("../pages/dispatch/DelayCodes"));
const FleetCodePage = lazy(() => import("../pages/dispatch/FleetCode"));
const CommentTypePage = lazy(() => import("../pages/dispatch/CommentType"));
const ShipmentManagementPage = lazy(() => import("../pages/shipment/Shipment"));
const EquipmentTypePage = lazy(
  () => import("../pages/equipment/EquipmentType"),
);
const EquipmentManufacturerPage = lazy(
  () => import("../pages/equipment/EquipmentManufacturer"),
);
const ServiceTypePage = lazy(() => import("../pages/shipment/ServiceType"));
const QualifierCodePage = lazy(() => import("../pages/stop/QualifierCode"));
const ReasonCodePage = lazy(() => import("../pages/shipment/ReasonCode"));
const ShipmentTypePage = lazy(() => import("../pages/shipment/ShipmentType"));
const LocationPage = lazy(() => import("../pages/dispatch/Location"));
const TrailerPage = lazy(() => import("../pages/equipment/Trailer"));
const TractorPage = lazy(() => import("../pages/equipment/Tractor"));
const DocumentClassPage = lazy(
  () => import("../pages/billing/DocumentClassification"),
);
const AdminPage = lazy(() => import("../pages/admin/Dashboard"));

export type RouteObjectWithPermission = RouteObject & {
  key?: string;
  title: string;
  group: string;
  subMenu?: string;
  path: string;
  description?: string;
  excludeFromMenu?: boolean;
  permission?: string;

  /**
   * If true, the route is accessible without authentication
   */
  isPublic: boolean;
};

export const routes: RouteObjectWithPermission[] = [
  {
    title: "Home",
    group: "main",
    path: "/",
    description: "Get to the main page",
    element: <HomePage />,
    isPublic: false,
  },
  // Authentication Pages
  {
    title: "Login",
    group: "auth",
    path: "/login",
    element: <LoginPage />,
    excludeFromMenu: true,
    isPublic: true,
  },
  {
    title: "Reset Password",
    group: "auth",
    path: "/reset-password",
    element: <ResetPasswordPage />,
    excludeFromMenu: true,
    isPublic: true,
  },
  // User Pages
  {
    title: "User Settings",
    group: "user",
    path: "/account/settings/",
    element: <UserSettingsPage />,
    excludeFromMenu: true,
    isPublic: false,
  },
  {
    title: "User Preferences",
    group: "user",
    path: "/account/settings/preferences",
    element: <UserPreferencesPage />,
    excludeFromMenu: true,
    isPublic: false,
  },
  // Accounting Pages
  {
    title: "Division Codes",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/division-codes",
    description: "Manage division codes",
    element: <DivisionCodesPage />,
    permission: "view_divisioncode",
    isPublic: false,
  },
  {
    title: "Revenue Codes",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/revenue-codes",
    description: "Manage revenue codes",
    element: <RevenueCodesPage />,
    permission: "view_revenuecode",
    isPublic: false,
  },
  {
    title: "General Ledger Accounts",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/gl-accounts",
    description: "Manage general ledger accounts",
    element: <GLAccountsPage />,
    permission: "view_generalledgeraccount",
    isPublic: false,
  },
  // Billing Pages
  {
    title: "Charge Types",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/charge-types",
    description: "Manage charge types",
    element: <ChargeTypePage />,
    permission: "view_chargetype",
    isPublic: false,
  },
  {
    title: "Document Classifications",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/document-classes",
    description: "Manage document classifications",
    element: <DocumentClassPage />,
    permission: "view_documentclassification",
    isPublic: false,
  },
  {
    title: "Accessorial Charges",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/accessorial-charges",
    description: "Manage accessorial charges",
    element: <AccessorialChargePage />,
    permission: "view_accessorialcharge",
    isPublic: false,
  },
  // Customer Page
  {
    title: "Customers",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/customers/",
    description: "Manage customers",
    element: <CustomerPage />,
    permission: "customer.view_customer",
    isPublic: false,
  },
  // Dispatch pages
  {
    title: "Delay Codes",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/delay-codes/",
    description: "Delay Codes",
    element: <DelayCodePage />,
    permission: "view_delaycode",
    isPublic: false,
  },
  {
    title: "Fleet Codes",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/fleet-codes/",
    description: "Fleet Codes",
    element: <FleetCodePage />,
    permission: "view_fleetcode",
    isPublic: false,
  },
  {
    title: "Workers",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/workers/",
    description: "Workers",
    element: <WorkerPage />,
    permission: "view_worker",
    isPublic: false,
  },
  {
    title: "Comment Types",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/comment-types/",
    description: "Comment Types",
    element: <CommentTypePage />,
    permission: "view_commenttype",
    isPublic: false,
  },
  {
    title: "Location Categories",
    group: "dispatch",
    path: "/dispatch/location-categories/",
    description: "Location Categories",
    element: <LocationCategoryPage />,
    permission: "view_locationcategory",
    isPublic: false,
  },
  {
    title: "Equipment Types",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/equipment-types/",
    description: "Equipment Types",
    element: <EquipmentTypePage />,
    permission: "view_equipmenttype",
    isPublic: false,
  },
  {
    title: "Equipment Manufacturers",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/equipment-manufacturers/",
    description: "Equipment Manufacturer",
    element: <EquipmentManufacturerPage />,
    permission: "view_equipmentmanufacturer",
    isPublic: false,
  },
  {
    title: "Trailers",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/trailer/",
    description: "Trailer",
    element: <TrailerPage />,
    permission: "view_trailer",
    isPublic: false,
  },
  {
    title: "Tractors",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/tractor/",
    description: "Tractor",
    element: <TractorPage />,
    permission: "view_tractor",
    isPublic: false,
  },
  {
    title: "Locations",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/locations/",
    description: "Locations",
    element: <LocationPage />,
    permission: "view_location",
    isPublic: false,
  },
  // Shipment Pages
  {
    title: "Shipment Management",
    group: "Shipment Management",
    path: "/shipment-management/",
    description: "Shipment Management",
    element: <ShipmentManagementPage />,
    permission: "view_shipment",
    isPublic: false,
  },
  {
    title: "Add New Shipment",
    group: "Shipment Management",
    path: "/shipment-management/new-shipment",
    description: "Add New Shipment",
    element: <AddShipmentPage />,
    permission: "add_shipment",
    isPublic: false,
  },
  {
    title: "Commodity Codes",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/commodity-codes/",
    description: "Manage Commodity Codes",
    element: <CommodityPage />,
    permission: "view_commodity",
    isPublic: false,
  },
  {
    title: "Hazaroudous Material",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/hazardous-materials/",
    description: "Manage Hazardous Material",
    element: <HazardousMaterialPage />,
    permission: "view_hazardousmaterial",
    isPublic: false,
  },
  {
    title: "Service Types",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/service-types/",
    description: "Service Types",
    element: <ServiceTypePage />,
    permission: "view_servicetype",
    isPublic: false,
  },
  {
    title: "Shipment Type",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/shipment-types/",
    description: "Shipment Types",
    element: <ShipmentTypePage />,
    permission: "view_shipmenttype",
    isPublic: false,
  },
  {
    title: "Reason Code",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/reason-codes/",
    description: "Reason Codes",
    element: <ReasonCodePage />,
    permission: "view_reasoncode",
    isPublic: false,
  },
  // Stop Pages
  {
    title: "Qualifier Code",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/qualifier-codes/",
    description: "Qualifier Codes",
    element: <QualifierCodePage />,
    permission: "view_qualifiercode",
    isPublic: false,
  },
  // Admin Pages
  {
    title: "Dashboard",
    group: "Administration",
    path: "/admin/dashboard/",
    description: "Admin Dashboard",
    element: <AdminPage />,
    permission: "view_admin_dashboard",
    isPublic: false,
  },
  {
    title: "Feature Management",
    group: "Administration",
    path: "/admin/feature-management/",
    description: "Feature Flag Management",
    element: <FeatureManagementPage />,
    permission: "view_organizationfeatureflag",
    isPublic: false,
  },
  {
    title: "Accounting Control",
    group: "Administration",
    path: "/admin/accounting-controls/",
    description: "Accounting Controls",
    element: <AccountingControlPage />,
    permission: "view_accountingcontrol",
    isPublic: false,
  },
  {
    title: "Billing Control",
    group: "Administration",
    path: "/admin/billing-controls/",
    description: "Billing Controls",
    element: <BillingControlPage />,
    permission: "view_billingcontrol",
    isPublic: false,
  },
  {
    title: "Invoice Control",
    group: "Administration",
    path: "/admin/invoice-controls/",
    description: "Invoice Controls",
    element: <InvoiceControlPage />,
    permission: "view_invoicecontrol",
    isPublic: false,
  },
  {
    title: "Dispatch Control",
    group: "Administration",
    path: "/admin/dispatch-controls/",
    description: "Dispatch Controls",
    element: <DispatchControlPage />,
    permission: "view_dispatchcontrol",
    isPublic: false,
  },
  {
    title: "Shipment Control",
    group: "Administration",
    path: "/admin/shipment-controls/",
    description: "Shipment Controls",
    element: <ShipmentControlPage />,
    permission: "view_shipmentcontrol",
    isPublic: false,
  },
  {
    title: "Route Control",
    group: "Administration",
    path: "/admin/route-controls/",
    description: "Route Controls",
    element: <RouteControlPage />,
    permission: "view_routecontrol",
    isPublic: false,
  },
  {
    title: "Feasibility Control",
    group: "Administration",
    path: "/admin/feasibility-controls/",
    description: "Feasibility Controls",
    element: <FeasibilityControlPage />,
    permission: "view_feasibilitytoolcontrol",
    isPublic: false,
  },
  {
    title: "Email Control",
    group: "Administration",
    path: "/admin/email-controls/",
    description: "Email Controls",
    element: <EmailControlPage />,
    permission: "view_emailcontrol",
    isPublic: false,
  },
  {
    title: "Email Profiles",
    group: "Administration",
    path: "/admin/email-profiles/",
    description: "Email Profiles",
    element: <EmailProfilePage />,
    permission: "view_emailprofile",
    isPublic: false,
  },
  {
    title: "Google API",
    group: "Administration",
    path: "/admin/google-api/",
    description: "Google API",
    element: <GoogleAPIPage />,
    permission: "view_googleapi",
    isPublic: false,
  },
  {
    title: "Table Change Alerts",
    group: "Administration",
    path: "/admin/table-change-alerts/",
    description: "Table Change Alerts",
    element: <TableChangeAlertPage />,
    permission: "view_tablechangealert",
    isPublic: false,
  },
  {
    title: "Data Retention",
    group: "Administration",
    path: "/admin/data-retention/",
    description: "Data Retention",
    element: <DataRetentionPage />,
    permission: "view_dataretention",
    isPublic: false,
  },
  // Error Page
  {
    title: "Error",
    group: "error",
    path: "*",
    element: <ErrorPage />,
    excludeFromMenu: true,
    isPublic: false,
  },
];
