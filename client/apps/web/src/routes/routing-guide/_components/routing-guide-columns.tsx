import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { RoutingGuideRow } from "@/lib/graphql/routing-guide-table";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  ROUTING_GUIDE_TIER_LABEL,
  formatRoutingGuideLane,
} from "@trenova/shared/types/routing-guide";

export function getColumns(t: TranslateFn): ColumnDef<RoutingGuideRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => {
        const choice = statusChoices.find((option) => option.value === row.original.status);
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
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      size: 220,
      minSize: 180,
      maxSize: 320,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
      },
    },
    {
      id: "lane",
      header: t("Lane"),
      cell: ({ row }) => <span className="text-xs">{formatRoutingGuideLane(row.original)}</span>,
      size: 240,
      minSize: 200,
      maxSize: 360,
      meta: {
        label: t("Lane"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "specificity",
      header: t("Match Level"),
      cell: ({ row }) => (
        <Badge variant="outline" className="text-[10px]">
          {ROUTING_GUIDE_TIER_LABEL[row.original.specificity ?? 0] ?? "—"}
        </Badge>
      ),
      size: 140,
      minSize: 120,
      maxSize: 170,
      meta: {
        label: t("Match Level"),
        apiField: "specificity",
        filterable: false,
        sortable: true,
      },
    },
    {
      id: "entries",
      header: t("Carriers"),
      cell: ({ row }) => {
        const count = row.original.entries?.length ?? 0;
        return (
          <span className="text-muted-foreground tabular-nums">
            {t("{0, plural, one {# carrier} other {# carriers}}", count)}
          </span>
        );
      },
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: t("Carriers"),
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={row.original.description ?? ""} truncateLength={80} />
      ),
      size: 260,
      minSize: 200,
      maxSize: 400,
      meta: {
        label: t("Description"),
        apiField: "description",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "createdAt",
      header: t("Created At"),
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
