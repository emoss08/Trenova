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
import { MontaTable } from "@/components/common/table/MontaTable";
import { USDollarFormat } from "@/helpers/constants";
import { MontaTableActionMenu } from "@/components/common/table/ActionsMenu";
import { Rate } from "@/types/dispatch";
import { useRateStore } from "@/stores/DispatchStore";
import { EditFleetCodeModal } from "@/components/fleet-codes/EditFleetCodeModal";
import { CreateRateModal } from "@/components/rate/CreateRateModal";
import { ViewRateModal } from "@/components/rate/ViewRateModal";

export function RateTable() {
  const columns: MRT_ColumnDef<Rate>[] = useMemo<MRT_ColumnDef<Rate>[]>(
    () => [
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
      {
        id: "actions",
        header: "Actions",
        Cell: ({ row }) => (
          <MontaTableActionMenu store={useRateStore} data={row.original} />
        ),
      },
    ],
    [],
  );

  return (
    <MontaTable
      store={useRateStore}
      link="/rates"
      columns={columns}
      TableEditModal={EditFleetCodeModal}
      TableViewModal={ViewRateModal}
      displayDeleteModal
      deleteKey="id"
      TableCreateDrawer={CreateRateModal}
      tableQueryKey="rate-table-data"
      exportModelName="Rate"
      name="Rate"
    />
  );
}
