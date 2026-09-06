import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { ptoTypeChoices, statusChoices } from "@/lib/choices";
import type { PTOPolicyRow } from "@/lib/graphql/pto-policy";
import { Badge } from "@trenova/shared/components/ui/badge";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { PTO_YEAR_BASIS_LABELS } from "@trenova/shared/types/pto-policy";

const POLICY_STATUS_CHOICES = [
  ...statusChoices,
  { value: "Draft", label: "Draft", color: "#6b7280" },
];

function ruleSummary(row: PTOPolicyRow): string[] {
  return row.rules.map((rule) => {
    const type =
      ptoTypeChoices.find((choice) => choice.value === rule.ptoType)?.label ?? rule.ptoType;
    if (rule.accrualMethod === "None") return `${type}: tracked`;
    if (rule.accrualMethod === "Monthly") return `${type}: ${rule.accrualAmountDays}/mo`;
    return `${type}: ${rule.accrualAmountDays}/yr`;
  });
}

export function getColumns(): ColumnDef<PTOPolicyRow>[] {
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
        filterOptions: POLICY_STATUS_CHOICES,
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "code",
      header: "Code",
      cell: ({ row }) => (
        <span className="flex items-center gap-2 font-medium">
          {row.original.code}
          {row.original.isDefault ? (
            <Badge variant="purple" className="px-1.5 py-0 text-[10px]">
              Default
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
      header: "Name",
      meta: {
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "yearBasis",
      header: "Year",
      cell: ({ row }) => PTO_YEAR_BASIS_LABELS[row.original.yearBasis],
      meta: {
        apiField: "yearBasis",
        filterable: false,
        sortable: true,
      },
    },
    {
      id: "rules",
      header: "Tracked Types",
      cell: ({ row }) => (
        <div className="flex flex-wrap gap-1">
          {ruleSummary(row.original).map((summary) => (
            <Badge key={summary} variant="secondary" className="px-1.5 py-0 text-[10px]">
              {summary}
            </Badge>
          ))}
        </div>
      ),
      enableSorting: false,
      meta: {
        apiField: "rules",
        filterable: false,
        sortable: false,
        exportValue: (row: PTOPolicyRow) => ruleSummary(row).join("; "),
      },
    },
    {
      accessorKey: "openAssignmentCount",
      header: "Workers",
      cell: ({ row }) => (
        <span className="font-table tabular-nums">{row.original.openAssignmentCount}</span>
      ),
      enableSorting: false,
      meta: {
        apiField: "openAssignmentCount",
        filterable: false,
        sortable: false,
      },
    },
    {
      accessorKey: "updatedAt",
      header: "Updated",
      cell: ({ row }) => (
        <HoverCardTimestamp
          className="font-table tracking-tight"
          timestamp={row.original.updatedAt}
        />
      ),
      meta: {
        apiField: "updatedAt",
        filterable: false,
        sortable: true,
        filterType: "date",
        defaultFilterOperator: "daterange",
      },
    },
  ];
}
