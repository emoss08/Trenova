import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Switch } from "@trenova/shared/components/ui/switch";
import type { ReportIR, ReportSortSpec } from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";
import { outputColumnChoices, type CatalogIndex } from "./builder-state";

const DIRECTION_CHOICES = [
  { value: "asc", label: "Ascending" },
  { value: "desc", label: "Descending" },
];

type SortLimitPanelProps = {
  index: CatalogIndex;
  ir: ReportIR;
  onSortChange: (sort: ReportSortSpec[]) => void;
  onLimitChange: (limit: number | undefined) => void;
  onTotalsChange: (totals: boolean) => void;
};

export function SortLimitPanel({
  index,
  ir,
  onSortChange,
  onLimitChange,
  onTotalsChange,
}: SortLimitPanelProps) {
  const sort = ir.sort ?? [];
  // Sorting addresses output columns, so a pivoted measure appears once per
  // bucket — sorting by the measure itself has nothing to point at.
  const choices = outputColumnChoices(index, ir);
  const items = choices.map((choice) => ({ value: choice.id, label: choice.label }));

  const hasMeasures = ir.columns.some((column) => column.kind !== "dimension");
  const hasDimensions = ir.columns.some((column) => column.kind === "dimension");
  const hasMeasureFilters = (ir.having?.filters?.length ?? 0) > 0;
  const totalsAvailable = hasMeasures && hasDimensions && !hasMeasureFilters;

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-2">
        {sort.map((spec, sortIndex) => (
          <div key={sortIndex} className="flex items-center gap-1.5">
            <Select
              value={spec.columnId}
              onValueChange={(columnId) => {
                if (!columnId) return;
                onSortChange(sort.map((s, i) => (i === sortIndex ? { ...s, columnId } : s)));
              }}
              items={items}
            >
              <SelectTrigger className="h-7 flex-1">
                <SelectValue placeholder="Column" />
              </SelectTrigger>
              <SelectContent>
                {choices.map((choice) => (
                  <SelectItem key={choice.id} value={choice.id}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={spec.direction}
              onValueChange={(direction) => {
                if (direction !== "asc" && direction !== "desc") return;
                onSortChange(sort.map((s, i) => (i === sortIndex ? { ...s, direction } : s)));
              }}
              items={DIRECTION_CHOICES}
            >
              <SelectTrigger className="h-7 w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="asc">Ascending</SelectItem>
                <SelectItem value="desc">Descending</SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="ghost"
              size="icon"
              className="size-6"
              onClick={() => onSortChange(sort.filter((_, i) => i !== sortIndex))}
              aria-label="Remove sort"
            >
              <XIcon className="size-3.5" />
            </Button>
          </div>
        ))}
        <Button
          variant="outline"
          size="sm"
          className="h-7 self-start"
          disabled={choices.length === 0}
          onClick={() => onSortChange([...sort, { columnId: choices[0].id, direction: "desc" }])}
        >
          <PlusIcon className="size-3.5" />
          Sort
        </Button>
      </div>
      <div className="flex items-center gap-2">
        <Label htmlFor="report-limit" className="text-xs text-muted-foreground">
          Row Limit
        </Label>
        <Input
          id="report-limit"
          className="h-7 w-32"
          type="number"
          min={1}
          placeholder="Server default"
          value={ir.limit ? String(ir.limit) : ""}
          onChange={(event) => {
            const parsed = Number.parseInt(event.target.value, 10);
            onLimitChange(Number.isNaN(parsed) || parsed <= 0 ? undefined : parsed);
          }}
        />
      </div>
      <div className="flex flex-col gap-1 rounded-md border border-border p-2">
        <div className="flex items-center justify-between gap-2">
          <Label htmlFor="report-totals" className="text-xs text-muted-foreground">
            Total row
          </Label>
          <Switch
            id="report-totals"
            checked={Boolean(ir.totals) && totalsAvailable}
            disabled={!totalsAvailable}
            onCheckedChange={onTotalsChange}
          />
        </div>
        <p className="text-2xs text-muted-foreground">
          {hasMeasureFilters
            ? "Totals are unavailable while measure filters are set — the total would count groups the report excludes."
            : "Totals are computed from every matching record, so an average stays an average instead of averaging the rows on screen."}
        </p>
      </div>
    </div>
  );
}
