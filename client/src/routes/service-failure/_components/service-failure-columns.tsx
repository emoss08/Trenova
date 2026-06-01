import {
  DataTableDescription,
  DataTableLink,
} from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import {
  findChoice,
  serviceFailureSourceChoices,
  serviceFailureStatusChoices,
  serviceFailureTypeChoices,
  stopTypeChoices,
} from "@/lib/choices";
import type { ServiceFailure } from "@/types/service-failure";
import type { ColumnDef } from "@tanstack/react-table";

function statusBadge(value: ServiceFailure["status"]) {
  const choice = findChoice(serviceFailureStatusChoices, value);
  return choice ? <ColorOptionValue color={choice.color} value={choice.label} /> : value;
}

export function getColumns(): ColumnDef<ServiceFailure>[] {
  return [
    {
      accessorKey: "number",
      header: "Failure",
      cell: ({ row }) => <span className="font-medium">{row.original.number}</span>,
      size: 150,
      minSize: 140,
      maxSize: 180,
      meta: {
        label: "Failure",
        apiField: "number",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => statusBadge(row.original.status),
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: serviceFailureStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => {
        const choice = findChoice(serviceFailureTypeChoices, row.original.type);
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.type
        );
      },
      size: 150,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: "Type",
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: serviceFailureTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "source",
      header: "Source",
      cell: ({ row }) => {
        const choice = findChoice(serviceFailureSourceChoices, row.original.source);
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.source
        );
      },
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        label: "Source",
        apiField: "source",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: serviceFailureSourceChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "shipmentId",
      header: "Shipment",
      cell: ({ row }) => {
        const label =
          row.original.shipment?.proNumber || row.original.shipment?.bol || row.original.shipmentId;
        return (
          <DataTableLink
            text={label}
            href={`/shipment-management/shipments?panelType=edit&panelEntityId=${row.original.shipmentId}`}
          />
        );
      },
      size: 190,
      minSize: 160,
      maxSize: 240,
      meta: {
        label: "Shipment",
        apiField: "shipmentId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "stopType",
      header: "Stop",
      cell: ({ row }) =>
        findChoice(stopTypeChoices, row.original.stopType)?.label ?? row.original.stopType,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        label: "Stop",
        apiField: "stopType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: stopTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "lateMinutes",
      header: "Late",
      cell: ({ row }) => `${row.original.lateMinutes} min`,
      size: 100,
      minSize: 90,
      maxSize: 120,
      meta: {
        label: "Late Minutes",
        apiField: "lateMinutes",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "reasonCode.code",
      header: "Reason",
      cell: ({ row }) =>
        row.original.reasonCode ? (
          <div className="flex min-w-0 flex-col">
            <span className="truncate font-medium">{row.original.reasonCode.code}</span>
            <span className="truncate text-2xs text-muted-foreground">
              {row.original.reasonCode.label}
            </span>
          </div>
        ) : (
          <span className="text-muted-foreground">Unassigned</span>
        ),
      size: 240,
      minSize: 200,
      maxSize: 320,
      meta: {
        label: "Reason",
        apiField: "reasonCodeId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "notes",
      header: "Notes",
      cell: ({ row }) => (
        <DataTableDescription description={row.original.notes} truncateLength={80} />
      ),
      size: 320,
      minSize: 240,
      maxSize: 420,
      meta: {
        label: "Notes",
        apiField: "notes",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "detectedAt",
      header: "Detected",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.detectedAt} />,
      size: 180,
      minSize: 160,
      maxSize: 220,
      meta: {
        label: "Detected",
        apiField: "detectedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
