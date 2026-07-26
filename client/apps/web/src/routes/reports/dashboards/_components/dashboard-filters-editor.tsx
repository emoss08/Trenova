import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { operatorsForFieldType, type ReportDashboardFilter } from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";
import { useMemo, useState } from "react";
import {
  preferReferenceField,
  refLabel,
  resolveField,
  type CatalogIndex,
} from "../../builder/_components/builder-state";
import { CatalogFieldTree } from "../../builder/_components/catalog-field-tree";
import { refKey } from "./dashboard-state";
import type { FilterCandidate } from "./use-tile-report";

type DashboardFiltersEditorProps = {
  index: CatalogIndex;
  filters: ReportDashboardFilter[];
  candidates: FilterCandidate[];
  entities: string[];
  onChange: (filters: ReportDashboardFilter[]) => void;
};

function filterKey(filter: ReportDashboardFilter): string {
  return `${filter.entity}::${refKey(filter.ref)}`;
}

function defaultOperator(fieldType: string): string {
  const choices = operatorsForFieldType(fieldType);
  // "Is any of" is what a dashboard control almost always wants.
  return (choices.find((choice) => choice.value === "in") ?? choices[0])?.value ?? "eq";
}

/** One row per already-selected filter, for renaming or changing the comparison. */
function SelectedFilter({
  index,
  filter,
  onUpdate,
  onRemove,
}: {
  index: CatalogIndex;
  filter: ReportDashboardFilter;
  onUpdate: (filter: ReportDashboardFilter) => void;
  onRemove: () => void;
}) {
  const field = resolveField(index, filter.entity, filter.ref);
  const operators = field ? operatorsForFieldType(field.type) : [];

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border p-2">
      <div className="flex items-center gap-1.5">
        <span className="min-w-0 flex-1 truncate text-xs font-medium">
          {refLabel(index, filter.entity, filter.ref)}
        </span>
        <Button
          variant="ghost"
          size="icon"
          className="size-6"
          onClick={onRemove}
          aria-label="Remove filter"
        >
          <XIcon className="size-3.5" />
        </Button>
      </div>
      <div className="grid grid-cols-2 gap-2">
        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Shown as</Label>
          <Input
            className="h-7"
            value={filter.label ?? ""}
            placeholder={field?.label ?? filter.ref.field}
            onChange={(event) => onUpdate({ ...filter, label: event.target.value || undefined })}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Comparison</Label>
          <Select
            value={filter.operator}
            onValueChange={(operator) => {
              if (operator) onUpdate({ ...filter, operator });
            }}
            items={operators}
          >
            <SelectTrigger className="h-7">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {operators.map((choice) => (
                <SelectItem key={choice.value} value={choice.value}>
                  {choice.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>
  );
}

function CustomFieldPicker({
  index,
  entities,
  onAdd,
}: {
  index: CatalogIndex;
  entities: string[];
  onAdd: (entity: string, ref: { path?: string[]; field: string }, operator: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [entity, setEntity] = useState(entities[0] ?? "");
  const entityChoices = entities.map((key) => ({
    value: key,
    label: index.entities.get(key)?.pluralLabel ?? key,
  }));

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm" className="h-7 self-start">
            <PlusIcon className="size-3.5" />
            Another field
          </Button>
        }
      />
      <PopoverContent className="flex h-96 w-72 flex-col gap-2 p-2" align="start">
        {entityChoices.length > 1 && (
          <Select
            value={entity}
            onValueChange={(next) => {
              if (next) setEntity(next);
            }}
            items={entityChoices}
          >
            <SelectTrigger className="h-7">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {entityChoices.map((choice) => (
                <SelectItem key={choice.value} value={choice.value}>
                  {choice.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <CatalogFieldTree
          index={index}
          entityKey={entity}
          className="min-h-0 flex-1"
          filterFields={(field, crossesToMany) =>
            field.filterable && field.accessible && !crossesToMany
          }
          onSelectField={(selection) => {
            onAdd(entity, selection.ref, defaultOperator(selection.field.type));
            setOpen(false);
          }}
        />
      </PopoverContent>
    </Popover>
  );
}

export function DashboardFiltersEditor({
  index,
  filters,
  candidates,
  entities,
  onChange,
}: DashboardFiltersEditorProps) {
  // A grouping like "Service Type › Code" is normalised to the reference
  // beside it, so the control offers real records instead of raw text. Two
  // candidates that normalise to the same field collapse into one row.
  const offered = useMemo(() => {
    const byKey = new Map<string, FilterCandidate>();
    for (const candidate of candidates) {
      const ref = preferReferenceField(index, candidate.entity, candidate.ref);
      const key = `${candidate.entity}::${refKey(ref)}`;
      const existing = byKey.get(key);
      if (existing) {
        for (const report of candidate.reports) {
          if (!existing.reports.includes(report)) existing.reports.push(report);
        }
        continue;
      }
      byKey.set(key, { ...candidate, key, ref, reports: [...candidate.reports] });
    }
    return [...byKey.values()];
  }, [candidates, index]);

  // Grouped by the report a field came from: the flat list interleaved tiles,
  // so one report could appear in two places, and the per-row report name had
  // to be truncated to fit. The heading carries it instead.
  const groups = useMemo(() => {
    const byReport = new Map<string, FilterCandidate[]>();
    for (const candidate of offered) {
      const field = resolveField(index, candidate.entity, candidate.ref);
      if (!field?.filterable || !field.accessible) continue;
      for (const report of candidate.reports.length > 0 ? candidate.reports : ["Other"]) {
        const bucket = byReport.get(report) ?? [];
        bucket.push(candidate);
        byReport.set(report, bucket);
      }
    }

    return [...byReport.entries()]
      .map(([report, items]) => ({
        report,
        items: [...items].sort((a, b) =>
          refLabel(index, a.entity, a.ref).localeCompare(refLabel(index, b.entity, b.ref)),
        ),
      }))
      .sort((a, b) => a.report.localeCompare(b.report));
  }, [offered, index]);

  if (entities.length === 0) {
    return (
      <p className="px-2 py-2 text-center text-sm text-muted-foreground">
        Add a tile first — filters are drawn from the reports on this dashboard.
      </p>
    );
  }

  const selected = new Set(filters.map(filterKey));

  const toggle = (candidate: FilterCandidate, checked: boolean) => {
    if (!checked) {
      onChange(filters.filter((filter) => filterKey(filter) !== candidate.key));
      return;
    }
    const field = resolveField(index, candidate.entity, candidate.ref);
    onChange([
      ...filters,
      {
        id: candidate.key,
        entity: candidate.entity,
        ref: candidate.ref,
        operator: defaultOperator(field?.type ?? "string"),
      },
    ]);
  };

  return (
    <div className="flex flex-col gap-4">
      {filters.length > 0 && (
        <section className="flex flex-col gap-2">
          <h3 className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
            On this dashboard
          </h3>
          {filters.map((filter, filterIndex) => (
            <SelectedFilter
              key={filter.id}
              index={index}
              filter={filter}
              onUpdate={(updated) =>
                onChange(filters.map((f, i) => (i === filterIndex ? updated : f)))
              }
              onRemove={() => onChange(filters.filter((_, i) => i !== filterIndex))}
            />
          ))}
        </section>
      )}

      <section className="flex flex-col gap-2">
        <h3 className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          Add from your reports
        </h3>
        <p className="text-2xs text-muted-foreground">
          What each report filters on and breaks down by. Turning one on narrows every tile built on
          the same data.
        </p>

        {groups.map((group) => (
          <div key={group.report} className="flex flex-col gap-0.5">
            <p
              className="truncate px-1 pt-1.5 text-2xs font-medium text-foreground/70"
              title={group.report}
            >
              {group.report}
            </p>
            {group.items.map((candidate) => (
              <label
                key={`${group.report}::${candidate.key}`}
                className="flex items-center gap-2 rounded-md px-1 py-1 text-xs hover:bg-accent/40"
              >
                <Checkbox
                  checked={selected.has(candidate.key)}
                  onCheckedChange={(checked) => toggle(candidate, Boolean(checked))}
                />
                <span
                  className="min-w-0 flex-1 truncate"
                  title={refLabel(index, candidate.entity, candidate.ref)}
                >
                  {refLabel(index, candidate.entity, candidate.ref)}
                </span>
                <Badge variant="outline" className="shrink-0 text-[10px]">
                  {candidate.source === "filter" ? "filtered" : "grouped"}
                </Badge>
              </label>
            ))}
          </div>
        ))}

        <CustomFieldPicker
          index={index}
          entities={entities}
          onAdd={(entity, ref, operator) => {
            const id = `${entity}::${refKey(ref)}`;
            if (filters.some((filter) => filter.id === id)) return;
            onChange([...filters, { id, entity, ref, operator }]);
          }}
        />
      </section>
    </div>
  );
}
