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

/** Links for Billing Navigation Menu */
export const billingNavLinks = [
  {
    links: [
      {
        label: "Billing Client",
        link: "/billing/client/",
        permission: "billing.use_billing_client",
        description:
          "This module enables the generation and dispatch of invoices to customers efficiently. It streamlines the billing cycle, ensuring timely and accurate invoicing, and supports various billing formats tailored to customer needs.",
      },
      {
        label: "Billing Control",
        link: "/admin/control-files#billing-controls/",
        permission: "view_billingcontrol",
        description:
          "Provides comprehensive control over the entire billing process. This includes setting billing parameters, managing billing cycles, and ensuring compliance with financial regulations.",
      },
      {
        label: "Configuration Files",
        description:
          "Centralize configuration settings for the billing process. Adjust and customize billing workflows, rules, and parameters to align with business practices and financial strategies.",
        link: "#", // Placeholder, replace with the actual link
        subLinks: [
          {
            label: "Charge Types",
            link: "/billing/charge-types/",
            permission: "view_chargetype",
            description:
              "Categorize and manage different types of charges. This facilitates accurate billing and reporting by distinguishing between various charge categories.",
          },
          {
            label: "Division Codes",
            link: "/accounting/division-codes/",
            permission: "view_divisioncode",
            description:
              "Use these codes to segment charges and revenue by business divisions. This classification aids in detailed financial analysis and budgeting at the division level.",
          },
          {
            label: "GL Accounts",
            link: "/accounting/gl-accounts/",
            permission: "view_generalledgeraccount",
            description:
              "Manage and categorize revenue in the General Ledger. Essential for accurate financial reporting and analysis, ensuring clear visibility into revenue streams.",
          },
          {
            label: "Revenue Codes",
            link: "/accounting/revenue-codes/",
            permission: "view_revenuecode",
            description:
              "Classify revenue sources for detailed financial tracking and analysis. These codes help in understanding revenue patterns and making informed financial decisions.",
          },
          {
            label: "Accessorial Charges",
            link: "/billing/accessorial-charges/",
            permission: "view_accessorialcharge",
            description:
              "Define and manage additional charges associated with transportation services. This includes detention, layover, and other incidental charges.",
          },
          {
            label: "Customers",
            link: "/billing/customers/",
            permission: "view_customer",
            description:
              "Manage customer-related data and categorize billing information. Essential for personalized billing management and maintaining accurate customer financial records.",
          },
        ],
      },
    ],
  },
];

/** Links for Dispatch Navigation Menu */
export const dispatchNavLinks = [
  {
    links: [
      {
        label: "Rate Management",
        link: "/dispatch/rate-management/",
        permission: "view_rate",
        description:
          "This module allows for the comprehensive management of freight and transportation rates. It includes features for setting, adjusting, and analyzing rates, ensuring competitive pricing and operational efficiency.",
      },
      {
        label: "Configuration Files",
        description:
          "Central hub for configuring and customizing the dispatch process. This includes setting dispatch parameters, defining operational rules, and ensuring alignment with logistical strategies.",
        link: "#",
        subLinks: [
          {
            label: "Comment Type",
            link: "/dispatch/comment-types/",
            permission: "view_commenttype",
            description:
              "Categorize and manage different types of operational comments. This aids in streamlining communication and documenting specific details related to dispatch activities.",
          },
          {
            label: "Delay Codes",
            link: "/dispatch/delay-codes/",
            permission: "view_delaycode",
            description:
              "Identify and categorize various types of delays encountered during dispatch operations. Essential for analyzing and mitigating operational disruptions.",
          },
          {
            label: "Fleet Codes",
            link: "/dispatch/fleet-codes/",
            permission: "view_fleetcode",
            description:
              "Organize and classify different fleet segments. Facilitates efficient fleet management and helps in tracking and analyzing fleet performance.",
          },
          {
            label: "Locations",
            link: "/dispatch/locations/",
            permission: "view_location",
            description:
              "Manage and categorize operational locations, including depots, warehouses, and delivery points. Crucial for route planning and logistical coordination.",
          },
          {
            label: "Routes",
            link: "/dispatch/routes/",
            permission: "view_route",
            description:
              "Define and categorize various transportation routes. Supports strategic route planning and optimization for enhanced delivery efficiency.",
          },
          {
            label: "Route Control",
            link: "/admin/control-files#route-controls",
            permission: "view_routecontrol",
            description:
              "Manage and control route configurations, ensuring adherence to predefined operational and safety standards. Key for maintaining consistency in route planning.",
          },
          {
            label: "Location Categories",
            link: "/dispatch/location-categories/",
            permission: "view_locationcategory",
            description:
              "Segment locations into distinct categories for better logistical planning. Helps in tailoring operations to specific location characteristics and requirements.",
          },
        ],
      },
    ],
  },
];

