import type { TranslateFn } from "@trenova/shared/i18n/use-t";
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
import type { ServiceFailureRow } from "@/lib/graphql/service-failure-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

function statusBadge(value: ServiceFailureRow["status"]) {
  const choice = findChoice(serviceFailureStatusChoices, value);
  return choice ? <ColorOptionValue color={choice.color} value={choice.label} /> : value;
}

export function getColumns(t: TranslateFn): ColumnDef<ServiceFailureRow>[] {
  return [
    {
      accessorKey: "number",
      header: t("Failure"),
      cell: ({ row }) => <span className="font-medium">{row.original.number}</span>,
      size: 150,
      minSize: 140,
      maxSize: 180,
      meta: {
        label: t("Failure"),
        apiField: "number",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => statusBadge(row.original.status),
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Status"),
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
      header: t("Type"),
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
        label: t("Type"),
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
      header: t("Source"),
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
        label: t("Source"),
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
      header: t("Shipment"),
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
        label: t("Shipment"),
        apiField: "shipmentId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "stopType",
      header: t("Stop"),
      cell: ({ row }) =>
        findChoice(stopTypeChoices, row.original.stopType)?.label ?? row.original.stopType,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        label: t("Stop"),
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
      header: t("Late"),
      cell: ({ row }) => `${row.original.lateMinutes} min`,
      size: 100,
      minSize: 90,
      maxSize: 120,
      meta: {
        label: t("Late Minutes"),
        apiField: "lateMinutes",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "reasonCode.code",
      header: t("Reason"),
      cell: ({ row }) =>
        row.original.reasonCode ? (
          <div className="flex min-w-0 flex-col">
            <span className="truncate font-medium">{row.original.reasonCode.code}</span>
            <span className="text-2xs text-muted-foreground truncate">
              {t(row.original.reasonCode.label)}
            </span>
          </div>
        ) : (
          <span className="text-muted-foreground">{t("Unassigned")}</span>
        ),
      size: 240,
      minSize: 200,
      maxSize: 320,
      meta: {
        label: t("Reason"),
        apiField: "reasonCodeId",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "notes",
      header: t("Notes"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.notes} truncateLength={80} />
      ),
      size: 320,
      minSize: 240,
      maxSize: 420,
      meta: {
        label: t("Notes"),
        apiField: "notes",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "detectedAt",
      header: t("Detected"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.detectedAt} />,
      size: 180,
      minSize: 160,
      maxSize: 220,
      meta: {
        label: t("Detected"),
        apiField: "detectedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
