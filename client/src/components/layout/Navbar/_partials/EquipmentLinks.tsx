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
import { faToolbox } from "@fortawesome/pro-duotone-svg-icons";
import {
  LinksGroup,
  LinksGroupProps,
} from "@/components/layout/Navbar/_partials/LinksGroup";

/** Links for Equipment Maintenance Navigation Menu */
export const equipmentNavLinks = [
  {
    label: "Equipment Maintenance",
    icon: faToolbox,
    link: "/",
    permission: "admin.equipment_maintenance.view",
    links: [
      {
        label: "Equipment Maintenance Plan",
        link: "#",
        permission: "view_equipmentmaintenanceplan",
      },
      {
        label: "Configuration Files",
        link: "#", // Placeholder, replace with the actual link
        subLinks: [
          {
            label: "Equipment Types",
            link: "/equipment/equipment-types/",
            permission: "view_equipmenttype",
          },
          {
            label: "Equipment Manufacturers",
            link: "/equipment/equipment-manufacturers/",
            permission: "view_equipmentmanufacturer",
          },
          {
            label: "Tractor",
            link: "/equipment/tractor/",
            permission: "view_tractor",
          },
          {
            label: "Trailer",
            link: "/equipment/trailer/",
            permission: "view_trailer",
          },
        ],
      },
    ],
  },
] satisfies LinksGroupProps[];

export function EquipLinks() {
  const equipmentLinks = equipmentNavLinks.map((item) => (
    <LinksGroup {...item} key={item.label} />
  ));

  return <>{equipmentLinks}</>;
}
