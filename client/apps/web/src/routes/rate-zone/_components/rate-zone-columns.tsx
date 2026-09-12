import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { RateZoneRow } from "@/lib/graphql/rate-tables";

export function getColumns(t: TranslateFn): ColumnDef<RateZoneRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const choice = statusChoices.find((option) => option.value === row.original.status);

        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.status
        );
      },
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: t("Status"),
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => <span className="font-mono text-xs">{row.original.code}</span>,
      size: 120,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: t("Code"),
        apiField: "code",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      size: 240,
      minSize: 200,
      maxSize: 320,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => <DataTableDescription description={t(row.original.description)} />,
      size: 320,
      minSize: 220,
      maxSize: 440,
      meta: {
        label: t("Description"),
        apiField: "description",
        filterable: true,
        sortable: false,
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 150,
      minSize: 130,
      maxSize: 190,
      meta: {
        label: t("Created"),
        apiField: "createdAt",
        filterable: true,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
