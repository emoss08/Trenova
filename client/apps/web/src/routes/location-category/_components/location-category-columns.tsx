import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { locationCategoryTypeChoices } from "@/lib/choices";
import type { LocationCategory } from "@/types/location-category";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<LocationCategory>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <ColorOptionValue color={row.original.color ?? ""} value={row.original.name} />
      ),
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "type",
      header: t("Type"),
      cell: ({ row }) => {
        const choice = locationCategoryTypeChoices.find((c) => c.value === row.original.type);
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.type
        );
      },
      size: 180,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: t("Type"),
        apiField: "type",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: locationCategoryTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={50} />
      ),
      meta: {
        label: t("Description"),
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
      size: 250,
      minSize: 150,
      maxSize: 400,
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => {
        return <HoverCardTimestamp timestamp={row.original.createdAt} />;
      },
      meta: {
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
      size: 200,
      minSize: 200,
      maxSize: 250,
    },
  ];
}
