import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DataTableDescription } from "@/components/data-table/_components/data-table-components";
import { ColorOptionValue } from "@/components/fields/select-components";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { detentionPolicyStatusChoices, detentionRateSourceChoices } from "@/lib/choices";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import type { DetentionPolicyRow } from "@/lib/graphql/detention-policy-table";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(t: TranslateFn): ColumnDef<DetentionPolicyRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
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
        label: t("Status"),
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
      header: t("Name"),
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5">
          <span className="font-medium">{row.original.name}</span>
          {row.original.isOrgDefault && (
            <Badge variant="outline" className="text-[10px]">
              {t("Org default")}
            </Badge>
          )}
        </div>
      ),
      size: 240,
      minSize: 200,
      maxSize: 320,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => (
        <span className="bg-muted rounded px-1.5 py-0.5 font-mono text-xs">
          {row.original.code}
        </span>
      ),
      size: 140,
      minSize: 120,
      maxSize: 180,
      meta: {
        label: t("Code"),
        apiField: "code",
        filterable: true,
        sortable: true,
      },
    },
    {
      accessorKey: "billingFreeMinutes",
      header: t("Free Time"),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatDetentionMinutes(row.original.billingFreeMinutes)}
        </span>
      ),
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: t("Free Time"),
        apiField: "billingFreeMinutes",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "rateSource",
      header: t("Rate Source"),
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
        label: t("Rate Source"),
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
      header: t("Specificity"),
      cell: ({ row }) => (
        <span className="text-muted-foreground tabular-nums">{row.original.specificityScore}</span>
      ),
      size: 110,
      minSize: 100,
      maxSize: 130,
      meta: {
        label: t("Specificity"),
        apiField: "specificityScore",
        filterable: false,
        sortable: true,
      },
    },
    {
      accessorKey: "description",
      header: t("Description"),
      cell: ({ row }) => (
        <DataTableDescription description={t(row.original.description)} truncateLength={80} />
      ),
      size: 280,
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
