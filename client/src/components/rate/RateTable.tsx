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
import { Badge } from "@mantine/core";
import { MontaTable } from "@/components/common/table/MontaTable";
import { Rate } from "@/types/dispatch";
import { useRateStore } from "@/stores/DispatchStore";
import { CreateRateModal } from "@/components/rate/CreateRateModal";
import { RateDrawer } from "@/components/rate/EditRateModal";
import { USDollarFormat } from "@/lib/utils";
import { TChoiceProps } from "@/types";

export function RateTable() {
  const columns = useMemo<MRT_ColumnDef<Rate>[]>(
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
          ] as TChoiceProps[],
        },
        filterVariant: "select",
      },
      {
        accessorKey: "rateNumber",
        header: "Rate Number",
      },
      {
        accessorKey: "effectiveDate",
        header: "Effective Date",
      },
      {
        accessorKey: "expirationDate",
        header: "Expiration Date",
      },
      {
        id: "rateAmount",
        accessorKey: "rateAmount",
        header: "Rate Amount",
        filterVariant: "text",
        sortingFn: "text",
        Cell: ({ cell }) =>
          USDollarFormat(Math.round(cell.getValue() as number)),
      },
    ],
    [],
  );

  return (
    <MontaTable
      store={useRateStore}
      link="/rates"
      columns={columns}
      displayDeleteModal
      deleteKey="id"
      TableCreateDrawer={CreateRateModal}
      TableDrawer={RateDrawer}
      tableQueryKey="rate-table-data"
      exportModelName="Rate"
      name="Rate"
    />
  );
}
