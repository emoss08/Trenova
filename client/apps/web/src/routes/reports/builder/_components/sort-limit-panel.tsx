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
import type { ReportIR, ReportSortSpec } from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";
import { columnDisplayLabel, type CatalogIndex } from "./builder-state";

const DIRECTION_CHOICES = [
  { value: "asc", label: "Ascending" },
  { value: "desc", label: "Descending" },
];

type SortLimitPanelProps = {
  index: CatalogIndex;
  ir: ReportIR;
  onSortChange: (sort: ReportSortSpec[]) => void;
  onLimitChange: (limit: number | undefined) => void;
};

export function SortLimitPanel({ index, ir, onSortChange, onLimitChange }: SortLimitPanelProps) {
  const sort = ir.sort ?? [];

  const columnLabel = (columnId: string): string => {
    const column = ir.columns.find((c) => c.id === columnId);
    if (!column) return columnId;
    return columnDisplayLabel(index, ir, column);
  };

  const columnChoices = ir.columns.map((column) => ({
    value: column.id,
    label: columnLabel(column.id),
  }));

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
              items={columnChoices}
            >
              <SelectTrigger className="h-7 flex-1">
                <SelectValue placeholder="Column" />
              </SelectTrigger>
              <SelectContent>
                {ir.columns.map((column) => (
                  <SelectItem key={column.id} value={column.id}>
                    {columnLabel(column.id)}
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
          disabled={ir.columns.length === 0}
          onClick={() => onSortChange([...sort, { columnId: ir.columns[0].id, direction: "desc" }])}
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
    </div>
  );
}