/** Links for Equipment Maintenance Navigation Menu */
export const equipmentNavLinks = [
  {
    links: [
      {
        label: "Equipment Maintenance Plan",
        link: "#",
        permission: "view_equipmentmaintenanceplan",
        description:
          "This section facilitates the creation and management of comprehensive maintenance schedules for various equipment. It enables precise tracking and proactive maintenance activities, ensuring optimal equipment performance and longevity.",
      },
      {
        label: "Configuration Files",
        link: "#",
        description:
          "Access and modify the core configuration settings governing the equipment maintenance processes. This central hub allows for the customization and fine-tuning of maintenance workflows and parameters.",
        subLinks: [
          {
            label: "Equipment Types",
            link: "/equipment/equipment-types/",
            permission: "view_equipmenttype",
            description:
              "Define and manage the different categories of equipment. This classification system aids in streamlining maintenance protocols and inventory management based on equipment types.",
          },
          {
            label: "Equipment Manufacturers",
            link: "/equipment/equipment-manufacturers/",
            permission: "view_equipmentmanufacturer",
            description:
              "Organize and view equipment based on their manufacturers. This section helps in aligning maintenance strategies with specific manufacturer guidelines and specifications.",
          },
          {
            label: "Tractor",
            link: "/equipment/tractor/",
            permission: "view_tractor",
            description:
              "Dedicated section for managing and categorizing tractors. It includes detailed information and specific maintenance guidelines tailored to tractors, enhancing their operational efficiency.",
          },
          {
            label: "Trailer",
            link: "/equipment/trailer/",
            permission: "view_trailer",
            description:
              "Focuses on the management and classification of trailers. This part of the system provides specialized maintenance schedules and operational details specific to different types of trailers.",
          },
        ],
      },
    ],
  },
];

/** Links for Shipment Navigation Menu */
export const shipmentNavLinks = [
  {
    links: [
      {
        label: "Shipment Management",
        link: "/shipment-management/",
        permission: "view_shipment",
        description:
          "This module provides comprehensive tools for managing all aspects of shipments, including scheduling, tracking, and status updates. It's designed to streamline the shipment lifecycle from origin to destination, ensuring timely and efficient delivery.",
      },
      {
        label: "Shipment Controls",
        link: "/admin/control-files#order-controls/",
        permission: "view_shipmentcontrol",
        description:
          "Gain control over the shipment process with customizable settings and rules. This section allows for the fine-tuning of shipment operations, ensuring compliance with logistical standards and customer expectations.",
      },
      {
        label: "Configuration Files",
        link: "#",
        description:
          "Centralize the configuration for all shipment-related processes. Adjust parameters and settings to align shipment operations with business goals and operational efficiency.",
        subLinks: [
          {
            label: "Formula Templates",
            link: "/order/formula-template/",
            permission: "view_formulatemplate",
            description:
              "Create and manage formula templates for calculating shipment-related metrics. Essential for automating and standardizing complex calculations in the shipment process.",
          },
          {
            label: "Shipment Types",
            link: "/shipment-management/shipment-types/",
            permission: "view_shipmenttype",
            description:
              "Categorize shipments into distinct types for better management and tracking. This helps in tailoring operations to the specific requirements of different shipment categories.",
          },
          {
            label: "Service Types",
            link: "/shipment-management/service-types/",
            permission: "view_servicetype",
            description:
              "Define and manage various service types offered in the shipping process. Facilitates customized service offerings and helps in aligning services with customer needs.",
          },
          {
            label: "Movements",
            link: "/shipment-management/movements/",
            permission: "movements.view_movement",
            description:
              "Organize and classify different types of shipment movements. Key for planning and optimizing logistical routes and schedules.",
          },
          {
            label: "Qualifier Codes",
            link: "/shipment-management/qualifier-codes/",
            permission: "stops.view_qualifiercode",
            description:
              "Manage codes that qualify different aspects of shipments. These codes are crucial for detailed categorization and analysis of shipment attributes.",
          },
          {
            label: "Reason Codes",
            link: "/shipment-management/reason-codes/",
            permission: "view_reasoncode",
            description:
              "Categorize and document different reasons related to shipment processes, such as delays or modifications. Essential for analyzing operational challenges and implementing improvements.",
          },
        ],
      },
    ],
  },
];
