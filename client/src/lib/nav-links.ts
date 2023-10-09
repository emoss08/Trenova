/** Links for Billing Navigation Menu */
export const billingNavLinks = [
  {
    links: [
      {
        label: "Billing Client",
        link: "/billing/client/",
        permission: "billing.billing.use_billing_client",
        description:
          "The Billing Client is used to create and send invoices to customers.",
      },
      {
        label: "Billing Control",
        link: "/admin/control-files#billing-controls/",
        permission: "billing.view_billingcontrol",
        description:
          "Billing Controls are used to control the billing process.",
      },
      {
        label: "Configuration Files",
        description:
          "Configuration Files are used to configure the billing process.",
        link: "#", // Placeholder, replace with the actual link
        subLinks: [
          {
            label: "Charge Types",
            link: "/billing/charge-types/",
            permission: "billing.view_chargetype",
            description: "Charge Types are used to categorize charges.",
          },
          {
            label: "Division Codes",
            link: "/accounting/division-codes/",
            permission: "accounting.view_divisioncode",
            description:
              "Division Codes are used to categorize charges and revenue.",
          },
          {
            label: "GL Accounts",
            link: "/accounting/gl-accounts/",
            permission: "accounting.view_generalledgeraccount",
            description: "GL Accounts are used to categorize revenue.",
          },
          {
            label: "Revenue Codes",
            link: "/accounting/revenue-codes/",
            permission: "accounting.view_revenuecode",
            description: "Revenue Codes are used to categorize revenue.",
          },
          {
            label: "Accessorial Charges",
            link: "/billing/accessorial-charges/",
            permission: "billing.view_accessorialcharge",
            description: "Accessorial Charges are used to categorize charges.",
          },
          {
            label: "Customers",
            link: "/billing/customers/",
            permission: "customer.view_customer",
            description: "Customers are used to categorize charges.",
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
        permission: "dispatch.view_rate",
        description: "Rate Management is used to manage rates.",
      },
      {
        label: "Configuration Files",
        description:
          "Configuration Files are used to configure the dispatch process.",
        link: "#",
        subLinks: [
          {
            label: "Comment Type",
            link: "/dispatch/comment-types/",
            permission: "dispatch.view_commenttype",
            description: "Comment Types are used to categorize comments.",
          },
          {
            label: "Delay Codes",
            link: "/dispatch/delay-codes/",
            permission: "dispatch.view_delaycode",
            description: "Delay Codes are used to categorize delays.",
          },
          {
            label: "Fleet Codes",
            link: "/dispatch/fleet-codes/",
            permission: "dispatch.view_fleetcode",
            description: "Fleet Codes are used to categorize fleets.",
          },
          {
            label: "Locations",
            link: "/dispatch/locations/",
            permission: "location.view_location",
            description: "Locations are used to categorize locations.",
          },
          {
            label: "Routes",
            link: "/dispatch/routes/",
            permission: "route.view_route",
            description: "Routes are used to categorize routes.",
          },
          {
            label: "Route Control",
            link: "/admin/control-files#route-controls",
            permission: "route.view_routecontrol",
            description: "Route Control are used to categorize routes.",
          },
          {
            label: "Locations Categories",
            link: "/dispatch/locations-categories/",
            permission: "location.view_locationcategory",
            description:
              "Locations Categories are used to categorize locations.",
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
        permission: "equipment.view_equipmentmaintenanceplan",
        description:
          "The Equipment Maintenance Plan is used to create and manage equipment maintenance plans.",
      },
      {
        label: "Configuration Files",
        link: "#", // Placeholder, replace with the actual link
        description:
          "Configuration Files are used to configure the equipment maintenance process.",
        subLinks: [
          {
            label: "Equipment Types",
            link: "/equipment/equipment-types/",
            permission: "equipment.view_equipmenttype",
            description: "Equipment Types are used to categorize equipment.",
          },
          {
            label: "Equipment Manufacturers",
            link: "/equipment/equipment-manufacturers/",
            permission: "equipment.view_equipmentmanufacturer",
            description:
              "Equipment Manufacturers are used to categorize equipment.",
          },
          {
            label: "Tractor",
            link: "/equipment/tractor/",
            permission: "equipment.view_tractor",
            description: "Tractor is used to categorize equipment.",
          },
          {
            label: "Trailer",
            link: "/equipment/trailer/",
            permission: "equipment.view_trailer",
            description: "Trailer is used to categorize equipment.",
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
        permission: "shipment.view_shipment",
        description: "Shipment Management is used to manage shipments.",
      },
      {
        label: "Shipment Controls",
        link: "/admin/control-files#order-controls/",
        permission: "shipment.view_shipmentcontrol",
        description:
          "Shipment Controls are used to control the shipment process.",
      },
      {
        label: "Configuration Files",
        link: "#",
        description:
          "Configuration Files are used to configure the shipment process.",
        subLinks: [
          {
            label: "Formula Templates",
            link: "/order/formula-template/",
            permission: "shipment.view_formulatemplate",
            description: "Formula Templates are used to create formulas.",
          },
          {
            label: "Shipment Types",
            link: "/shipment-management/shipment-types/",
            permission: "shipment.view_shipmenttype",
            description: "Shipment Types are used to categorize shipments.",
          },
          {
            label: "Service Types",
            link: "/shipment-management/service-types/",
            permission: "shipment.view_servicetype",
            description: "Service Types are used to categorize services.",
          },
          {
            label: "Movements",
            link: "/shipment-management/movements/",
            permission: "movements.view_movement",
            description: "Movements are used to categorize movements.",
          },
          {
            label: "Qualifier Codes",
            link: "/shipment-management/qualifier-codes/",
            permission: "stops.view_qualifiercode",
            description: "Qualifier Codes are used to categorize qualifiers.",
          },
          {
            label: "Reason Codes",
            link: "/shipment-management/reason-codes/",
            permission: "shipment.view_reasoncode",
            description: "Reason Codes are used to categorize reasons.",
          },
        ],
      },
    ],
  },
];
