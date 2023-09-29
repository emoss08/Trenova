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
import { truncateText } from "@/lib/utils";
import { TChoiceProps } from "@/types";
import { FleetCode } from "@/types/dispatch";
import { useFleetCodeStore } from "@/stores/DispatchStore";
import { CreateFleetCodeModal } from "@/components/fleet-codes/CreateFleetCodeModal";
import { ViewFleetCodeModal } from "@/components/fleet-codes/ViewFleetCodeModal";
import { EditFleetCodeModal } from "@/components/fleet-codes/EditFleetCodeModal";

export function FleetCodeTable() {
  const columns: MRT_ColumnDef<FleetCode>[] = useMemo<
    MRT_ColumnDef<FleetCode>[]
  >(
    () => [
      {
        accessorKey: "code",
        header: "Code",
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
        id: "isActive",
        accessorKey: "isActive",
        header: "Is Active",
        Cell: ({ cell }) => (
          <Badge
            color={cell.getValue() === true ? "green" : "red"}
            variant="filled"
            radius="xs"
          >
            {cell.getValue() === true ? "Yes" : "No"}
          </Badge>
        ),
        mantineFilterSelectProps: {
          data: [
            { value: "", label: "All" },
            { value: true, label: "Yes" },
            { value: false, label: "No" },
          ] as ReadonlyArray<TChoiceProps>,
        },
        filterVariant: "select",
      },
    ],
    [],
  );

  return (
    <MontaTable
      store={useFleetCodeStore}
      link="/fleet_codes"
      columns={columns}
      TableEditModal={EditFleetCodeModal}
      TableViewModal={ViewFleetCodeModal}
      displayDeleteModal
      deleteKey="code"
      TableCreateDrawer={CreateFleetCodeModal}
      tableQueryKey="fleet-code-table-data"
      exportModelName="FleetCode"
      name="Fleet Code"
    />
  );
}
