import { Badge, type BadgeVariant } from "@/components/ui/badge";
import type { NotificationPriority, TCASubscription } from "@/types/table-change-alert";
import type { ColumnDef } from "@tanstack/react-table";

const PRIORITY_BADGE_VARIANT: Record<NotificationPriority, BadgeVariant> = {
  critical: "inactive",
  high: "orange",
  medium: "secondary",
  low: "teal",
};

export function getColumns(): ColumnDef<TCASubscription>[] {
  return [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <span className="font-medium">{row.original.name}</span>
      ),
    },
    {
      accessorKey: "tableName",
      header: "Table",
    },
    {
      accessorKey: "eventTypes",
      header: "Events",
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
      header: "Status",
      cell: ({ row }) => (
        <Badge
          variant={row.original.status === "Active" ? "active" : "secondary"}
        >
          {row.original.status}
        </Badge>
      ),
    },
    {
      accessorKey: "priority",
      header: "Priority",
      cell: ({ row }) => {
        const p = row.original.priority ?? "medium";
        return (
          <Badge variant={PRIORITY_BADGE_VARIANT[p as NotificationPriority]}>
            {p}
          </Badge>
        );
      },
    },
    {
      accessorKey: "conditions",
      header: "Conditions",
      cell: ({ row }) => {
        const count = row.original.conditions?.length ?? 0;
        if (count === 0) return <span className="text-muted-foreground">None</span>;
        return (
          <Badge variant="info">
            {count} condition{count !== 1 ? "s" : ""}
          </Badge>
        );
      },
    },
  ];
}
