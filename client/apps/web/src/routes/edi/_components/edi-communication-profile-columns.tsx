import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { EDICommunicationProfileRow } from "@/lib/graphql/edi-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { communicationProfileMethods, profileStatusOptions } from "./edi-schemas";

export function getCommunicationProfileColumns(
  t: TranslateFn,
): ColumnDef<EDICommunicationProfileRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.name}</div>
          <div className="text-muted-foreground text-xs">
            {row.original.description || t("No description")}
          </div>
        </div>
      ),
      size: 280,
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
      accessorKey: "method",
      header: t("Method"),
      cell: ({ row }) => <Badge variant="outline">{row.original.method}</Badge>,
      size: 120,
      meta: {
        label: t("Method"),
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
      header: t("Status"),
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      size: 120,
      meta: {
        label: t("Status"),
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
      header: t("Partner"),
      cell: ({ row }) =>
        row.original.partner?.name ?? row.original.ediPartnerId ?? <DataTablePlaceholder />,
      size: 220,
      meta: {
        label: t("Partner"),
        apiField: "ediPartnerId",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "secretState",
      header: t("Secrets"),
      cell: ({ row }) => {
        const secretState = row.original.secretState;

        if (secretState != null && secretState.length > 0) {
          return (
            <div className="flex flex-wrap gap-1">
              {secretState.map((secret) => (
                <Badge key={secret.key} variant="secondary">
                  {secret.key}
                </Badge>
              ))}
            </div>
          );
        }

        return <DataTablePlaceholder />;
      },
      size: 220,
      meta: {
        label: t("Secrets"),
        apiField: "secretState",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "updatedAt",
      header: t("Updated"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt ?? undefined} />,
      size: 180,
      meta: {
        label: t("Updated"),
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
      },
    },
  ];
}
