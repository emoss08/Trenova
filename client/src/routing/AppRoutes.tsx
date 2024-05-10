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
const GLAccountsPage = lazy(
  () => import("../pages/accounting/GeneralLedgerAccounts"),
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
const HazardousMaterialSegregationPage = lazy(
  () => import("../pages/admin/HazardousMaterialSegregation"),
);
const TableChangeAlertPage = lazy(
  () => import("../pages/admin/TableChangeAlerts"),
);
const DataRetentionPage = lazy(() => import("../pages/admin/DataRetention"));
const RoleManagementPage = lazy(() => import("../pages/admin/Roles"));

// Commodity Pages
const CommodityPage = lazy(() => import("../pages/commodities/Commodities"));
const HazardousMaterialPage = lazy(
  () => import("../pages/commodities/HazardousMaterials"),
);

// Other Pages
const ResetPasswordPage = lazy(() => import("../pages/reset-password-page"));
const ChargeTypePage = lazy(() => import("../pages/billing/ChargeTypes"));
const AccessorialChargePage = lazy(
  () => import("../pages/billing/AccessorialCharges"),
);
const CustomerPage = lazy(() => import("../pages/customer/Customers"));
const WorkerPage = lazy(() => import("../pages/worker/Workers"));
const DelayCodePage = lazy(() => import("../pages/dispatch/DelayCodes"));
const FleetCodePage = lazy(() => import("../pages/dispatch/FleetCodes"));
const CommentTypePage = lazy(() => import("../pages/dispatch/CommentTypes"));
const ShipmentManagementPage = lazy(
  () => import("../pages/shipment/Shipments"),
);
const EquipmentTypePage = lazy(
  () => import("../pages/equipment/EquipmentTypes"),
);
const EquipmentManufacturerPage = lazy(
  () => import("../pages/equipment/EquipmentManufacturers"),
);
const ServiceTypePage = lazy(() => import("../pages/shipment/ServiceTypes"));
const QualifierCodePage = lazy(() => import("../pages/stop/QualifierCodes"));
const ReasonCodePage = lazy(() => import("../pages/shipment/ReasonCodes"));
const ShipmentTypePage = lazy(() => import("../pages/shipment/ShipmentTypes"));
const LocationPage = lazy(() => import("../pages/dispatch/Locations"));
const TrailerPage = lazy(() => import("../pages/equipment/Trailers"));
const TractorPage = lazy(() => import("../pages/equipment/Tractors"));
const DocumentClassPage = lazy(
  () => import("../pages/billing/DocumentClassifications"),
);
const AdminPage = lazy(() => import("../pages/admin/Dashboard"));

export type RouteObjectWithPermission = RouteObject & {
  /**
   * The unique key of the route
   * This is used to identify the route in the menu
   */
  key?: string;

  /**
   * The title of the route
   * This is displayed in the menu
   */
  title: string;

  /**
   * The group to which the route belongs
   * This is used to group the routes in the menu
   * If not provided, the route is displayed in the main menu.
   */
  group: string;

  /**
   * The sub-menu to which the route belongs
   * This is used to group the routes in the menu
   * If not provided, the route is displayed in the main menu.
   */
  subMenu?: string;

  /**
   * The path of the route
   * This is used to match the route with the current URL
   */
  path: string;

  /**
   * The component to render when the route is active
   * This is a lazy-loaded component
   * @see https://reactjs.org/docs/code-splitting.html
   */
  description?: string;

  /**
   * If true, the route is not displayed in the menu
   * This is useful for routes that are only accessible via a link or a button
   * or for routes that are not meant to be accessed directly.
   */
  excludeFromMenu?: boolean;

  /**
   * The permission required to access the route
   * If not provided, the route is accessible to all authenticated users
   * If the route is public, the permission is ignored
   */
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
  // Accounting Pages
  {
    title: "Division Codes",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/division-codes",
    description: "Manage division codes",
    element: <DivisionCodesPage />,
    permission: "divisioncode.view",
    isPublic: false,
  },
  {
    title: "Revenue Codes",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/revenue-codes",
    description: "Manage revenue codes",
    element: <RevenueCodesPage />,
    permission: "revenuecode.view",
    isPublic: false,
  },
  {
    title: "General Ledger Accounts",
    group: "accounting",
    subMenu: "configuration files",
    path: "/accounting/gl-accounts",
    description: "Manage general ledger accounts",
    element: <GLAccountsPage />,
    permission: "generalledgeraccount.view",
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
    permission: "chargetype.view",
    isPublic: false,
  },
  {
    title: "Document Classifications",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/document-classes",
    description: "Manage document classifications",
    element: <DocumentClassPage />,
    permission: "documentclassification.view",
    isPublic: false,
  },
  {
    title: "Accessorial Charges",
    group: "billing",
    subMenu: "configuration files",
    path: "/billing/accessorial-charges",
    description: "Manage accessorial charges",
    element: <AccessorialChargePage />,
    permission: "accessorialcharge.view",
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
    permission: "customer.view",
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
    permission: "delaycode.view",
    isPublic: false,
  },
  {
    title: "Fleet Codes",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/fleet-codes/",
    description: "Fleet Codes",
    element: <FleetCodePage />,
    permission: "fleetcode.view",
    isPublic: false,
  },
  {
    title: "Workers",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/workers/",
    description: "Workers",
    element: <WorkerPage />,
    permission: "worker.view",
    isPublic: false,
  },
  {
    title: "Comment Types",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/comment-types/",
    description: "Comment Types",
    element: <CommentTypePage />,
    permission: "commenttype.view",
    isPublic: false,
  },
  {
    title: "Location Categories",
    group: "dispatch",
    path: "/dispatch/location-categories/",
    description: "Location Categories",
    element: <LocationCategoryPage />,
    permission: "locationcategory.view",
    isPublic: false,
  },
  {
    title: "Equipment Types",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/equipment-types/",
    description: "Equipment Types",
    element: <EquipmentTypePage />,
    permission: "equipmenttype.view",
    isPublic: false,
  },
  {
    title: "Equipment Manufacturers",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/equipment-manufacturers/",
    description: "Equipment Manufacturer",
    element: <EquipmentManufacturerPage />,
    permission: "equipmentmanufacturer.view",
    isPublic: false,
  },
  {
    title: "Trailers",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/trailer/",
    description: "Trailer",
    element: <TrailerPage />,
    permission: "trailer.view",
    isPublic: false,
  },
  {
    title: "Tractors",
    group: "equipment",
    subMenu: "configuration files",
    path: "/equipment/tractor/",
    description: "Tractor",
    element: <TractorPage />,
    permission: "tractor.view",
    isPublic: false,
  },
  {
    title: "Locations",
    group: "dispatch",
    subMenu: "configuration files",
    path: "/dispatch/locations/",
    description: "Locations",
    element: <LocationPage />,
    permission: "location",
    isPublic: false,
  },
  // Shipment Pages
  {
    title: "Shipment Management",
    group: "Shipment Management",
    path: "/shipment-management/",
    description: "Shipment Management",
    element: <ShipmentManagementPage />,
    permission: "shipment.view",
    isPublic: false,
  },
  {
    title: "Add New Shipment",
    group: "Shipment Management",
    path: "/shipment-management/new-shipment",
    description: "Add New Shipment",
    element: <AddShipmentPage />,
    permission: "shipment.add",
    isPublic: false,
  },
  {
    title: "Commodity Codes",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/commodity-codes/",
    description: "Manage Commodity Codes",
    element: <CommodityPage />,
    permission: "commodity.view",
    isPublic: false,
  },
  {
    title: "Hazaroudous Material",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/hazardous-materials/",
    description: "Manage Hazardous Material",
    element: <HazardousMaterialPage />,
    permission: "hazardousmaterial.view",
    isPublic: false,
  },
  {
    title: "Service Types",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/service-types/",
    description: "Service Types",
    element: <ServiceTypePage />,
    permission: "servicetype.view",
    isPublic: false,
  },
  {
    title: "Shipment Types",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/shipment-types/",
    description: "Shipment Types",
    element: <ShipmentTypePage />,
    permission: "shipmenttype.view",
    isPublic: false,
  },
  {
    title: "Reason Codes",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/reason-codes/",
    description: "Reason Codes",
    element: <ReasonCodePage />,
    permission: "reasoncode.view",
    isPublic: false,
  },
  // Stop Pages
  {
    title: "Qualifier Codes",
    group: "Shipment Management",
    subMenu: "configuration files",
    path: "/shipment-management/qualifier-codes/",
    description: "Qualifier Codes",
    element: <QualifierCodePage />,
    permission: "qualifiercode.view",
    isPublic: false,
  },
  // Admin Pages
  {
    title: "Dashboard",
    group: "Administration",
    path: "/admin/dashboard/",
    description: "Admin Dashboard",
    element: <AdminPage />,
    permission: "admin_dashboard.view",
    isPublic: false,
  },
  {
    title: "Feature Management",
    group: "Administration",
    path: "/admin/feature-management/",
    description: "Feature Flag Management",
    element: <FeatureManagementPage />,
    permission: "organizationfeatureflag.view",
    isPublic: false,
  },
  {
    title: "Accounting Control",
    group: "Administration",
    path: "/admin/accounting-controls/",
    description: "Accounting Controls",
    element: <AccountingControlPage />,
    permission: "accountingcontrol.view",
    isPublic: false,
  },
  {
    title: "Billing Control",
    group: "Administration",
    path: "/admin/billing-controls/",
    description: "Billing Controls",
    element: <BillingControlPage />,
    permission: "billingcontrol.view",
    isPublic: false,
  },
  {
    title: "Invoice Control",
    group: "Administration",
    path: "/admin/invoice-controls/",
    description: "Invoice Controls",
    element: <InvoiceControlPage />,
    permission: "invoicecontrol.view",
    isPublic: false,
  },
  {
    title: "Dispatch Control",
    group: "Administration",
    path: "/admin/dispatch-controls/",
    description: "Dispatch Controls",
    element: <DispatchControlPage />,
    permission: "dispatchcontrol.view",
    isPublic: false,
  },
  {
    title: "Shipment Control",
    group: "Administration",
    path: "/admin/shipment-controls/",
    description: "Shipment Controls",
    element: <ShipmentControlPage />,
    permission: "shipmentcontrol.view",
    isPublic: false,
  },
  {
    title: "Route Control",
    group: "Administration",
    path: "/admin/route-controls/",
    description: "Route Controls",
    element: <RouteControlPage />,
    permission: "routecontrol.view",
    isPublic: false,
  },
  {
    title: "Feasibility Control",
    group: "Administration",
    path: "/admin/feasibility-controls/",
    description: "Feasibility Controls",
    element: <FeasibilityControlPage />,
    permission: "feasibilitytoolcontrol.view",
    isPublic: false,
  },
  {
    title: "Email Control",
    group: "Administration",
    path: "/admin/email-controls/",
    description: "Email Controls",
    element: <EmailControlPage />,
    permission: "emailcontrol.view",
    isPublic: false,
  },
  {
    title: "Email Profiles",
    group: "Administration",
    path: "/admin/email-profiles/",
    description: "Email Profiles",
    element: <EmailProfilePage />,
    permission: "emailprofile.view",
    isPublic: false,
  },
  {
    title: "Google API",
    group: "Administration",
    path: "/admin/google-api/",
    description: "Google API",
    element: <GoogleAPIPage />,
    permission: "googleapi.view",
    isPublic: false,
  },
  {
    title: "Table Change Alerts",
    group: "Administration",
    path: "/admin/table-change-alerts/",
    description: "Table Change Alerts",
    element: <TableChangeAlertPage />,
    permission: "tablechangealert.view",
    isPublic: false,
  },
  {
    title: "Data Retention",
    group: "Administration",
    path: "/admin/data-retention/",
    description: "Data Retention",
    element: <DataRetentionPage />,
    permission: "dataretention.view",
    isPublic: false,
  },
  {
    title: "Hazardous Material Seg. Rules",
    group: "Administration",
    path: "/admin/hazardous-rules/",
    description: "Hazardous Material Seg. Rules",
    element: <HazardousMaterialSegregationPage />,
    permission: "hazardousmaterialsegregation.view",
    isPublic: false,
  },
  {
    title: "Role Management",
    group: "Administration",
    path: "/admin/roles/",
    description: "Role Management",
    element: <RoleManagementPage />,
    permission: "role.view",
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
