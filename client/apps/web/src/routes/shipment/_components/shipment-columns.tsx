import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { ShipmentTenderStatusBadge } from "@trenova/shared/components/status-badge";
import { shipmentStatusChoices, shipmentTenderStatusChoices } from "@/lib/choices";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import type { Customer } from "@trenova/shared/types/customer";
import type { RowAction } from "@trenova/shared/types/data-table";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";
import { type ColumnDef } from "@tanstack/react-table";
import { Link } from "react-router";
import { ActionsCell } from "./command-center/cells/actions-cell";
import { DriverCell } from "./command-center/cells/driver-cell";
import { EtaCell } from "./command-center/cells/eta-cell";
import { LaneCell } from "./command-center/cells/lane-cell";
import { MarginCell } from "./command-center/cells/margin-cell";
import { RevenueCell } from "./command-center/cells/revenue-cell";
import { StatusCell } from "./command-center/cells/status-cell";

function formatAppointment(stop: Stop | null) {
  const appointment = getAppointmentStop(stop);
  if (!appointment?.scheduledWindowStart) return "—";

  const start = formatToUserTimezone(appointment.scheduledWindowStart, {
    showTimeZone: false,
    showSeconds: false,
  });

  if (!appointment.scheduledWindowEnd) return start;

  const end = formatToUserTimezone(appointment.scheduledWindowEnd, {
    showTimeZone: false,
    showSeconds: false,
  });

  return `${start} - ${end}`;
}

function getAppointmentStop(stop: Stop | null) {
  return stop?.scheduleType === "Appointment" ? stop : null;
}

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
      id: "tenderStatus",
      accessorKey: "tenderStatus",
      header: "Tender",
      cell: ({ row }) => {
        const status = row.original.tenderStatus;

        if (status === null) {
          return "-";
        }

        return <ShipmentTenderStatusBadge status={row.original.tenderStatus} />;
      },
      meta: {
        apiField: "tenderStatus",
        label: "Tender Status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: shipmentTenderStatusChoices,
        defaultFilterOperator: "eq",
      },
      size: 130,
      minSize: 120,
      maxSize: 170,
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
      id: "order",
      accessorKey: "orderNumber",
      header: "Order",
      cell: ({ row }) => {
        const { orderId, orderNumber } = row.original;
        if (!orderId) return "—";
        return (
          <Link
            to={`/shipment-management/orders?panelType=edit&panelEntityId=${orderId}`}
            className="truncate font-table text-[11.5px] tabular-nums hover:underline"
            onClick={(event) => event.stopPropagation()}
          >
            {orderNumber || orderId.slice(0, 12)}
          </Link>
        );
      },
      size: 130,
      minSize: 110,
      maxSize: 180,
      meta: {
        label: "Order",
        apiField: "orderId",
        filterable: false,
        sortable: false,
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
      id: "pickupAppointment",
      header: "Pickup Appt",
      accessorFn: (row) => getAppointmentStop(getOriginStop(row))?.scheduledWindowStart ?? null,
      cell: ({ row }) => (
        <span className="font-table text-[11.5px] tabular-nums">
          {formatAppointment(getOriginStop(row.original))}
        </span>
      ),
      size: 170,
      minSize: 150,
      maxSize: 220,
      meta: {
        apiField: "pickupAppointment.scheduledWindowStart",
        label: "Pickup Appointment",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "deliveryAppointment",
      header: "Delivery Appt",
      accessorFn: (row) =>
        getAppointmentStop(getDestinationStop(row))?.scheduledWindowStart ?? null,
      cell: ({ row }) => (
        <span className="font-table text-[11.5px] tabular-nums">
          {formatAppointment(getDestinationStop(row.original))}
        </span>
      ),
      size: 170,
      minSize: 150,
      maxSize: 220,
      meta: {
        apiField: "deliveryAppointment.scheduledWindowStart",
        label: "Delivery Appointment",
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
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
      id: "margin",
      header: () => <div className="text-right">Margin</div>,
      accessorFn: (row) => row.profitabilityEstimate?.marginPercent ?? null,
      cell: ({ row }) => <MarginCell shipment={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: "Margin",
        sortable: false,
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
