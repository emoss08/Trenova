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
import { Badge, Text } from "@mantine/core";
import { MontaTableActionMenu } from "@/components/ui/table/ActionsMenu";
import { Commodity } from "@/types/apps/commodities";
import { commodityTableStore } from "@/stores/CommodityStore";
import { truncateText } from "@/lib/utils";

export const CommodityTableColumns = (): MRT_ColumnDef<Commodity>[] => {
  return [
    {
      accessorKey: "name",
      header: "Name",
    },
    {
      id: "description",
      accessorKey: "description",
      header: "Description",
      Cell: ({ cell }) => {
        if (cell.getValue()) {
          return truncateText(cell.getValue() as string, 50);
        }
        return <Text>No Description</Text>;
      },
    },
    {
      id: "temp_range",
      accessorFn: (row) => `${row.min_temp} - ${row.max_temp}`,
      header: "Temperature Range",
      Cell: ({ cell }) => {
        if (cell.getValue() === "null - null") {
          return <Text>No Temp Range</Text>;
        }
        return <Text>{cell.getValue() as string}</Text>;
      },
    },
    {
      id: "is_hazmat",
      accessorFn: (originalRow) => (originalRow.is_hazmat ? "true" : "false"),
      header: "Is Hazmat",
      filterFn: "equals",
      Cell: ({ cell }) => (
        <Badge
          color={cell.getValue() === "true" ? "green" : "red"}
          variant="filled"
          radius="xs"
        >
          {cell.getValue() === "true" ? "Yes" : "No"}
        </Badge>
      ),
      mantineFilterSelectProps: {
        data: [
          { value: "", label: "All" },
          { value: "true", label: "Active" },
          { value: "false", label: "Inactive" },
        ] as any,
      },
      filterVariant: "select",
    },
    {
      id: "actions",
      header: "Actions",
      Cell: ({ row }) => (
        <MontaTableActionMenu store={commodityTableStore} data={row.original} />
      ),
    },
  ];
};
