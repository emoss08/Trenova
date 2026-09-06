import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { driverTypeChoices, statusChoices } from "@/lib/choices";
import type { WorkerCredentialTypeRow } from "@/lib/graphql/worker-credential";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import {
  CREDENTIAL_CATEGORY_LABELS,
  credentialCategorySchema,
} from "@trenova/shared/types/worker-credential";

const CATEGORY_CHOICES = credentialCategorySchema.options.map((value) => ({
  value,
  label: CREDENTIAL_CATEGORY_LABELS[value],
}));

function requiredSummary(row: WorkerCredentialTypeRow): string {
  if (!row.isRequired) return "Optional";
  if (row.requiredForDriverTypes.length === 0) return "All drivers";
  return row.requiredForDriverTypes
    .map((type) => driverTypeChoices.find((choice) => choice.value === type)?.label ?? type)
    .join(", ");
}

export function getColumns(): ColumnDef<WorkerCredentialTypeRow>[] {
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
      cell: ({ row }) => (
        <span className="flex items-center gap-2 font-medium">{row.original.code}</span>
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
      cell: ({ row }) => CREDENTIAL_CATEGORY_LABELS[row.original.category] ?? row.original.category,
      size: 130,
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
      accessorKey: "renewalWindowDays",
      header: "Alert window",
      cell: ({ row }) => `${row.original.renewalWindowDays} days`,
      size: 110,
      meta: { apiField: "renewalWindowDays", sortable: true },
    },
    {
      accessorKey: "validityMonths",
      header: "Validity",
      cell: ({ row }) =>
        row.original.validityMonths ? (
          `${row.original.validityMonths} mo`
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 90,
    },
    {
      id: "requirements",
      header: "Needs",
      cell: ({ row }) => (
        <span className="flex gap-1">
          {row.original.requiresNumber ? (
            <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
              Number
            </Badge>
          ) : null}
          {row.original.requiresDocument ? (
            <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
              Document
            </Badge>
          ) : null}
        </span>
      ),
      size: 140,
    },
    {
      accessorKey: "activeCredentialCount",
      header: "Held by",
      cell: ({ row }) => <span className="tabular-nums">{row.original.activeCredentialCount}</span>,
      size: 90,
    },
    {
      accessorKey: "createdAt",
      header: "Created",
      cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.createdAt} />,
      meta: { apiField: "createdAt", sortable: true, filterable: true, filterType: "date" },
    },
  ];
}
