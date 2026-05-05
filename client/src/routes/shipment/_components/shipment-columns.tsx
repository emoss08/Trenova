import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { shipmentStatusChoices } from "@/lib/choices";
import type { Customer } from "@/types/customer";
import type { RowAction } from "@/types/data-table";
import type { Shipment } from "@/types/shipment";
import { type ColumnDef } from "@tanstack/react-table";
import { ActionsCell } from "./command-center/cells/actions-cell";
import { DriverCell } from "./command-center/cells/driver-cell";
import { EtaCell } from "./command-center/cells/eta-cell";
import { LaneCell } from "./command-center/cells/lane-cell";
import { RevenueCell } from "./command-center/cells/revenue-cell";
import { StatusCell } from "./command-center/cells/status-cell";

export function getColumns(rowActions: RowAction<Shipment>[]): ColumnDef<Shipment>[] {
  return [
    {
      id: "lane",
      header: "Lane",
      accessorFn: () => null,
      cell: ({ row }) => <LaneCell shipment={row.original} />,
      size: 280,
      minSize: 240,
      maxSize: 360,
      meta: { label: "Lane", sortable: false, filterable: false },
    },
    {
      id: "status",
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusCell shipment={row.original} />,
      meta: {
        apiField: "status",
        label: "Status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: shipmentStatusChoices,
        defaultFilterOperator: "eq",
      },
      size: 160,
      minSize: 140,
      maxSize: 200,
    },
    {
      id: "proBol",
      header: "PRO / BOL",
      accessorFn: (row) => row.proNumber ?? row.bol ?? "",
      cell: ({ row }) => (
        <div className="flex flex-col gap-0.5">
          <span className="truncate font-table text-[11.5px] font-semibold tabular-nums">
            {row.original.proNumber || "—"}
          </span>
          <span className="truncate font-table text-[10px] text-muted-foreground tabular-nums">
            {row.original.bol || "—"}
          </span>
        </div>
      ),
      size: 160,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: "PRO Number",
        apiField: "proNumber",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "customer",
      accessorKey: "customer",
      header: "Customer",
      size: 220,
      minSize: 180,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer, weight } = row.original;
        if (!customer) {
          return <p className="text-muted-foreground">—</p>;
        }
        return (
          <div className="flex min-w-0 flex-col gap-0.5">
            <EntityRefCell<Customer, Shipment>
              entity={customer}
              config={{
                basePath: "/billing/configuration-files/customers",
                getId: (c) => c.id,
                getDisplayText: (c) => c.name,
                getHeaderText: "Customer",
              }}
              parent={row.original}
            />
            {typeof weight === "number" && weight > 0 && (
              <span className="font-table text-[10px] text-muted-foreground tabular-nums">
                {weight.toLocaleString()} lb
              </span>
            )}
          </div>
        );
      },
      meta: {
        apiField: "customer.name",
        label: "Customer Name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "driver",
      header: "Driver / Equip",
      accessorFn: () => null,
      cell: ({ row }) => <DriverCell shipment={row.original} />,
      size: 200,
      minSize: 160,
      maxSize: 260,
      meta: { label: "Driver / Equip", sortable: false, filterable: false },
    },
    {
      id: "eta",
      header: "ETA",
      accessorFn: () => null,
      cell: ({ row }) => <EtaCell shipment={row.original} />,
      size: 160,
      minSize: 140,
      maxSize: 200,
      meta: { label: "ETA", sortable: false, filterable: false },
    },
    {
      id: "revenue",
      header: () => <div className="text-right">Revenue</div>,
      accessorKey: "totalChargeAmount",
      cell: ({ row }) => <RevenueCell shipment={row.original} />,
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: "Revenue",
        apiField: "totalChargeAmount",
        sortable: true,
        filterable: false,
      },
    },
    {
      id: "actions",
      header: () => <span className="sr-only">Actions</span>,
      cell: ({ row }) => <ActionsCell row={row} actions={rowActions} />,
      size: 56,
      minSize: 56,
      maxSize: 56,
      enableHiding: false,
      meta: { label: "Actions", sortable: false, filterable: false },
    },
  ];
}
