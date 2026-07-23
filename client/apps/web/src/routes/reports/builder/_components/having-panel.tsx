import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  REPORT_AGGREGATION_LABELS,
  type ReportFieldFilter,
  type ReportFilterGroup,
  type ReportIR,
} from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";
import { measureColumns, refLabel, type CatalogIndex } from "./builder-state";

const HAVING_OPERATORS = [
  { value: "eq", label: "Equals" },
  { value: "ne", label: "Not equals" },
  { value: "gt", label: "Greater than" },
  { value: "gte", label: "Greater than or equal" },
  { value: "lt", label: "Less than" },
  { value: "lte", label: "Less than or equal" },
];

type HavingPanelProps = {
  index: CatalogIndex;
  ir: ReportIR;
  onChange: (group: ReportFilterGroup | undefined) => void;
};

function measureKey(filter: ReportFieldFilter): string {
  return `${(filter.ref.path ?? []).join(".")}|${filter.ref.field}|${filter.agg ?? ""}`;
}

export function HavingPanel({ index, ir, onChange }: HavingPanelProps) {
  const measures = measureColumns(ir);
  const filters = ir.having?.filters ?? [];

  if (measures.length === 0) {
    return (
      <p className="px-2 py-4 text-center text-sm text-muted-foreground">
        Add measure columns to filter on aggregated values.
      </p>
    );
  }

  const measureChoices = measures.map((column) => ({
    value: measureKey({ ref: column.ref, operator: "", agg: column.agg }),
    label: `${column.agg ? `${REPORT_AGGREGATION_LABELS[column.agg]} of ` : ""}${refLabel(
      index,
      ir.entity,
      column.ref,
    )}`,
  }));

  const updateFilters = (next: ReportFieldFilter[]) => {
    onChange(next.length > 0 ? { op: "and", filters: next } : undefined);
  };

  return (
    <div className="flex flex-col gap-2">
      {filters.map((filter, filterIndex) => (
        <div
          key={filterIndex}
          className="flex flex-wrap items-center gap-1.5 rounded-md border border-border p-2"
        >
          <Select
            value={measureKey(filter)}
            onValueChange={(key) => {
              const measure = measures.find(
                (column) => measureKey({ ref: column.ref, operator: "", agg: column.agg }) === key,
              );
              if (!measure) return;
              updateFilters(
                filters.map((f, i) =>
                  i === filterIndex ? { ...f, ref: measure.ref, agg: measure.agg } : f,
                ),
              );
            }}
            items={measureChoices}
          >
            <SelectTrigger className="h-7 w-56">
              <SelectValue placeholder="Select measure" />
            </SelectTrigger>
            <SelectContent>
              {measures.map((column) => (
                <SelectItem
                  key={column.id}
                  value={measureKey({ ref: column.ref, operator: "", agg: column.agg })}
                >
                  {column.agg ? `${REPORT_AGGREGATION_LABELS[column.agg]} of ` : ""}
                  {refLabel(index, ir.entity, column.ref)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            value={filter.operator}
            onValueChange={(operator) => {
              if (!operator) return;
              updateFilters(filters.map((f, i) => (i === filterIndex ? { ...f, operator } : f)));
            }}
            items={HAVING_OPERATORS}
          >
            <SelectTrigger className="h-7 w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {HAVING_OPERATORS.map((op) => (
                <SelectItem key={op.value} value={op.value}>
                  {op.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Input
            className="h-7 w-32"
            type="number"
            step="any"
            value={typeof filter.value === "number" ? String(filter.value) : ""}
            onChange={(event) => {
              const parsed = Number.parseFloat(event.target.value);
              updateFilters(
                filters.map((f, i) =>
                  i === filterIndex
                    ? { ...f, value: Number.isNaN(parsed) ? undefined : parsed }
                    : f,
                ),
              );
            }}
          />
          <Button
            variant="ghost"
            size="icon"
            className="size-6"
            onClick={() => updateFilters(filters.filter((_, i) => i !== filterIndex))}
            aria-label="Remove measure filter"
          >
            <XIcon className="size-3.5" />
          </Button>
        </div>
      ))}
      <Button
        variant="outline"
        size="sm"
        className="h-7 self-start"
        onClick={() => {
          const first = measures[0];
          updateFilters([
            ...filters,
            { ref: first.ref, agg: first.agg, operator: "gt", value: undefined },
          ]);
        }}
      >
        <PlusIcon className="size-3.5" />
        Measure Filter
      </Button>
    </div>
  );
}
