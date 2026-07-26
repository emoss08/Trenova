import { bandIsSet, type ReportDashboardFilter, type ReportIR } from "@/types/report";
import { refKey } from "./dashboard-state";
import { splitOutputId } from "../../builder/_components/builder-state";

/**
 * A category picked by clicking a chart. It narrows every tile on the same
 * entity, exactly like a configured dashboard filter, but is transient: it
 * lives for the session and is cleared from a chip in the filter bar.
 */
export type CrossFilter = {
  filter: ReportDashboardFilter;
  value: unknown;
  /** What the chart displayed, for the chip. */
  label: string;
};

export type CrossFilterState = Record<string, CrossFilter>;

/**
 * Resolves the chart's category column back to the dimension it groups by.
 *
 * The grouping ref is used verbatim — deliberately *not* normalised to the
 * foreign key the way the filter picker does. A chart grouped by
 * `serviceType.code` yields a code, and matching that against `serviceTypeId`
 * would compare a code to an identifier and quietly return nothing.
 *
 * Returns null when the click cannot be expressed as an equality: a bucketed
 * date and a banded number both need a half-open range, and a transformed
 * dimension compares against a value the raw column never holds.
 */
export function crossFilterFor(
  ir: ReportIR,
  outputColumnId: string | undefined,
): ReportDashboardFilter | null {
  if (!outputColumnId) return null;

  const [baseId] = splitOutputId(outputColumnId);
  const column = ir.columns.find((candidate) => candidate.id === baseId);
  if (!column?.ref || column.kind !== "dimension") return null;
  if (column.bucket || column.transform || bandIsSet(column.band)) return null;

  return {
    id: `${ir.entity}::${refKey(column.ref)}`,
    entity: ir.entity,
    ref: column.ref,
    operator: "eq",
  };
}

/** Toggling: clicking the active category again clears it. */
export function toggleCrossFilter(
  state: CrossFilterState,
  filter: ReportDashboardFilter,
  value: unknown,
  label: string,
): CrossFilterState {
  const existing = state[filter.id];
  if (existing && String(existing.value) === String(value)) {
    return removeCrossFilter(state, filter.id);
  }
  return { ...state, [filter.id]: { filter, value, label } };
}

export function removeCrossFilter(state: CrossFilterState, id: string): CrossFilterState {
  return Object.fromEntries(Object.entries(state).filter(([key]) => key !== id));
}

/**
 * Cross-filters are applied through the same pipeline as configured filters,
 * so entity matching and override-rather-than-stack behaviour are shared.
 */
export function crossFilterDefinitions(state: CrossFilterState): ReportDashboardFilter[] {
  return Object.values(state).map((entry) => entry.filter);
}

export function crossFilterValues(state: CrossFilterState): Record<string, unknown> {
  return Object.fromEntries(Object.entries(state).map(([id, entry]) => [id, entry.value]));
}

/** The value active for one chart's category column, for selection feedback. */
export function selectedCategory(
  state: CrossFilterState,
  filter: ReportDashboardFilter | null,
): unknown {
  return filter ? state[filter.id]?.value : undefined;
}
