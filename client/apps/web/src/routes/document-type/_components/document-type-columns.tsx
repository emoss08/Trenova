import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { documentCategoryChoices, documentClassificationChoices } from "@/lib/choices";
import type { DocumentType } from "@trenova/shared/types/document-type";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<DocumentType>[] {
  return [
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => (
        <DataTableColorColumn color={row.original.color ?? undefined} text={row.original.code} />
      ),
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
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => row.original.name,
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
      accessorKey: "documentClassification",
      header: t("Classification"),
      cell: ({ row }) => {
        const choice = documentClassificationChoices.find(
          (c) => c.value === row.original.documentClassification,
        );
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.documentClassification
        );
      },
      size: 180,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: t("Classification"),
        apiField: "documentClassification",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: documentClassificationChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "documentCategory",
      header: t("Category"),
      cell: ({ row }) => {
        const choice = documentCategoryChoices.find(
          (c) => c.value === row.original.documentCategory,
        );
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.documentCategory
        );
      },
      size: 180,
      minSize: 140,
      maxSize: 220,
      meta: {
        label: t("Category"),
        apiField: "documentCategory",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: documentCategoryChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description ?? undefined}
          truncateLength={50}
        />
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
