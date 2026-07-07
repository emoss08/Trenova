import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { EDIPartnerReadinessBadge, StatusBadge } from "@/components/status-badge";
import { useQuery } from "@tanstack/react-query";
import { partnerReadinessQueryOptions } from "./edi-partner-readiness";
import { Badge } from "@/components/ui/badge";
import { statusChoices } from "@/lib/choices";
import type { EDIPartner } from "@/types/edi";
import type { ColumnDef } from "@tanstack/react-table";

export function getPartnerColumns(): ColumnDef<EDIPartner>[] {
  return [
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => <span className="font-medium">{row.original.code}</span>,
      size: 140,
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
      size: 220,
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
      accessorKey: "internalOrganization.name",
      header: "Target Organization",
      cell: ({ row }) =>
        row.original.internalOrganization?.name ??
        row.original.internalOrganizationId ?? <DataTablePlaceholder />,
      size: 240,
      meta: {
        label: "Target Organization",
        apiField: "internalOrganizationId",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "direction",
      header: "Direction",
      cell: ({ row }) => (
        <div className="flex gap-1">
          <Badge variant={row.original.enabledForInbound ? "secondary" : "outline"}>Inbound</Badge>
          <Badge variant={row.original.enabledForOutbound ? "secondary" : "outline"}>
            Outbound
          </Badge>
        </div>
      ),
      size: 180,
      meta: {
        label: "Direction",
        apiField: "direction",
        filterable: false,
        sortable: false,
      },
    },
    {
      id: "readiness",
      header: "Readiness",
      cell: ({ row }) => <PartnerReadinessCell partnerId={row.original.id ?? ""} />,
      size: 120,
      enableSorting: false,
      meta: {
        label: "Readiness",
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      size: 120,
      meta: {
        label: "Status",
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
      header: "Connection",
      cell: ({ row }) => {
        const connection = row.original.connection as
          | { status?: string; method?: string }
          | null
          | undefined;
        return connection ? connection.method : <DataTablePlaceholder />;
      },
      size: 180,
      meta: {
        label: "Connection",
        apiField: "ediConnectionId",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "defaultTransport.name",
      header: "Profile",
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
        label: "Profile",
        apiField: "defaultTransportId",
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

function PartnerReadinessCell({ partnerId }: { partnerId: string }) {
  const { data, isPending, isError } = useQuery({
    ...partnerReadinessQueryOptions(partnerId),
    enabled: partnerId !== "",
  });
  if (partnerId === "" || isError) return <DataTablePlaceholder />;
  if (isPending) {
    return <span className="text-xs text-muted-foreground">…</span>;
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
