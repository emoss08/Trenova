import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { statusChoices } from "@/lib/choices";
import type { WorkerChecklistTemplateRow } from "@/lib/graphql/worker-checklist";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  CHECKLIST_KIND_LABELS,
  CHECKLIST_TRIGGER_LABELS,
  checklistKindSchema,
  checklistTriggerSchema,
  type ChecklistKind,
  type ChecklistTrigger,
} from "@trenova/shared/types/worker-checklist";

const KIND_CHOICES = checklistKindSchema.options.map((value) => ({
  value,
  label: CHECKLIST_KIND_LABELS[value],
}));
const TRIGGER_CHOICES = checklistTriggerSchema.options.map((value) => ({
  value,
  label: CHECKLIST_TRIGGER_LABELS[value],
}));

export function getColumns(t: TranslateFn): ColumnDef<WorkerChecklistTemplateRow>[] {
  return [
    {
      accessorKey: "status",
      header: t("Status"),
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
      size: 110,
      meta: {
        apiField: "status",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: statusChoices,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: t("Code"),
      cell: ({ row }) => (
        <span className="flex items-center gap-2 font-medium">
          {row.original.code}
          {row.original.isDefault ? (
            <Badge variant="purple" className="px-1.5 py-0 text-[10px]">
              {t("Default")}
            </Badge>
          ) : null}
        </span>
      ),
      meta: {
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
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span>{row.original.name}</span>
          {row.original.description ? (
            <span className="text-muted-foreground max-w-md truncate text-xs">
              {t(row.original.description)}
            </span>
          ) : null}
        </div>
      ),
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "kind",
      header: t("Kind"),
      cell: ({ row }) =>
        CHECKLIST_KIND_LABELS[row.original.kind as ChecklistKind] ?? row.original.kind,
      size: 120,
      meta: {
        apiField: "kind",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: KIND_CHOICES,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "trigger",
      header: t("Starts"),
      cell: ({ row }) =>
        CHECKLIST_TRIGGER_LABELS[row.original.trigger as ChecklistTrigger] ?? row.original.trigger,
      size: 140,
      meta: {
        apiField: "trigger",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: TRIGGER_CHOICES,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "items",
      header: t("Items"),
      cell: ({ row }) => {
        const required = row.original.items.filter((item) => item.required).length;
        return (
          <span className="tabular-nums">
            {row.original.items.length}
            <span className="text-muted-foreground"> {t("· {0} required", required)}</span>
          </span>
        );
      },
      size: 130,
    },
    {
      accessorKey: "openChecklistCount",
      header: t("In progress"),
      cell: ({ row }) => <span className="tabular-nums">{row.original.openChecklistCount}</span>,
      size: 100,
    },
    {
      accessorKey: "createdAt",
      header: t("Created"),
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: { apiField: "createdAt", sortable: true, filterable: true, filterType: "date" },
    },
  ];
}
