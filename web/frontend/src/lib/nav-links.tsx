/**
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

import { type NavLinkGroup } from "@/types/sidebar-nav";
import {
  faFolders,
  faMoneyBillTransfer,
  faScrewdriverWrench,
  faTools,
  faTruck,
  faVault,
} from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

/** Links for Billing Navigation Menu */
export const billingNavLinks: NavLinkGroup[] = [
  {
    menuKey: "dispatchMenu",
    minimizedIcon: <FontAwesomeIcon icon={faVault} />,
    label: "Dispatch Management",
    links: [
      {
        menuKey: "billingMenu",
        key: "billingClient",
        label: "Billing Client",
        link: "/billing/client/",
        permission: "billing_client.view",
        description:
          "This module enables the generation and dispatch of invoices to customers efficiently. It streamlines the billing cycle, ensuring timely and accurate invoicing, and supports various billing formats tailored to customer needs.",
        icon: <FontAwesomeIcon icon={faMoneyBillTransfer} />,
      },
      {
        menuKey: "billingMenu",
        key: "billingConfiguration",
        label: "Configuration Files",
        description:
          "Centralize configuration settings for the billing process. Adjust and customize billing workflows, rules, and parameters to align with business practices and financial strategies.",
        link: "#", // Placeholder, replace with the actual link
        icon: <FontAwesomeIcon icon={faFolders} />,
        subLinks: [
          {
            menuKey: "billingMenu",
            label: "Charge Types",
            link: "/billing/charge-types/",
            key: "chargeTypes",
            permission: "charge_type:view",
            description:
              "Categorize and manage different types of charges. This facilitates accurate billing and reporting by distinguishing between various charge categories.",
          },
          {
            menuKey: "billingMenu",
            key: "divisionCodes",
            label: "Division Codes",
            link: "/accounting/division-codes/",
            permission: "division_code:view",
            description:
              "Use these codes to segment charges and revenue by business divisions. This classification aids in detailed financial analysis and budgeting at the division level.",
          },
          {
            menuKey: "billingMenu",
            key: "glAccounts",
            label: "GL Accounts",
            link: "/accounting/gl-accounts/",
            permission: "general_ledger_account:view",
            description:
              "Manage and categorize revenue in the General Ledger. Essential for accurate financial reporting and analysis, ensuring clear visibility into revenue streams.",
          },
          {
            menuKey: "billingMenu",
            key: "revenueCodes",
            label: "Revenue Codes",
            link: "/accounting/revenue-codes/",
            permission: "revenue_code:view",
            description:
              "Classify revenue sources for detailed financial tracking and analysis. These codes help in understanding revenue patterns and making informed financial decisions.",
          },
          {
            menuKey: "billingMenu",
            key: "accessorialCharges",
            label: "Accessorial Charges",
            link: "/billing/accessorial-charges/",
            permission: "accessorial_charge:view",
            description:
              "Define and manage additional charges associated with transportation services. This includes detention, layover, and other incidental charges.",
          },
          {
            menuKey: "billingMenu",
            key: "customers",
            label: "Customers",
            link: "/billing/customers/",
            permission: "customer:view",
            description:
              "Manage customer-related data and categorize billing information. Essential for personalized billing management and maintaining accurate customer financial records.",
          },
          {
            menuKey: "billingMenu",
            key: "documentClassifications",
            label: "Document Classifications",
            link: "/billing/document-classes/",
            permission: "document_classification:view",
            description:
              "Optimize billing management by categorizing essential documents like Proof of Delivery and Bills of Lading, ensuring accurate and efficient customer financial record keeping.",
          },
        ],
      },
    ],
  },
];

