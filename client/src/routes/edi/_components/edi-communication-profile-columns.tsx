import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@/components/ui/badge";
import type { EDICommunicationProfile } from "@/types/edi";
import type { ColumnDef } from "@tanstack/react-table";
import { communicationProfileMethods, profileStatusOptions } from "./edi-schemas";

export function getCommunicationProfileColumns(): ColumnDef<EDICommunicationProfile>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.name}</div>
          <div className="text-xs text-muted-foreground">{row.original.description || "No description"}</div>
        </div>
      ),
      size: 280,
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
      accessorKey: "method",
      header: "Method",
      cell: ({ row }) => <Badge variant="outline">{row.original.method}</Badge>,
      size: 120,
      meta: {
        label: "Method",
        apiField: "method",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: communicationProfileMethods.map((method) => ({
          label: method,
          value: method,
        })),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "outline"}>
          {row.original.status}
        </Badge>
      ),
      size: 120,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: [...profileStatusOptions],
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "partner.name",
      header: "Partner",
      cell: ({ row }) => row.original.partner?.name ?? row.original.ediPartnerId ?? <DataTablePlaceholder />,
      size: 220,
      meta: {
        label: "Partner",
        apiField: "ediPartnerId",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "secretState",
      header: "Secrets",
      cell: ({ row }) =>
        row.original.secretState.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {row.original.secretState.map((secret) => (
              <Badge key={secret.key} variant="secondary">
                {secret.key}
              </Badge>
            ))}
          </div>
        ) : (
          <DataTablePlaceholder />
        ),
      size: 220,
      meta: {
        label: "Secrets",
        apiField: "secretState",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt ?? undefined} />,
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
