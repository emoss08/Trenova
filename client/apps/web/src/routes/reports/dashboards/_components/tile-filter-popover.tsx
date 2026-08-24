import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Label } from "@trenova/shared/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import {
  operatorsForFieldType,
  type ReportDashboardFilter,
  type ReportFieldRef,
  type ReportFilterGroup,
  type ReportIR,
} from "@/types/report";
import { FilterIcon } from "lucide-react";
import { useMemo } from "react";
import {
  preferReferenceField,
  refLabel,
  refTargetEntity,
  resolveField,
  type CatalogIndex,
} from "../../builder/_components/builder-state";
import { FilterValueEditor } from "../../builder/_components/filter-value-editor";
import { refKey } from "./dashboard-state";

type TileFilterPopoverProps = {
  index: CatalogIndex;
  ir: ReportIR;
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
};

function collectRefs(group: ReportFilterGroup | null | undefined, into: ReportFieldRef[]) {
  if (!group) return;
  for (const leaf of group.filters ?? []) {
    if (!leaf.param) into.push(leaf.ref);
  }
  for (const nested of group.groups ?? []) collectRefs(nested, into);
}

/**
 * The fields one tile can be narrowed by: what its own report filters on and
 * breaks down by, normalised to the reference beside a grouping so the control
 * offers real records.
 */
export function useTileFilterFields(
  index: CatalogIndex | null,
  ir: ReportIR | null,
): ReportDashboardFilter[] {
  return useMemo(() => {
    if (!index || !ir) return [];

    const refs: ReportFieldRef[] = [];
    collectRefs(ir.filters, refs);
    for (const column of ir.columns) {
      if (column.kind === "dimension" && column.ref) refs.push(column.ref);
    }

    const byKey = new Map<string, ReportDashboardFilter>();
    for (const raw of refs) {
      const ref = preferReferenceField(index, ir.entity, raw);
      const field = resolveField(index, ir.entity, ref);
      if (!field?.filterable || !field.accessible) continue;

      const id = refKey(ref);
      if (byKey.has(id)) continue;

      const choices = operatorsForFieldType(field.type);
      const operator = (choices.find((c) => c.value === "in") ?? choices[0])?.value ?? "eq";
      byKey.set(id, { id, entity: ir.entity, ref, operator });
    }

    return [...byKey.values()].sort((a, b) =>
      refLabel(index, a.entity, a.ref).localeCompare(refLabel(index, b.entity, b.ref)),
    );
  }, [index, ir]);
}

export function activeFilterCount(values: Record<string, unknown>): number {
  return Object.values(values).filter((value) => {
    if (value === undefined || value === null || value === "") return false;
    return !(Array.isArray(value) && value.length === 0);
  }).length;
}

/**
 * Narrows a single tile without touching the rest of the dashboard — the
 * counterpart to the shared bar, for when a question is about one report.
 */
export function TileFilterPopover({ index, ir, values, onChange }: TileFilterPopoverProps) {
  const fields = useTileFilterFields(index, ir);
  const active = activeFilterCount(values);

  if (fields.length === 0) return null;

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              "size-6 shrink-0",
              active === 0 && "opacity-0 transition-opacity group-hover/tile:opacity-100",
            )}
            aria-label="Filter this tile"
            title="Filter this tile"
          >
            <FilterIcon className={cn("size-3.5", active > 0 && "text-primary")} />
          </Button>
        }
      />
      {/*
        Opening with the mouse must not focus the first control: it is a date
        field that opens its own suggestion list on focus, so the popover
        appeared with a dropdown already covering it. Keyboard opens still
        move focus, which is what keyboard users need.
      */}
      <PopoverContent
        className="w-72 p-2"
        align="end"
        initialFocus={(openType) => openType === "keyboard"}
      >
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <span className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
              Filter this tile
            </span>
            {active > 0 && (
              <Badge variant="secondary" className="text-[10px]">
                {active}
              </Badge>
            )}
            <div className="flex-1" />
            {active > 0 && (
              <Button
                variant="ghost"
                size="sm"
                className="text-2xs h-6"
                onClick={() => onChange({})}
              >
                Clear
              </Button>
            )}
          </div>

          {fields.map((filter) => {
            const field = resolveField(index, filter.entity, filter.ref);
            return (
              <div key={filter.id} className="flex flex-col gap-1">
                <Label className="text-muted-foreground text-xs">
                  {refLabel(index, filter.entity, filter.ref)}
                </Label>
                <FilterValueEditor
                  field={field}
                  operator={filter.operator}
                  value={values[filter.id]}
                  refEntityKey={refTargetEntity(index, filter.entity, filter.ref)}
                  onChange={(value) => onChange({ ...values, [filter.id]: value })}
                />
              </div>
            );
          })}
        </div>
      </PopoverContent>
    </Popover>
  );
}
