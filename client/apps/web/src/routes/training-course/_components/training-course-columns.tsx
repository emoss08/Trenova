import { translate } from "@trenova/shared/i18n/runtime";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { driverTypeChoices, statusChoices } from "@/lib/choices";
import type { TrainingCourseRow } from "@/lib/graphql/worker-training";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  TRAINING_CATEGORY_LABELS,
  TRAINING_DELIVERY_LABELS,
  trainingCategorySchema,
  trainingDeliverySchema,
  type TrainingCategory,
  type TrainingDelivery,
} from "@trenova/shared/types/worker-training";

const CATEGORY_CHOICES = trainingCategorySchema.options.map((value) => ({
  value,
  label: TRAINING_CATEGORY_LABELS[value],
}));

const DELIVERY_CHOICES = trainingDeliverySchema.options.map((value) => ({
  value,
  label: TRAINING_DELIVERY_LABELS[value],
}));

function requiredSummary(row: TrainingCourseRow): string {
  if (!row.isRequired) return "Optional";
  if (row.requiredForDriverTypes.length === 0) return "All drivers";
  return row.requiredForDriverTypes
    .map((type) => driverTypeChoices.find((choice) => choice.value === type)?.label ?? type)
    .join(", ");
}

export function getColumns(): ColumnDef<TrainingCourseRow>[] {
  return [
    {
      accessorKey: "status",
      header: "Status",
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
      header: "Code",
      cell: ({ row }) => <span className="font-medium">{row.original.code}</span>,
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
      header: "Name",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span>{row.original.name}</span>
          {row.original.description ? (
            <span className="text-muted-foreground max-w-md truncate text-xs">
              {row.original.description}
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
      accessorKey: "category",
      header: "Category",
      cell: ({ row }) =>
        TRAINING_CATEGORY_LABELS[row.original.category as TrainingCategory] ??
        row.original.category,
      size: 140,
      meta: {
        apiField: "category",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: CATEGORY_CHOICES,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "delivery",
      header: "Delivery",
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5">
          {TRAINING_DELIVERY_LABELS[row.original.delivery as TrainingDelivery] ??
            row.original.delivery}
          {row.original.durationMinutes > 0 ? (
            <span className="text-muted-foreground text-xs">
              {translate("{0} min", row.original.durationMinutes)}
            </span>
          ) : null}
        </span>
      ),
      size: 170,
      meta: {
        apiField: "delivery",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: DELIVERY_CHOICES,
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "required",
      header: "Required for",
      cell: ({ row }) => (
        <span className={row.original.isRequired ? "font-medium" : "text-muted-foreground"}>
          {requiredSummary(row.original)}
        </span>
      ),
      size: 160,
    },
    {
      id: "rules",
      header: "Rules",
      cell: ({ row }) => (
        <span className="flex flex-wrap gap-1">
          {row.original.passingScore ? (
            <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
              {translate("Pass ≥ {0}%", Number(row.original.passingScore).toFixed(0))}
            </Badge>
          ) : null}
          {row.original.validityMonths ? (
            <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
              {translate("Every {0} mo", row.original.validityMonths)}
            </Badge>
          ) : (
            <Badge variant="outline" className="text-muted-foreground px-1.5 py-0 text-[10px]">
              {translate("One-time")}
            </Badge>
          )}
        </span>
      ),
      size: 180,
    },
    {
      accessorKey: "dueDaysAfterAssignment",
      header: "Due after",
      cell: ({ row }) =>
        row.original.dueDaysAfterAssignment > 0 ? (
          `${row.original.dueDaysAfterAssignment} days`
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 100,
      meta: { apiField: "dueDaysAfterAssignment", sortable: true },
    },
    {
      accessorKey: "openRecordCount",
      header: "In progress",
      cell: ({ row }) => <span className="tabular-nums">{row.original.openRecordCount}</span>,
      size: 100,
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: { apiField: "createdAt", sortable: true, filterable: true, filterType: "date" },
    },
  ];
}
