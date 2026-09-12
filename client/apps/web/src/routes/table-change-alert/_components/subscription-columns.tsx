import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import type { TCASubscriptionRow } from "@/lib/graphql/table-change-alert-table";
import type { NotificationPriority } from "@/types/table-change-alert";
import type { ColumnDef } from "@trenova/shared/types/data-table";

const PRIORITY_BADGE_VARIANT: Record<NotificationPriority, BadgeVariant> = {
  critical: "inactive",
  high: "orange",
  medium: "secondary",
  low: "teal",
};

export function getColumns(t: TranslateFn): ColumnDef<TCASubscriptionRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      accessorKey: "tableName",
      header: t("Table"),
    },
    {
      accessorKey: "eventTypes",
      header: t("Events"),
      cell: ({ row }) => (
        <div className="flex gap-1">
          {row.original.eventTypes.map((et) => (
            <Badge key={et} variant="outline" className="text-2xs">
              {et}
            </Badge>
          ))}
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => (
        <Badge variant={row.original.status === "Active" ? "active" : "secondary"}>
          {row.original.status}
        </Badge>
      ),
    },
    {
      accessorKey: "priority",
      header: t("Priority"),
      cell: ({ row }) => {
        const p = row.original.priority ?? "medium";
        return <Badge variant={PRIORITY_BADGE_VARIANT[p as NotificationPriority]}>{p}</Badge>;
      },
    },
    {
      accessorKey: "conditions",
      header: t("Conditions"),
      cell: ({ row }) => {
        const count = row.original.conditions?.length ?? 0;
        if (count === 0) return <span className="text-muted-foreground">{t("None")}</span>;
        return (
          <Badge variant="info">
            {t("{0, plural, one {# condition} other {# conditions}}", count)}
          </Badge>
        );
      },
    },
  ];
}
