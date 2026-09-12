import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { BooleanBadge } from "@trenova/shared/components/status-badge";
import {
  findChoice,
  serviceFailureReasonCategoryChoices,
  serviceFailureReasonCodeAppliesToChoices,
} from "@/lib/choices";
import type { ServiceFailureReasonCodeRow } from "@/lib/graphql/service-failure-reason-code-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<ServiceFailureReasonCodeRow>[] {
  return [
    {
      accessorKey: "active",
      header: t("Active"),
      cell: ({ row }) => <BooleanBadge value={row.original.active} />,
      size: 100,
      minSize: 100,
      maxSize: 100,
      meta: {
        label: t("Active"),
        apiField: "active",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => <span className="font-medium">{row.original.code}</span>,
      size: 180,
      minSize: 160,
      maxSize: 220,
      meta: {
        label: t("Code"),
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "label",
      header: t("Label"),
      cell: ({ row }) => row.original.label,
      size: 240,
      minSize: 200,
      maxSize: 320,
      meta: {
        label: t("Label"),
        apiField: "label",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "category",
      header: t("Category"),
      cell: ({ row }) => {
        const choice = findChoice(serviceFailureReasonCategoryChoices, row.original.category);
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.category
        );
      },
      size: 160,
      minSize: 140,
      maxSize: 200,
      meta: {
        label: t("Category"),
        apiField: "category",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: serviceFailureReasonCategoryChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "appliesTo",
      header: t("Applies To"),
      cell: ({ row }) =>
        findChoice(serviceFailureReasonCodeAppliesToChoices, row.original.appliesTo)?.label ??
        row.original.appliesTo,
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Applies To"),
        apiField: "appliesTo",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: serviceFailureReasonCodeAppliesToChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={90} />
      ),
      size: 320,
      minSize: 260,
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
      accessorKey: "defaultStatusCode",
      header: t("X12 Status"),
      cell: ({ row }) => row.original.defaultStatusCode || "-",
      size: 120,
      minSize: 100,
      maxSize: 140,
      meta: {
        label: t("X12 Status"),
        apiField: "defaultStatusCode",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "defaultReasonCode",
      header: t("X12 Reason"),
      cell: ({ row }) => row.original.defaultReasonCode || "-",
      size: 120,
      minSize: 100,
      maxSize: 140,
      meta: {
        label: t("X12 Reason"),
        apiField: "defaultReasonCode",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "sortOrder",
      header: t("Sort"),
      cell: ({ row }) => row.original.sortOrder,
      size: 90,
      minSize: 80,
      maxSize: 110,
      meta: {
        label: t("Sort"),
        apiField: "sortOrder",
        filterable: true,
        sortable: true,
        filterType: "number",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 180,
      minSize: 160,
      maxSize: 220,
      meta: {
        label: t("Created"),
        apiField: "createdAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