/** Links for Dispatch Navigation Menu */
export const dispatchNavLinks: NavLinkGroup[] = [
  {
    menuKey: "dispatchMenu",
    minimizedIcon: <FontAwesomeIcon icon={faVault} />,
    label: "Dispatch Management",
    links: [
      {
        menuKey: "dispatchMenu",
        key: "rateManagement",
        label: "Rate Management",
        link: "/dispatch/rate-management/",
        permission: "rate:view",
        description:
          "This module allows for the comprehensive management of freight and transportation rates. It includes features for setting, adjusting, and analyzing rates, ensuring competitive pricing and operational efficiency.",
        icon: <FontAwesomeIcon icon={faVault} />,
      },
      {
        menuKey: "dispatchMenu",
        key: "dispatchConfiguration",
        label: "Configuration Files",
        description:
          "Central hub for configuring and customizing the dispatch process. This includes setting dispatch parameters, defining operational rules, and ensuring alignment with logistical strategies.",
        link: "#",
        icon: <FontAwesomeIcon icon={faFolders} />,
        subLinks: [
          {
            menuKey: "dispatchMenu",
            key: "commentType",
            label: "Comment Type",
            link: "/dispatch/comment-types/",
            permission: "comment_type:view",
            description:
              "Categorize and manage different types of operational comments. This aids in streamlining communication and documenting specific details related to dispatch activities.",
          },
          {
            menuKey: "dispatchMenu",
            key: "delayCodes",
            label: "Delay Codes",
            link: "/dispatch/delay-codes/",
            permission: "delay_code:view",
            description:
              "Identify and categorize various types of delays encountered during dispatch operations. Essential for analyzing and mitigating operational disruptions.",
          },
          {
            menuKey: "dispatchMenu",
            key: "fleetCodes",
            label: "Fleet Codes",
            link: "/dispatch/fleet-codes/",
            permission: "fleet_code:view",
            description:
              "Organize and classify different fleet segments. Facilitates efficient fleet management and helps in tracking and analyzing fleet performance.",
          },
          {
            menuKey: "dispatchMenu",
            key: "locations",
            label: "Locations",
            link: "/dispatch/locations/",
            permission: "location:view",
            description:
              "Manage and categorize operational locations, including depots, warehouses, and delivery points. Crucial for route planning and logistical coordination.",
          },
          {
            menuKey: "dispatchMenu",
            key: "routes",
            label: "Routes",
            link: "/dispatch/routes/",
            permission: "route:view",
            description:
              "Define and categorize various transportation routes. Supports strategic route planning and optimization for enhanced delivery efficiency.",
          },
          {
            menuKey: "dispatchMenu",
            key: "locationCategories",
            label: "Location Categories",
            link: "/dispatch/location-categories/",
            permission: "location_category:view",
            description:
              "Segment locations into distinct categories for better logistical planning. Helps in tailoring operations to specific location characteristics and requirements.",
          },
          {
            menuKey: "dispatchMenu",
            key: "workers",
            label: "Workers",
            link: "/dispatch/workers/",
            permission: "worker:view",
            description:
              "Manage and categorize workers involved in dispatch operations. This includes drivers, dispatchers, and other operational staff.",
          },
        ],
      },
    ],
  },
];

/** Links for Equipment Maintenance Navigation Menu */
export const equipmentNavLinks: NavLinkGroup[] = [
  {
    menuKey: "equipmentMenu",
    minimizedIcon: <FontAwesomeIcon icon={faScrewdriverWrench} />,
    label: "Equipment Management",
    links: [
      {
        menuKey: "equipmentMenu",
        key: "equipmentMaintenancePlan",
        label: "Equipment Maintenance Plan",
        link: "#",
        permission: "equipment_maintenance_plan:view",
        description:
          "This section facilitates the creation and management of comprehensive maintenance schedules for various equipment. It enables precise tracking and proactive maintenance activities, ensuring optimal equipment performance and longevity.",
        icon: <FontAwesomeIcon icon={faScrewdriverWrench} />,
      },
      {
        menuKey: "equipmentMenu",
        key: "equipmentConfiguration",
        label: "Configuration Files",
        link: "#",
        description:
          "Access and modify the core configuration settings governing the equipment maintenance processes. This central hub allows for the customization and fine-tuning of maintenance workflows and parameters.",
        icon: <FontAwesomeIcon icon={faFolders} />,
        subLinks: [
          {
            menuKey: "equipmentMenu",
            key: "equipmentTypes",
            label: "Equipment Types",
            link: "/equipment/equipment-types/",
            permission: "equipment_type:view",
            description:
              "Define and manage the different categories of equipment. This classification system aids in streamlining maintenance protocols and inventory management based on equipment types.",
          },
          {
            menuKey: "equipmentMenu",
            key: "equipmentManufacturers",
            label: "Equipment Manufacturers",
            link: "/equipment/equipment-manufacturers/",
            permission: "equipment_manufacturer:view",
            description:
              "Organize and view equipment based on their manufacturers. This section helps in aligning maintenance strategies with specific manufacturer guidelines and specifications.",
          },
          {
            menuKey: "equipmentMenu",
            key: "tractors",
            label: "Tractor",
            link: "/equipment/tractor/",
            permission: "tractor:view",
            description:
              "Dedicated section for managing and categorizing tractors. It includes detailed information and specific maintenance guidelines tailored to tractors, enhancing their operational efficiency.",
          },
          {
            menuKey: "equipmentMenu",
            key: "trailers",
            label: "Trailer",
            link: "/equipment/trailer/",
            permission: "trailer:view",
            description:
              "Focuses on the management and classification of trailers. This part of the system provides specialized maintenance schedules and operational details specific to different types of trailers.",
          },
        ],
      },
    ],
  },
];

