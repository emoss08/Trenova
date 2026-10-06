import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { EntityRefCell } from "@/components/data-table/_components/entity-ref-link";
import { ShipmentTenderStatusBadge } from "@trenova/shared/components/status-badge";
import {
  shipmentBillingStatusChoices,
  shipmentStatusChoices,
  shipmentTenderStatusChoices,
} from "@/lib/choices";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import type { Customer } from "@trenova/shared/types/customer";
import type { RowAction, ColumnDef } from "@trenova/shared/types/data-table";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";
import { Link } from "react-router";
import { ActionsCell } from "./cells/actions-cell";
import { BillingCell } from "./cells/billing-cell";
import { CoverageCell } from "./cells/coverage-cell";
import { EtaCell } from "./cells/eta-cell";
import { LaneCell } from "./cells/lane-cell";
import { MarginCell } from "./cells/margin-cell";
import { RevenueCell } from "./cells/revenue-cell";
import { StatusCell } from "./cells/status-cell";
import { recordPath } from "@/config/record-links";
import { resolveCoverage } from "@/lib/shipment-board/coverage";

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

export type ShipmentColumnsParams = {
  rowActions: RowAction<Shipment>[];
  t: TranslateFn;
  expandedRowId: string | null;
  onToggleExpanded: (rowId: string) => void;
};

/** Hidden until a dispatcher asks for them from the Columns menu. */
export const SHIPMENT_HIDDEN_COLUMNS: Record<string, boolean> = {
  pickupAppointment: false,
  deliveryAppointment: false,
};

export function getColumns({
  rowActions,
  t,
  expandedRowId,
  onToggleExpanded,
}: ShipmentColumnsParams): ColumnDef<Shipment>[] {
  return [
    {
      id: "lane",
      header: t("Lane"),
      accessorFn: () => null,
      cell: ({ row }) => <LaneCell shipment={row.original} />,
      size: 280,
      minSize: 240,
      maxSize: 360,
      enableHiding: false,
      meta: {
        label: t("Lane"),
        sortable: false,
        filterable: false,
        exportValue: (row: Shipment) =>
          [getOriginStop(row)?.location?.city, getDestinationStop(row)?.location?.city]
            .filter(Boolean)
            .join(" → "),
      },
    },
    {
      id: "status",
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <StatusCell shipment={row.original} />,
      meta: {
        apiField: "status",
        label: t("Status"),
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
      header: t("Tender"),
      cell: ({ row }) => {
        const status = row.original.tenderStatus;

        if (status === null) {
          return "—";
        }

        return <ShipmentTenderStatusBadge status={row.original.tenderStatus} />;
      },
      meta: {
        apiField: "tenderStatus",
        label: t("Tender status"),
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
      id: "billing",
      accessorKey: "billingTransferStatus",
      header: t("Billing"),
      cell: ({ row }) => <BillingCell shipment={row.original} />,
      meta: {
        apiField: "billingTransferStatus",
        label: "Billing",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: shipmentBillingStatusChoices,
        defaultFilterOperator: "eq",
      },
      size: 140,
      minSize: 120,
      maxSize: 180,
    },
    {
      id: "proBol",
      header: t("PRO / BOL"),
      accessorFn: (row) => row.proNumber ?? row.bol ?? "",
      cell: ({ row }) => (
        <div className="flex flex-col gap-0.5">
          <span className="font-table truncate font-medium tabular-nums">
            {row.original.proNumber || "—"}
          </span>
          <span className="font-table text-muted-foreground truncate text-2xs tabular-nums">
            {row.original.bol || "—"}
          </span>
        </div>
      ),
      size: 160,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: t("PRO number"),
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
      header: t("Order"),
      cell: ({ row }) => {
        const { orderId, orderNumber } = row.original;
        if (!orderId) return "—";
        return (
          <Link
            to={recordPath("order", orderId)}
            className="font-table truncate tabular-nums hover:underline"
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
        label: t("Order"),
        apiField: "orderId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "customer",
      accessorKey: "customer",
      header: t("Customer"),
      size: 220,
      minSize: 180,
      maxSize: 300,
      cell: ({ row }) => {
        const { customer, weight, commodities } = row.original;
        const commodity = commodities?.[0]?.commodity?.name;
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
              <span className="text-muted-foreground in-data-[density=compact]:hidden truncate font-mono text-xs tabular-nums">
                {[t("{0} lb", weight.toLocaleString()), commodity].filter(Boolean).join(" · ")}
              </span>
            )}
          </div>
        );
      },
      meta: {
        apiField: "customer.name",
        label: t("Customer name"),
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "driver",
      header: t("Coverage"),
      accessorFn: () => null,
      cell: ({ row }) => <CoverageCell shipment={row.original} />,
      size: 200,
      minSize: 160,
      maxSize: 260,
      meta: {
        label: t("Coverage"),
        sortable: false,
        filterable: false,
        exportValue: (row: Shipment) => {
          const coverage = resolveCoverage(row);
          return "name" in coverage ? coverage.name : "";
        },
      },
    },
    {
      id: "eta",
      header: t("ETA"),
      accessorFn: () => null,
      cell: ({ row }) => <EtaCell shipment={row.original} />,
      size: 150,
      minSize: 130,
      maxSize: 200,
      meta: {
        label: t("ETA"),
        apiField: "deliveryAppointment.scheduledWindowStart",
        sortable: true,
        filterable: false,
      },
    },
    {
      id: "pickupAppointment",
      header: t("Pickup appt"),
      accessorFn: (row) => getAppointmentStop(getOriginStop(row))?.scheduledWindowStart ?? null,
      cell: ({ row }) => (
        <span className="font-table tabular-nums">
          {formatAppointment(getOriginStop(row.original))}
        </span>
      ),
      size: 170,
      minSize: 150,
      maxSize: 220,
      meta: {
        apiField: "pickupAppointment.scheduledWindowStart",
        label: t("Pickup appointment"),
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "deliveryAppointment",
      header: t("Delivery appt"),
      accessorFn: (row) =>
        getAppointmentStop(getDestinationStop(row))?.scheduledWindowStart ?? null,
      cell: ({ row }) => (
        <span className="font-table tabular-nums">
          {formatAppointment(getDestinationStop(row.original))}
        </span>
      ),
      size: 170,
      minSize: 150,
      maxSize: 220,
      meta: {
        apiField: "deliveryAppointment.scheduledWindowStart",
        label: t("Delivery appointment"),
        filterable: true,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
    {
      id: "revenue",
      header: () => <div className="text-right">{t("Revenue")}</div>,
      accessorKey: "totalChargeAmount",
      cell: ({ row }) => <RevenueCell shipment={row.original} />,
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Revenue"),
        apiField: "totalChargeAmount",
        sortable: true,
        filterable: false,
      },
    },
    {
      id: "margin",
      header: () => <div className="text-right">{t("Margin")}</div>,
      accessorFn: (row) => row.profitabilityEstimate?.marginPercent ?? null,
      cell: ({ row }) => <MarginCell shipment={row.original} />,
      size: 120,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: t("Margin"),
        sortable: false,
        filterable: false,
      },
    },
    {
      id: "actions",
      header: () => <span className="sr-only">{t("Actions")}</span>,
      cell: ({ row }) => (
        <ActionsCell
          row={row}
          actions={rowActions}
          expanded={expandedRowId === row.original.id}
          onToggleExpanded={() => row.original.id && onToggleExpanded(row.original.id)}
        />
      ),
      size: 72,
      minSize: 72,
      maxSize: 72,
      enableHiding: false,
      meta: { label: t("Actions"), sortable: false, filterable: false },
    },
  ];
}
