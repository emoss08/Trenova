import type { FilterItem, SingleFilterItem } from "@trenova/shared/types/data-table";

export type DataTableFacetValue = { value: string; label: string; count: number };

export type DataTableFacet = {
  key: string;
  label: string;
  /** The API field the facet filters on. */
  field: string;
  values: DataTableFacetValue[];
};

const facetFilterId = (field: string) => `facet:${field}`;

function isFieldFilter(item: FilterItem, field: string): item is SingleFilterItem {
  return item.type === "filter" && item.apiField === field;
}

/** The values currently chosen for a facet, whichever control chose them. */
export function facetSelection(filters: readonly FilterItem[], field: string): string[] {
  const values: string[] = [];
  for (const item of filters) {
    if (!isFieldFilter(item, field)) continue;
    if (item.operator === "in" && Array.isArray(item.value)) {
      values.push(...item.value.map(String));
    } else if (item.operator === "eq" && item.value != null) {
      values.push(String(item.value));
    }
  }
  return values;
}

export function activeFacetCount(
  filters: readonly FilterItem[],
  facets: readonly DataTableFacet[],
): number {
  return facets.reduce((total, facet) => total + facetSelection(filters, facet.field).length, 0);
}

/**
 * Toggles one value of a facet. A facet is one `in` filter on its field, so
 * the chips, the builder and saved views all read it as an ordinary filter.
 */
export function toggleFacetValue(
  filters: readonly FilterItem[],
  facet: DataTableFacet,
  value: string,
): FilterItem[] {
  const current = facetSelection(filters, facet.field);
  const next = current.includes(value)
    ? current.filter((entry) => entry !== value)
    : [...current, value];
  const others = filters.filter((item) => !isFieldFilter(item, facet.field));
  if (next.length === 0) {
    return others;
  }
  const item: SingleFilterItem = {
    id: facetFilterId(facet.field),
    type: "filter",
    connector: "and",
    field: facet.field,
    apiField: facet.field,
    label: facet.label,
    operator: "in",
    value: next,
    filterType: "select",
    filterOptions: facet.values.map((entry) => ({ value: entry.value, label: entry.label })),
  };
  return [...others, item];
}

export function clearFacets(
  filters: readonly FilterItem[],
  facets: readonly DataTableFacet[],
): FilterItem[] {
  const fields = new Set(facets.map((facet) => facet.field));
  return filters.filter((item) => !(item.type === "filter" && fields.has(item.apiField)));
}
