import { Badge } from "@trenova/shared/components/ui/badge";
import { formatFeetInches, formatPounds } from "@trenova/shared/lib/permit";
import { truncateText } from "@trenova/shared/lib/utils";
import type { JurisdictionRuleOverride } from "@/types/jurisdiction-rule-override";
import { type ColumnDef } from "@tanstack/react-table";

/**
 * Which limits this override actually narrows. An override is sparse by design —
 * unset means defer to the statute — so a column per limit would be mostly
 * empty and would not answer the question an operator has, which is "what does
 * this row change".
 */
function overriddenFields(row: JurisdictionRuleOverride): string[] {
  const applied: string[] = [];

  if (row.maxWidthFeet != null) applied.push(`Width ${formatFeetInches(row.maxWidthFeet)}`);
  if (row.maxHeightFeet != null) applied.push(`Height ${formatFeetInches(row.maxHeightFeet)}`);
  if (row.maxLengthFeet != null) applied.push(`Length ${formatFeetInches(row.maxLengthFeet)}`);
  if (row.maxWeightPounds != null) applied.push(formatPounds(row.maxWeightPounds));
  if (row.permitLeadTimeDays != null) applied.push(`${row.permitLeadTimeDays}d lead`);
  if (row.daylightOnly) applied.push("Daylight only");
  if (row.holidayRestricted) applied.push("No holidays");

  return applied;
}

export function getColumns(): ColumnDef<JurisdictionRuleOverride>[] {
  return [
    {
      accessorKey: "state",
      header: "State",
      cell: ({ row }) => (
        <span className="text-sm font-medium">
          {row.original.state?.abbreviation ?? "—"}
          <span className="ml-2 text-muted-foreground">{row.original.state?.name}</span>
        </span>
      ),
      size: 200,
      meta: { label: "State", apiField: "stateId", sortable: true },
    },
    {
      id: "overrides",
      header: "Narrows",
      cell: ({ row }) => {
        const applied = overriddenFields(row.original);

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
        <span className="text-sm text-muted-foreground">
          {truncateText(row.original.reason, 80)}
        </span>
      ),
      size: 300,
      meta: { label: "Reason", apiField: "reason", filterable: true, filterType: "text" },
    },
  ];
}
