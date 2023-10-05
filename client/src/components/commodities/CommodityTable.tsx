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

import React, { useMemo } from "react";
import { MRT_ColumnDef } from "mantine-react-table";
import { Badge, Text } from "@mantine/core";
import { MontaTable } from "@/components/common/table/MontaTable";
import { commodityTableStore } from "@/stores/CommodityStore";
import { CreateCommodityModal } from "@/components/commodities/CreateCommodityModal";
import { Commodity } from "@/types/commodities";
import { truncateText } from "@/lib/utils";
import { TChoiceProps } from "@/types";
import { CommodityDrawer } from "@/components/commodities/CommodityDrawer";

export function CommodityTable() {
  const columns = useMemo<MRT_ColumnDef<Commodity>[]>(
    () => [
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
          ] satisfies ReadonlyArray<TChoiceProps>,
        },
        filterVariant: "select",
      },
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
        accessorFn: (row) => `${row.minTemp} - ${row.maxTemp}`,
        header: "Temperature Range",
        Cell: ({ cell }) => {
          if (cell.getValue() === "null - null") {
            return <Text>No Temp Range</Text>;
          }
          return <Text>{cell.getValue() as string}</Text>;
        },
      },
      {
        id: "isHazmat",
        header: "Is Hazmat",
        accessorKey: "isHazmat",
        filterFn: "equals",
        Cell: ({ cell }) => (
          <Badge
            color={cell.getValue() === "N" ? "green" : "red"}
            variant="filled"
            radius="xs"
          >
            {cell.getValue() === "Y" ? "Yes" : "No"}
          </Badge>
        ),
        mantineFilterSelectProps: {
          data: [
            { value: "", label: "All" },
            { value: "Y", label: "Yes" },
            { value: "N", label: "No" },
          ] as ReadonlyArray<TChoiceProps>,
        },
        filterVariant: "select",
      },
    ],
    [],
  );

  return (
    <MontaTable
      store={commodityTableStore}
      link="/commodities"
      columns={columns}
      displayDeleteModal
      TableCreateDrawer={CreateCommodityModal}
      TableDrawer={CommodityDrawer}
      tableQueryKey="commodity-table-data"
      exportModelName="Commodity"
      name="Commodity"
    />
  );
}
