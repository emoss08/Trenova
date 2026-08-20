import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { detentionPolicyStatusChoices, detentionRateSourceChoices } from "@/lib/choices";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import type { DetentionPolicy } from "@trenova/shared/types/detention";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(): ColumnDef<DetentionPolicy>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const choice = detentionPolicyStatusChoices.find(
          (option) => option.value === row.original.status,
        );
        return choice ? (
          <ColorOptionValue color={choice.color} value={choice.label} />
        ) : (
          row.original.status
        );
      },
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: "Status",
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: detentionPolicyStatusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <span className="font-medium">{row.original.name}</span>
          {row.original.isOrgDefault && (
            <Badge variant="outline" className="text-[10px]">
              Org default
            </Badge>
          )}
        </div>
      ),
      size: 240,
      minSize: 200,
      maxSize: 320,
      meta: {
        label: "Name",
        apiField: "name",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => (
        <span className="bg-muted rounded px-1.5 py-0.5 font-mono text-xs">
          {row.original.code}
        </span>
      ),
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: "Code",
        apiField: "code",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "billingFreeMinutes",
      header: "Free Time",
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatDetentionMinutes(row.original.billingFreeMinutes)}
        </span>
      ),
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: "Free Time",
        apiField: "billingFreeMinutes",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "rateSource",
      header: "Rate Source",
      cell: ({ row }) => {
        const choice = detentionRateSourceChoices.find(
          (option) => option.value === row.original.rateSource,
        );
        return choice?.label ?? row.original.rateSource;
      },
      size: 150,
      minSize: 130,
      maxSize: 180,
      meta: {
        label: "Rate Source",
        apiField: "rateSource",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: detentionRateSourceChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "specificityScore",
      header: "Specificity",
      cell: ({ row }) => (
        <span className="text-muted-foreground tabular-nums">{row.original.specificityScore}</span>
      ),
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: "Specificity",
        apiField: "specificityScore",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description} truncateLength={80} />
      ),
      size: 280,
      minSize: 200,
      maxSize: 400,
      meta: {
        label: "Description",
        apiField: "description",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "createdAt",
      header: "Created At",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      size: 200,
      minSize: 180,
      maxSize: 250,
      meta: {
        apiField: "createdAt",
        filterable: true,
        sortable: true,
      },
    },
  ];
}
