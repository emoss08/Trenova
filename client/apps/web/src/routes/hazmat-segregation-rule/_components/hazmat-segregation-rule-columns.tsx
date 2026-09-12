import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { BooleanBadge } from "@trenova/shared/components/status-badge";
import {
  hazardousClassChoices,
  segregationDistanceUnitChoices,
  segregationTypeChoices,
  statusChoices,
} from "@/lib/choices";
import type { HazmatSegregationRule } from "@/types/hazmat-segregation-rule";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<HazmatSegregationRule>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const choice = statusChoices.find((c) => c.value === row.original.status);
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.status
        );
      },
      size: 120,
      minSize: 120,
      maxSize: 160,
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
      accessorKey: "name",
      header: t("Name"),
      size: 260,
      minSize: 220,
      maxSize: 320,
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
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={100} />
      ),
      size: 320,
      minSize: 280,
      maxSize: 420,
      meta: {
        label: t("Description"),
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "classA",
      header: t("Class A"),
      cell: ({ row }) => {
        const choice = hazardousClassChoices.find((c) => c.value === row.original.classA);
        return choice ? choice.label : row.original.classA;
      },
      size: 240,
      minSize: 220,
      maxSize: 280,
      meta: {
        label: t("Class A"),
        apiField: "classA",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: hazardousClassChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "classB",
      header: t("Class B"),
      cell: ({ row }) => {
        const choice = hazardousClassChoices.find((c) => c.value === row.original.classB);
        return choice ? choice.label : row.original.classB;
      },
      size: 240,
      minSize: 220,
      maxSize: 280,
      meta: {
        label: t("Class B"),
        apiField: "classB",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: hazardousClassChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "segregationType",
      header: t("Segregation Type"),
      cell: ({ row }) => {
        const choice = segregationTypeChoices.find((c) => c.value === row.original.segregationType);
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.segregationType
        );
      },
      size: 180,
      minSize: 160,
      maxSize: 220,
      meta: {
        label: t("Segregation Type"),
        apiField: "segregationType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: segregationTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "minimumDistance",
      header: t("Min Distance"),
      cell: ({ row }) => {
        if (typeof row.original.minimumDistance !== "number") {
          return "-";
        }

        const unit = segregationDistanceUnitChoices.find(
          (c) => c.value === row.original.distanceUnit,
        )?.label;

        return `${row.original.minimumDistance}${unit ? ` ${unit}` : ""}`;
      },
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: t("Minimum Distance"),
        apiField: "minimumDistance",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "gte",
      },
    },
    {
      accessorKey: "hasExceptions",
      header: t("Has Exceptions"),
      cell: ({ row }) => <BooleanBadge value={row.original.hasExceptions} />,
      size: 140,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: t("Has Exceptions"),
        apiField: "hasExceptions",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "updatedAt",
      header: t("Updated At"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt} />,
      size: 180,
      minSize: 160,
      maxSize: 220,
      meta: {
        label: t("Updated At"),
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
