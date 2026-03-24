import {
  DataTableColorColumn,
  DataTableDescription,
} from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import {
  documentCategoryChoices,
  documentClassificationChoices,
} from "@/lib/choices";
import type { DocumentType } from "@/types/document-type";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(): ColumnDef<DocumentType>[] {
  return [
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => (
        <DataTableColorColumn
          color={row.original.color ?? undefined}
          text={row.original.code}
        />
      ),
      meta: {
        label: "Code",
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => row.original.name,
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "documentClassification",
      header: "Classification",
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
        label: "Classification",
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
      header: "Category",
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
        label: "Category",
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
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description ?? undefined}
          truncateLength={50}
        />
      ),
      meta: {
        label: "Description",
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
      header: "Created At",
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
