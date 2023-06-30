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
import { MRT_ColumnDef } from "mantine-react-table";
import { DivisionCode } from "@/types/apps/accounting";
import { Badge, Button, Menu } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faChevronDown } from "@fortawesome/pro-solid-svg-icons";
import {
  faUser,
  faUserGear,
  faUserMinus,
} from "@fortawesome/pro-duotone-svg-icons";
import { divisionCodeTableStore } from "@/stores/AccountingStores";
import { SelectItem } from "@/components/ui/fields/SelectInput";

export const DCTableColumns = (): MRT_ColumnDef<DivisionCode>[] => {
  return [
    {
      id: "status",
      accessorKey: "status",
      header: "Status",
      filterFn: "equals",
      Cell: ({ cell }) => (
        <Badge
          color={cell.getValue() === "A" ? "green" : "red"}
          variant="filled"
          radius="xs"
        >
          {cell.getValue() === "A" ? "Active" : "Inactive"}
        </Badge>
      ),
      mantineFilterSelectProps: {
        data: [
          { value: "", label: "All" },
          { value: "A", label: "Active" },
          { value: "I", label: "Inactive" },
        ] as SelectItem[],
      },
      filterVariant: "select",
    },
    {
      accessorKey: "code", //access nested data with dot notation
      header: "Code",
    },
    {
      accessorKey: "description",
      header: "Description",
    },
    {
      id: "actions",
      header: "Actions",
      Cell: ({ row }) => (
        <>
          <Menu width={200} shadow="md" withArrow offset={5} position="bottom">
            <Menu.Target>
              <Button
                variant="light"
                color="gray"
                size="xs"
                rightIcon={<FontAwesomeIcon icon={faChevronDown} size="sm" />}
              >
                Actions
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>Division Actions</Menu.Label>
              <Menu.Item
                icon={<FontAwesomeIcon icon={faUser} />}
                onClick={() => {
                  divisionCodeTableStore.set("selectedRecord", row.original);
                  divisionCodeTableStore.set("viewModalOpen", true);
                }}
              >
                View Division Code
              </Menu.Item>
              <Menu.Item
                icon={<FontAwesomeIcon icon={faUserGear} />}
                onClick={() => {
                  divisionCodeTableStore.set("selectedRecord", row.original);
                  divisionCodeTableStore.set("editModalOpen", true);
                }}
              >
                Edit Division Code
              </Menu.Item>
              <Menu.Item
                color="red"
                icon={<FontAwesomeIcon icon={faUserMinus} />}
                onClick={() => {
                  divisionCodeTableStore.set("selectedRecord", row.original);
                  divisionCodeTableStore.set("deleteModalOpen", true);
                }}
              >
                Delete Division Code
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
        </>
      ),
    },
  ];
};
