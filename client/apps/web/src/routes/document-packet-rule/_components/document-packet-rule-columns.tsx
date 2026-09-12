import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import { resourceTypeChoices } from "@/lib/choices";
import type { DocumentPacketRuleRow } from "@/lib/graphql/document-packet-rule-table";
import type { DocumentType } from "@trenova/shared/types/document-type";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(
  documentTypeMap: Map<string, DocumentType>,
  t: TranslateFn,
): ColumnDef<DocumentPacketRuleRow>[] {
  return [
    {
      accessorKey: "resourceType",
      header: t("Resource Type"),
      cell: ({ row }) => {
        return <p>{row.original.resourceType}</p>;
      },
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Resource Type"),
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
      header: t("Document Type"),
      cell: ({ row }) => {
        const docType = documentTypeMap.get(row.original.documentTypeId);
        return docType ? (
          <span className="font-medium">{docType.name}</span>
        ) : (
          <span className="text-muted-foreground">{row.original.documentTypeId}</span>
        );
      },
      size: 200,
      minSize: 150,
      maxSize: 300,
      meta: {
        label: t("Document Type"),
        apiField: "documentTypeId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "required",
      header: t("Required"),
      cell: ({ row }) => (
        <Badge variant={row.original.required ? "active" : "outline"}>
          {row.original.required ? t("Yes") : t("No")}
        </Badge>
      ),
      size: 100,
      minSize: 80,
      maxSize: 120,
      meta: {
        label: t("Required"),
        apiField: "required",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "allowMultiple",
      header: t("Allow Multiple"),
      cell: ({ row }) => (
        <Badge variant={row.original.allowMultiple ? "info" : "outline"}>
          {row.original.allowMultiple ? t("Yes") : t("No")}
        </Badge>
      ),
      size: 130,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: t("Allow Multiple"),
        apiField: "allowMultiple",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "displayOrder",
      header: t("Order"),
      cell: ({ row }) => row.original.displayOrder,
      size: 80,
      minSize: 60,
      maxSize: 100,
      meta: {
        label: t("Order"),
        apiField: "displayOrder",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "expirationRequired",
      header: t("Expiration Req."),
      cell: ({ row }) => (
        <Badge variant={row.original.expirationRequired ? "warning" : "outline"}>
          {row.original.expirationRequired ? t("Yes") : t("No")}
        </Badge>
      ),
      size: 140,
      minSize: 110,
      maxSize: 170,
      meta: {
        label: t("Expiration Required"),
        apiField: "expirationRequired",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "expirationWarningDays",
      header: t("Warning Days"),
      cell: ({ row }) =>
        row.original.expirationRequired ? (
          <span>{t("{0}d", row.original.expirationWarningDays)}</span>
        ) : (
          <span className="text-muted-foreground">-</span>
        ),
      size: 120,
      minSize: 90,
      maxSize: 150,
      meta: {
        label: t("Warning Days"),
        apiField: "expirationWarningDays",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
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
