import {
  BooleanBadge,
  DataTableDescription,
  HoverCardTimestamp,
} from "@/components/data-table/_components/data-table-components";
import type {
  DocumentTemplateSchema,
  TemplateStatusSchema,
} from "@/lib/schemas/document-template-schema";
import { truncateText } from "@/lib/utils";
import type { ChoiceProps } from "@/types/common";
import { type ColumnDef } from "@tanstack/react-table";
import { TemplateStatusBadge } from "./template-status-badge";

export const templateStatusChoices = [
  { value: "Draft", label: "Draft", color: "#6b7280" },
  { value: "Active", label: "Active", color: "#22c55e" },
  { value: "Archived", label: "Archived", color: "#ef4444" },
] satisfies ReadonlyArray<ChoiceProps<TemplateStatusSchema>>;

export const pageSizeChoices = [
  { value: "Letter", label: "Letter" },
  { value: "A4", label: "A4" },
  { value: "Legal", label: "Legal" },
] satisfies ReadonlyArray<ChoiceProps<string>>;

export const orientationChoices = [
  { value: "Portrait", label: "Portrait" },
  { value: "Landscape", label: "Landscape" },
] satisfies ReadonlyArray<ChoiceProps<string>>;

export function getColumns(): ColumnDef<DocumentTemplateSchema>[] {
  return [
    {
      id: "code",
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => {
        return (
          <span className="font-mono text-xs font-medium">
            {row.original.code}
          </span>
        );
      },
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        apiField: "code",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => {
        return (
          <span className="font-medium">
            {truncateText(row.original.name, 30)}
          </span>
        );
      },
      size: 250,
      minSize: 200,
      maxSize: 350,
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "documentType",
      accessorKey: "documentType.name",
      header: "Document Type",
      cell: ({ row }) => {
        const docType = row.original.documentType;
        if (!docType) return <span className="text-muted-foreground">-</span>;
        return (
          <div className="flex items-center gap-2">
            {docType.color && (
              <span
                className="size-2 rounded-full"
                style={{ backgroundColor: docType.color }}
              />
            )}
            <span>{docType.name}</span>
          </div>
        );
      },
      size: 180,
      minSize: 150,
      maxSize: 220,
      meta: {
        apiField: "documentType.name",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "status",
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <TemplateStatusBadge status={row.original.status} />,
      size: 120,
      minSize: 100,
      maxSize: 150,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: templateStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "pageSize",
      accessorKey: "pageSize",
      header: "Page Size",
      cell: ({ row }) => <span>{row.original.pageSize}</span>,
      size: 100,
      minSize: 80,
      maxSize: 120,
      meta: {
        apiField: "pageSize",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: pageSizeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "orientation",
      accessorKey: "orientation",
      header: "Orientation",
      cell: ({ row }) => <span>{row.original.orientation}</span>,
      size: 120,
      minSize: 100,
      maxSize: 140,
      meta: {
        apiField: "orientation",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: orientationChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "isDefault",
      accessorKey: "isDefault",
      header: "Default",
      cell: ({ row }) => <BooleanBadge value={row.original.isDefault} />,
      size: 100,
      minSize: 80,
      maxSize: 120,
      meta: {
        apiField: "isDefault",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "description",
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription
          description={row.original.description}
          truncateLength={50}
        />
      ),
      size: 250,
      minSize: 200,
      maxSize: 400,
      meta: {
        apiField: "description",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      id: "createdAt",
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => (
        <HoverCardTimestamp timestamp={row.original.createdAt} />
      ),
      size: 180,
      minSize: 150,
      maxSize: 220,
      meta: {
        apiField: "createdAt",
        label: "Created At",
        filterable: false,
        sortable: true,
      },
    },
  ];
}
