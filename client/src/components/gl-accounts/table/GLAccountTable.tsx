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
import { generalLedgerTableStore } from "@/stores/AccountingStores";
import { MontaTable } from "@/components/common/table/MontaTable";
import { EditGLAccountModal } from "@/components/gl-accounts/table/EditGLAccountModal";
import { ViewGLAccountModal } from "@/components/gl-accounts/table/ViewGLAccountModal";
import { CreateGLAccountModal } from "@/components/gl-accounts/table/CreateGLAccountModal";
import { GeneralLedgerAccount } from "@/types/accounting";
import { IChoiceProps, StatusChoiceProps } from "@/types";
import { MontaTableActionMenu } from "@/components/common/table/ActionsMenu";
import { truncateText } from "@/helpers/constants";

function StatusBadge({ status }: { status: string }) {
  return (
    <Badge
      color={status === "A" ? "green" : "red"}
      variant="filled"
      radius="xs"
    >
      {status === "A" ? "Active" : "Inactive"}
    </Badge>
  );
}

export function GLAccountTable() {
  const columns = useMemo<MRT_ColumnDef<GeneralLedgerAccount>[]>(
    () => [
      {
        id: "status",
        accessorKey: "status",
        header: "Status",
        filterFn: "equals",
        Cell: ({ cell }) => <StatusBadge status={cell.getValue() as string} />,
        mantineFilterSelectProps: {
          data: [
            { value: "", label: "All" },
            { value: "A", label: "Active" },
            { value: "I", label: "Inactive" },
          ] as ReadonlyArray<IChoiceProps<StatusChoiceProps>>,
        },
        filterVariant: "select",
      },
      {
        accessorKey: "accountNumber",
        header: "Account Number",
      },
      {
        id: "description",
        accessorKey: "description",
        header: "Description",
        Cell: ({ cell }) => truncateText(cell.getValue() as string, 50),
      },
      {
        accessorKey: "accountType",
        header: "Account Type",
      },
      {
        id: "actions",
        header: "Actions",
        Cell: ({ row }) => (
          <MontaTableActionMenu
            store={generalLedgerTableStore}
            data={row.original}
          />
        ),
      },
    ],
    [],
  );

  return (
    <MontaTable
      store={generalLedgerTableStore}
      link="/gl_accounts"
      columns={columns}
      TableEditModal={EditGLAccountModal}
      TableViewModal={ViewGLAccountModal}
      displayDeleteModal
      TableCreateDrawer={CreateGLAccountModal}
      tableQueryKey="gl-account-table-data"
      exportModelName="GeneralLedgerAccount"
      name="GL Account"
    />
  );
}
