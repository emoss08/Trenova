import { Badge } from "@trenova/shared/components/ui/badge";
import { overriddenLimits } from "@trenova/shared/lib/permit";
import { truncateText } from "@trenova/shared/lib/utils";
import type { JurisdictionRuleOverride } from "@/types/jurisdiction-rule-override";
import type { ColumnDef } from "@trenova/shared/types/data-table";

export function getColumns(): ColumnDef<JurisdictionRuleOverride>[] {
  return [
    {
      accessorKey: "state",
      header: "State",
      cell: ({ row }) => (
        <span className="text-sm font-medium">
          {row.original.state?.abbreviation ?? "—"}
          <span className="text-muted-foreground ml-2">{row.original.state?.name}</span>
        </span>
      ),
      size: 200,
      meta: { label: "State", apiField: "stateId", sortable: true },
    },
    {
      id: "overrides",
      header: "Narrows",
      cell: ({ row }) => {
        const applied = overriddenLimits(row.original);

        return applied.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {applied.map((label) => (
              <Badge key={label} variant="outline">
                {label}
              </Badge>
            ))}
          </div>
        ) : (
          <span className="text-muted-foreground">Nothing</span>
        );
      },
      size: 320,
      meta: { label: "Narrows" },
    },
    {
      accessorKey: "reason",
      header: "Reason",
      cell: ({ row }) => (
        <span className="text-muted-foreground text-sm">
          {truncateText(row.original.reason, 80)}
        </span>
      ),
      size: 300,
      meta: { label: "Reason", apiField: "reason", filterable: true, filterType: "text" },
    },
  ];
}
