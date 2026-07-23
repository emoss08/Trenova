import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import { resourceTypeChoices } from "@/lib/choices";
import type { DocumentPacketRule } from "@/types/document-packet-rule";
import type { DocumentType } from "@/types/document-type";
import { type ColumnDef } from "@tanstack/react-table";

export function getColumns(
  documentTypeMap: Map<string, DocumentType>,
): ColumnDef<DocumentPacketRule>[] {
  return [
    {
      accessorKey: "resourceType",
      header: "Resource Type",
      cell: ({ row }) => {
        return <p>{row.original.resourceType}</p>;
      },
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: "Resource Type",
        apiField: "resourceType",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: resourceTypeChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "documentTypeId",
      header: "Document Type",
      cell: ({ row }) => {
        const docType = documentTypeMap.get(row.original.documentTypeId);
        return docType ? (
          <span className="font-medium">{docType.name}</span>
        ) : (
          <span className="text-muted-foreground">
            {row.original.documentTypeId}
          </span>
        );
      },
      size: 200,
      minSize: 150,
      maxSize: 300,
      meta: {
        label: "Document Type",
        apiField: "documentTypeId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "required",
      header: "Required",
      cell: ({ row }) => (
        <Badge variant={row.original.required ? "active" : "outline"}>
          {row.original.required ? "Yes" : "No"}
        </Badge>
      ),
      size: 100,
      minSize: 80,
      maxSize: 120,
      meta: {
        label: "Required",
        apiField: "required",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "allowMultiple",
      header: "Allow Multiple",
      cell: ({ row }) => (
        <Badge variant={row.original.allowMultiple ? "info" : "outline"}>
          {row.original.allowMultiple ? "Yes" : "No"}
        </Badge>
      ),
      size: 130,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: "Allow Multiple",
        apiField: "allowMultiple",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "displayOrder",
      header: "Order",
      cell: ({ row }) => row.original.displayOrder,
      size: 80,
      minSize: 60,
      maxSize: 100,
      meta: {
        label: "Order",
        apiField: "displayOrder",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "expirationRequired",
      header: "Expiration Req.",
      cell: ({ row }) => (
        <Badge
          variant={row.original.expirationRequired ? "warning" : "outline"}
        >
          {row.original.expirationRequired ? "Yes" : "No"}
        </Badge>
      ),
      size: 140,
      minSize: 110,
      maxSize: 170,
      meta: {
        label: "Expiration Required",
        apiField: "expirationRequired",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "expirationWarningDays",
      header: "Warning Days",
      cell: ({ row }) =>
        row.original.expirationRequired ? (
          <span>{row.original.expirationWarningDays}d</span>
        ) : (
          <span className="text-muted-foreground">-</span>
        ),
      size: 120,
      minSize: 90,
      maxSize: 150,
      meta: {
        label: "Warning Days",
        apiField: "expirationWarningDays",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => (
        <HoverCardTimestamp timestamp={row.original.createdAt} />
      ),
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
