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
import {
  MantineReactTable,
  MRT_ColumnDef,
  useMantineReactTable,
} from "mantine-react-table";
import { BillingQueue } from "@/types/apps/billing";
import { Badge, Button, Stack } from "@mantine/core";
import { TChoiceProps } from "@/types";
import { WebSocketManager } from "@/utils/websockets";
import { billingClientStore } from "@/stores/BillingStores";

interface Props {
  data: BillingQueue[];
  websocketManager: WebSocketManager;
}

export const BillingQueueTable = ({ data, websocketManager }: Props) => {
  const [, setStep] = billingClientStore.use("step");

  // Add up the total amounts of each row
  const totalAmount = useMemo(
    () => data.reduce((acc, row) => acc + parseFloat(row.total_amount), 0),
    [data]
  );

  const columns = useMemo<MRT_ColumnDef<BillingQueue>[]>(
    () => [
      {
        accessorKey: "invoice_number",
        header: "Invoice Number",
        enableClickToCopy: true,
      },
      {
        accessorKey: "customer_name",
        header: "Customer",
      },
      {
        accessorKey: "other_charge_total",
        header: "Other Charge Total",
      },
      {
        id: "is_summary",
        accessorKey: "is_summary",
        header: "Is Summary Invoice?",
        filterFn: "equals",
        Cell: ({ cell }) => (
          <Badge
            color={cell.getValue() === true ? "red" : "green"}
            variant="filled"
            radius="xs"
          >
            {cell.getValue() === true ? "Yes" : "No"}
          </Badge>
        ),
        mantineFilterSelectProps: {
          data: [
            { value: "", label: "All" },
            { value: "true", label: "Yes" },
            { value: "false", label: "No" },
          ] as TChoiceProps[],
        },
        filterVariant: "select",
      },
      {
        accessorKey: "total_amount",
        header: "Sub Total",
        Cell: ({ cell }) => {
          const value = parseFloat(cell.getValue<string>());
          return (
            <>
              {value?.toLocaleString?.("en-US", {
                style: "currency",
                currency: "USD",
                minimumFractionDigits: 2,
                maximumFractionDigits: 2,
              })}
            </>
          );
        },
        Footer: () => (
          <Stack>
            <div>Sub Total:</div>
            {totalAmount?.toLocaleString?.("en-US", {
              style: "currency",
              currency: "USD",
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
            })}
          </Stack>
        ),
      },
    ],
    []
  );

  if (!data) return <div>Loading...</div>;

  const table = useMantineReactTable({
    columns,
    data,
    enableColumnFilterModes: false,
    enableColumnOrdering: true,
    enableFacetedValues: true,
    enableGrouping: true,
    enableColumnFilters: true,
    enablePinning: true,
    enablePagination: true,
    enableRowSelection: true,
    initialState: {
      showColumnFilters: false,
      density: "xs",
    },
    mantineFilterTextInputProps: {
      sx: { borderBottom: "unset", marginTop: "8px" },
      variant: "filled",
    },
    mantineFilterSelectProps: {
      sx: { borderBottom: "unset", marginTop: "8px" },
      variant: "filled",
    },
    positionToolbarAlertBanner: "bottom",
    renderDetailPanel: ({ row }) => (
      <div style={{ padding: "16px" }}>
        <pre>{JSON.stringify(row.original, null, 2)}</pre>
      </div>
    ),
    renderTopToolbarCustomActions: ({ table }) => {
      const handleTransfer = () => {
        const invoiceNumbers = table
          .getSelectedRowModel()
          .flatRows.map((row) => row.getValue("invoice_number"));

        websocketManager.sendJsonMessage("billing_client", {
          action: "bill_orders",
          payload: invoiceNumbers,
        });
      };

      const restart = () => {
        websocketManager.sendJsonMessage("billing_client", {
          action: "orders_ready",
        });
        setStep(1);
      };

      return (
        <div style={{ display: "flex", gap: "8px" }}>
          <Button
            color="green"
            onClick={handleTransfer}
            disabled={
              !Object.keys(table.getState().rowSelection).some(
                (key) => table.getState().rowSelection[key]
              )
            }
            variant="filled"
          >
            Bill Order(s)
          </Button>
          <Button color="violet" onClick={restart}>
            Go Back
          </Button>
        </div>
      );
    },
  });

  return <MantineReactTable table={table} />;
};
