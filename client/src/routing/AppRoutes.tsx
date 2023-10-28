/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
const UserManagementPage = lazy(
  () => import("../pages/admin/users/UserManagement"),
);
const DivisionCodesPage = lazy(
  () => import("../pages/accounting/DivisionCodes"),
);
const RevenueCodesPage = lazy(() => import("../pages/accounting/RevenueCodes"));
const GLAccountsPage = lazy(() => import("../pages/accounting/GLAccounts"));
const UserSettingsPage = lazy(() => import("../pages/users/UserSettings"));
const ResetPasswordPage = lazy(() => import("../pages/reset-password-page"));
const JobTitlePage = lazy(() => import("../pages/accounts/JobTitles"));
const ControlFilesPage = lazy(
  () => import("../pages/admin/control-files/ControlFiles"),
);
const ChargeTypePage = lazy(() => import("../pages/billing/ChargeTypes"));
const AccessorialChargePage = lazy(
  () => import("../pages/billing/AccessorialCharges"),
);
const BillingClientPage = lazy(() => import("../pages/billing/BillingClient"));

const HazardousMaterialPage = lazy(
  () => import("../pages/commodities/HazardousMaterial"),
);
const CommodityPage = lazy(() => import("../pages/commodities/Commodity"));
const CustomerPage = lazy(() => import("../pages/customer/Customer"));
const ViewCustomerPage = lazy(() => import("../pages/customer/ViewCustomer"));
const EditCustomerPage = lazy(() => import("../pages/customer/EditCustomer"));
const DelayCodePage = lazy(() => import("../pages/dispatch/DelayCodes"));
const FleetCodePage = lazy(() => import("../pages/dispatch/FleetCode"));
const CommentTypePage = lazy(() => import("../pages/dispatch/CommentType"));
const RatePage = lazy(() => import("../pages/dispatch/Rate"));
const EquipmentTypePage = lazy(
  () => import("../pages/equipment/EquipmentType"),
);
const EquipmentManfacturerPage = lazy(
  () => import("../pages/equipment/EquipmentManufacturer"),
);

export type RouteObjectWithPermission = RouteObject & {
  title: string;
  group: string;
  subMenu?: string;
  path: string;
  description?: string;
  excludeFromMenu?: boolean;
  permission?: string;
};

export const routes: RouteObjectWithPermission[] = [
  {
    title: "Home",
    group: "main",
    path: "/",
    description: "Get to the main page",
    element: <HomePage />,
  },
  // Authentication Pages
  {
    title: "Login",
    group: "auth",
    path: "/login",
    element: <LoginPage />,
    excludeFromMenu: true,
  },
  {
    title: "Reset Password",
    group: "auth",
    path: "/reset-password",
    element: <ResetPasswordPage />,
    excludeFromMenu: true,
  },
  // Admin Pages
  {
    title: "User Management",
    group: "admin",
    subMenu: "users",
    path: "/admin/users",
    description: "Manage users and their permissions",
    element: <UserManagementPage />,
    permission: "admin.users.view",
  },
  {
    title: "Control Files",
    group: "admin",
    subMenu: "control files",
    path: "/admin/control-files",
    description: "Manage organization control files",
    element: <ControlFilesPage />,
    permission: "admin.can_view_all_controls",
  },
  // User Pages
  {
    title: "User Settings",
    group: "user",
    path: "/account/settings/",
    element: <UserSettingsPage />,
    excludeFromMenu: true,
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
  },
  {
    title: "Revenue Codes",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/revenue-codes",
    description: "Manage revenue codes",
    element: <RevenueCodesPage />,
    permission: "view_revenuecode",
  },
  {
    title: "General Ledger Accounts",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/gl-accounts",
    description: "Manage general ledger accounts",
    element: <GLAccountsPage />,
    permission: "view_generalledgeraccount",
  },
  // Accounts Pages
  {
    title: "Job Titles",
    group: "accounts",
    subMenu: "configuration files",
    path: "/accounts/job-titles",
    description: "Manage job titles",
    element: <JobTitlePage />,
    permission: "view_jobtitle",
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
  },
  {
    title: "Accessorial Charges",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/accessorial-charges",
    description: "Manage accessorial charges",
    element: <AccessorialChargePage />,
    permission: "view_accessorialcharge",
  },
  {
    title: "Billing Client",
    group: "billing",
    path: "/billing/client",
    description: "Your efficient partner for end-to-end billing management",
    element: <BillingClientPage />,
    permission: "billing.use_billing_client",
  },
  // Commodities Pages
  {
    title: "Hazardous Material",
    group: "commodities",
    subMenu: "configuration files",
    path: "/commodities/hazardous-material",
    description: "Manage hazardous materials",
    element: <HazardousMaterialPage />,
    permission: "view_hazardousmaterial",
  },
  {
    title: "Commodities",
    group: "commodities",
    subMenu: "configuration files",
    path: "/commodities/",
    description: "Manage commodities",
    element: <CommodityPage />,
    permission: "view_commodity",
  },
  // Customer Page
  {
    title: "Customers",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/customers/",
    description: "Manage customers",
    element: <CustomerPage />,
    permission: "view_customer",
  },
  {
    title: "View Customer",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/customers/view/:id",
    description: "View customer",
    element: <ViewCustomerPage />,
    permission: "view_customer",
  },
  {
    title: "Edit Customer",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/customers/edit/:id",
    description: "Edit customer",
    element: <EditCustomerPage />,
    permission: "view_customer",
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
  },
  {
    title: "Fleet Codes",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/fleet-codes/",
    description: "Fleet Codes",
    element: <FleetCodePage />,
    permission: "view_fleetcode",
  },
  {
    title: "Comment Types",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/comment-types/",
    description: "Comment Types",
    element: <CommentTypePage />,
    permission: "view_commenttype",
  },
  {
    title: "Rate Management",
    group: "dispatch",
    path: "/dispatch/rate-management/",
    description: "Rate Management",
    element: <RatePage />,
    permission: "view_rate",
  },
  {
    title: "Equipment Types",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/equipment-types/",
    description: "Equipment Types",
    element: <EquipmentTypePage />,
    permission: "view_equipmenttype",
  },
  {
    title: "Equipment Manufacturer",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/equipment-manufacturers/",
    description: "Equipment Manufacturer",
    element: <EquipmentManfacturerPage />,
    permission: "view_equipmentmanufacturer",
  },
  // Error Page
  {
    title: "Error",
    group: "error",
    path: "*",
    element: <ErrorPage />,
    excludeFromMenu: true,
  },
];
