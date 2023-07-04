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
import { Badge, Text } from "@mantine/core";
import { MRT_ColumnDef } from "mantine-react-table";
import { MontaTableActionMenu } from "@/components/ui/table/ActionsMenu";
import { AccessorialCharge } from "@/types/apps/billing";
import { accessorialChargeTableStore } from "@/stores/BillingStores";
import { truncateText } from "@/lib/utils";

export const ACTableColumns = (): MRT_ColumnDef<AccessorialCharge>[] => {
  return [
    {
      accessorKey: "code",
      header: "Code",
    },
    {
      id: "description",
      accessorKey: "description",
      header: "Description",
      Cell: ({ cell }) => truncateText(cell.getValue() as string, 50),
    },
    {
      id: "is_detention",
      accessorFn: (originalRow) =>
        originalRow.is_detention ? "true" : "false",
      header: "Is Detention",
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
      accessorFn: (row) => `${row.charge_amount} ${row.charge_amount_currency}`,
      id: "charge_amount",
      header: "Charge Amount",
      filterVariant: "text",
      sortingFn: "text",
      Cell: ({ renderedCellValue }) => <Text>{renderedCellValue}</Text>,
    },
    {
      id: "actions",
      header: "Actions",
      Cell: ({ row }) => (
        <MontaTableActionMenu
          store={accessorialChargeTableStore}
          name="Accessorial Charge"
          data={row.original}
        />
      ),
    },
  ];
};
