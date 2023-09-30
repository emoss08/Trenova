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

import React from "react";
import { faBoxTaped } from "@fortawesome/pro-duotone-svg-icons";
import {
  LinksGroup,
  LinksGroupProps,
} from "@/components/layout/Navbar/_partials/LinksGroup";

/** Links for Shipment Navigation Menu */
export const shipmentNavLinks = [
  {
    label: "Shipment Management",
    icon: faBoxTaped,
    link: "#",
    links: [
      {
        label: "Shipment Management",
        link: "/shipment-management/",
        permission: "view_order",
      },
      {
        label: "Shipment Controls",
        link: "/admin/control-files#order-controls/",
        permission: "view_ordercontrol",
      },
      {
        label: "Configuration Files",
        link: "#",
        subLinks: [
          {
            label: "Formula Templates",
            link: "/order/formula-template/",
            permission: "view_formulatemplate",
          },
          {
            label: "Shipment Types",
            link: "/shipment-management/order-types/",
            permission: "view_ordertype",
          },
          {
            label: "Movements",
            link: "/shipment-management/movements/",
            permission: "view_movement",
          },
          {
            label: "Qualifier Codes",
            link: "/shipment-management/qualifier-codes/",
            permission: "view_qualifiercode",
          },
          {
            label: "Reason Codes",
            link: "/shipment-management/reason-codes/",
            permission: "view_reasoncode",
          },
        ],
      },
    ],
  },
] satisfies LinksGroupProps[];

export function ShipmentLinks() {
  const links = shipmentNavLinks.map((item) => (
    <LinksGroup {...item} key={item.label} />
  ));

  return <>{links}</>;
}
