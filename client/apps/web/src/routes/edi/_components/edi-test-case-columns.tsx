import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import type { EDITestCaseRow } from "@/types/edi";
import type { ColumnDef } from "@tanstack/react-table";

export function getTestCaseColumns(): ColumnDef<EDITestCaseRow>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate font-medium">{row.original.name}</div>
          {row.original.description ? (
            <div className="truncate text-xs text-muted-foreground">{row.original.description}</div>
          ) : null}
        </div>
      ),
      size: 260,
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
      id: "partner",
      header: "Partner",
      cell: ({ row }) =>
        row.original.documentProfile?.partner ? (
          <div className="min-w-0">
            <div className="truncate font-medium">{row.original.documentProfile.partner.name}</div>
            <div className="truncate text-xs text-muted-foreground">
              {row.original.documentProfile.partner.code}
            </div>
          </div>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 220,
      meta: {
        label: "Partner",
        apiField: "partnerDocumentProfileId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "transaction",
      header: "Transaction",
      cell: ({ row }) =>
        row.original.documentProfile ? (
          <div className="flex items-center gap-2">
            <Badge variant="secondary">{row.original.documentProfile.transactionSet}</Badge>
            <Badge variant="outline">{row.original.documentProfile.direction}</Badge>
          </div>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 170,
      meta: {
        label: "Transaction",
        apiField: "partnerDocumentProfileId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "documentProfile",
      header: "Document Profile",
      cell: ({ row }) =>
        row.original.documentProfile?.name ? (
          <span className="truncate">{row.original.documentProfile.name}</span>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 220,
      meta: {
        label: "Document Profile",
        apiField: "partnerDocumentProfileId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "expectations",
      header: "Expected Outcome",
      cell: ({ row }) => {
        const { expectedWarnings, expectedErrors } = row.original;
        if (expectedWarnings === 0 && expectedErrors === 0) {
          return <Badge variant="outline">Clean</Badge>;
        }
        return (
          <div className="flex items-center gap-1.5">
            {expectedWarnings > 0 && (
              <Badge variant="secondary">
                {expectedWarnings} warning{expectedWarnings === 1 ? "" : "s"}
              </Badge>
            )}
            {expectedErrors > 0 && (
              <Badge variant="warning">
                {expectedErrors} error{expectedErrors === 1 ? "" : "s"}
              </Badge>
            )}
          </div>
        );
      },
      size: 180,
      meta: {
        label: "Expected Outcome",
        apiField: "expectedWarnings",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
      cell: ({ row }) =>
        row.original.updatedAt ? (
          <HoverCardTimestamp timestamp={row.original.updatedAt} />
        ) : (
          <DataTablePlaceholder />
        ),
      size: 180,
      meta: {
        label: "Updated",
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
