import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { EDIPartnerReadinessBadge, StatusBadge } from "@trenova/shared/components/status-badge";
import { useQuery } from "@tanstack/react-query";
import { partnerReadinessQueryOptions } from "./edi-partner-readiness";
import { Badge } from "@trenova/shared/components/ui/badge";
import { statusChoices } from "@/lib/choices";
import type { EDIPartner } from "@trenova/shared/types/edi";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getPartnerColumns(t: TranslateFn): ColumnDef<EDIPartner>[] {
  return [
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => <span className="font-medium">{row.original.code}</span>,
      size: 140,
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
      size: 220,
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
      accessorKey: "internalOrganization.name",
      header: t("Target Organization"),
      cell: ({ row }) =>
        row.original.internalOrganization?.name ??
        row.original.internalOrganizationId ?? <DataTablePlaceholder />,
      size: 240,
      meta: {
        label: t("Target Organization"),
        apiField: "internalOrganizationId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "direction",
      header: t("Direction"),
      cell: ({ row }) => (
        <div className="flex gap-1">
          <Badge variant={row.original.enabledForInbound ? "secondary" : "outline"}>
            {t("Inbound")}
          </Badge>
          <Badge variant={row.original.enabledForOutbound ? "secondary" : "outline"}>
            {t("Outbound")}
          </Badge>
        </div>
      ),
      size: 180,
      meta: {
        label: t("Direction"),
        apiField: "direction",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "readiness",
      header: t("Readiness"),
      cell: ({ row }) => <PartnerReadinessCell partnerId={row.original.id ?? ""} />,
      size: 120,
      enableSorting: false,
      meta: {
        label: t("Readiness"),
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
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "connection.status",
      header: t("Connection"),
      cell: ({ row }) => {
        const connection = row.original.connection as
          | { status?: string; method?: string }
          | null
          | undefined;
        return connection ? connection.method : <DataTablePlaceholder />;
      },
      size: 180,
      meta: {
        label: t("Connection"),
        apiField: "ediConnectionId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "defaultTransport.name",
      header: t("Profile"),
      cell: ({ row }) => {
        const profile = row.original.defaultTransport as
          | { name?: string; method?: string }
          | null
          | undefined;
        return profile ? (
          `${profile.name ?? "Profile"} (${profile.method ?? "Internal"})`
        ) : (
          <DataTablePlaceholder />
        );
      },
      size: 220,
      meta: {
        label: t("Profile"),
        apiField: "defaultTransportId",
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

function PartnerReadinessCell({ partnerId }: { partnerId: string }) {
  const { data, isPending, isError } = useQuery({
    ...partnerReadinessQueryOptions(partnerId),
    enabled: partnerId !== "",
  });
  if (partnerId === "" || isError) return <DataTablePlaceholder />;
  if (isPending) {
    return <span className="text-muted-foreground text-xs">…</span>;
  }
  if (!data) return <DataTablePlaceholder />;
  return (
    <EDIPartnerReadinessBadge
      ready={data.ready}
      completedCount={data.completedCount}
      totalCount={data.totalCount}
    />
  );
}
