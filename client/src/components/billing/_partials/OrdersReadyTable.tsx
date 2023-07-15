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
import { OrdersReadyProps } from "@/types/apps/billing";
import { Badge, Button } from "@mantine/core";
import { TChoiceProps } from "@/types";
import { WebSocketManager } from "@/utils/websockets";
import { billingClientStore } from "@/stores/BillingStores";

interface Props {
  data: OrdersReadyProps[];
  websocketManager: WebSocketManager;
}

export const OrdersReadyTable = ({ data, websocketManager }: Props) => {
  const [, setStep] = billingClientStore.use("step");
  const [approveTransfer, setApproveTransfer] =
    billingClientStore.use("approveTransfer");

  const sendTransferRequest = () => {
    const proNumbers = table
      .getSelectedRowModel()
      .flatRows.map((row) => row.getValue("pro_number"));

    console.log("proNumbers", proNumbers);
    websocketManager.sendJsonMessage("billing_client", {
      action: "billing_queue",
      message: proNumbers,
    });
  };

  React.useEffect(() => {
    if (approveTransfer) {
      billingClientStore.set("invalidOrders", []);
      billingClientStore.set("transferConfirmModalOpen", false);
      setApproveTransfer(false);
      sendTransferRequest();
    }
  }, [approveTransfer]);

  const columns = useMemo<MRT_ColumnDef<OrdersReadyProps>[]>(
    () => [
      {
        accessorKey: "pro_number",
        header: "Pro Number",
        enableClickToCopy: true,
      },
      {
        accessorKey: "customer_name",
        header: "Customer",
      },
      {
        accessorKey: "freight_charge_amount",
        header: "FC Amount",
      },
      {
        accessorKey: "other_charge_amount",
        header: "Other Charge Amount",
      },
      {
        id: "is_missing_documents",
        accessorKey: "is_missing_documents",
        header: "Is Missing Documents",
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
        accessorKey: "sub_total",
        header: "Sub Total",
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
    enablePagination: false,
    enableRowSelection: true,
    mantineTableBodyCellProps: ({ row }) => {
      if (row.getValue("is_missing_documents") === true) {
        return {
          sx: {
            backgroundColor: "rgba(235,52,52,0.25)",
            borderRight: "1px solid rgba(224,224,224,1)",
          },
        };
      } else {
        return {
          sx: {
            backgroundColor: "rgba(98,235,52,0.25)",
            borderRight: "1px solid rgba(224,224,224,1)",
          },
        };
      }
    },
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
        // If any selected row is missing documents and the user hasn't approved the transfer yet, show the modal
        if (
          table
            .getSelectedRowModel()
            .flatRows.some((row) => row.getValue("is_missing_documents")) &&
          !approveTransfer
        ) {
          // Set all invalid orders in state
          billingClientStore.set(
            "invalidOrders",
            table
              .getSelectedRowModel()
              .flatRows.filter((row) => row.getValue("is_missing_documents"))
          );
          billingClientStore.set("transferConfirmModalOpen", true);
          return;
        }

        // If we're here, then either all the rows are valid or the user has confirmed the transfer, so start the transfer
        console.log("sending transfer request");
        sendTransferRequest();
      };

      const restart = () => {
        websocketManager.sendJsonMessage("billing_client", {
          action: "restart",
        });
        setStep(0);
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
            Transfer Order(s)
          </Button>
          <Button color="red" onClick={restart}>
            Start Over
          </Button>
        </div>
      );
    },
  });

  return <MantineReactTable table={table} />;
};