/** Links for Shipment Navigation Menu */
export const shipmentNavLinks: NavLinkGroup[] = [
  {
    menuKey: "shipmentMenu",
    minimizedIcon: <FontAwesomeIcon icon={faTruck} />,
    label: "Shipment",
    links: [
      {
        menuKey: "shipmentMenu",
        key: "shipmentManagement",
        label: "Shipment Management",
        link: "/shipments/shipment-management/",
        permission: "shipment:view",
        description:
          "This module provides comprehensive tools for managing all aspects of shipments, including scheduling, tracking, and status updates. It's designed to streamline the shipment lifecycle from origin to destination, ensuring timely and efficient delivery.",
        icon: <FontAwesomeIcon icon={faTruck} />,
      },
      {
        menuKey: "shipmentMenu",
        key: "shipmentConfiguration",
        label: "Configuration Files",
        link: "#",
        description:
          "Centralize the configuration for all shipment-related processes. Adjust parameters and settings to align shipment operations with business goals and operational efficiency.",
        icon: <FontAwesomeIcon icon={faFolders} />,
        subLinks: [
          {
            menuKey: "shipmentMenu",
            key: "formulaTemplates",
            label: "Formula Templates",
            link: "/order/formula-template/",
            permission: "formula_template:view",
            description:
              "Create and manage formula templates for calculating shipment-related metrics. Essential for automating and standardizing complex calculations in the shipment process.",
          },
          {
            menuKey: "shipmentMenu",
            key: "shipmentTypes",
            label: "Shipment Types",
            link: "/shipments/shipment-types/",
            permission: "shipment_type:view",
            description:
              "Categorize shipments into distinct types for better management and tracking. This helps in tailoring operations to the specific requirements of different shipment categories.",
          },
          {
            menuKey: "shipmentMenu",
            key: "serviceTypes",
            label: "Service Types",
            link: "/shipments/service-types/",
            permission: "service_type:view",
            description:
              "Define and manage various service types offered in the shipping process. Facilitates customized service offerings and helps in aligning services with customer needs.",
          },
          {
            menuKey: "shipmentMenu",
            key: "qualifierCodes",
            label: "Qualifier Codes",
            link: "/shipments/qualifier-codes/",
            permission: "qualifier_code:view",
            description:
              "Manage codes that qualify different aspects of shipments. These codes are crucial for detailed categorization and analysis of shipment attributes.",
          },
          {
            menuKey: "shipmentMenu",
            key: "commodityCodes",
            label: "Commodity Codes",
            link: "/shipments/commodity-codes/",
            permission: "commodity:view",
            description:
              "Categorize shipments based on the type of commodities being transported. This classification system helps in streamlining shipment operations and optimizing routes.",
          },
          {
            menuKey: "shipmentMenu",
            key: "hazardousMaterials",
            label: "Hazardous Materials",
            link: "/shipments/hazardous-materials/",
            permission: "hazardous_material:view",
            description:
              "Manage and categorize shipments containing hazardous materials. This section includes detailed information and specific guidelines for handling hazardous materials.",
          },
          {
            menuKey: "shipmentMenu",
            key: "reasonCodes",
            label: "Reason Codes",
            link: "/shipments/reason-codes/",
            permission: "reason_code:view",
            description:
              "Categorize and document different reasons related to shipment processes, such as delays or modifications. Essential for analyzing operational challenges and implementing improvements.",
          },
        ],
      },
    ],
  },
];

export const adminNavLinks: NavLinkGroup[] = [
  {
    menuKey: "adminMenu",
    minimizedIcon: <FontAwesomeIcon icon={faTools} />,
    label: "Administration",
    links: [
      {
        menuKey: "adminMenu",
        key: "adminDashboard",
        label: "Admin Dashboard",
        link: "/admin/dashboard/",
        permission: "admin_dashboard:view",
        description: "Monitor system performance and user activity...",
        icon: <FontAwesomeIcon icon={faTools} />,
      },
    ],
  },
];
