import type { FieldFilter } from "@trenova/shared/types/data-table";

export type TableBucketSpec = Readonly<Record<string, readonly string[]>>;

export function tableBucketFilters(spec: TableBucketSpec): FieldFilter[] {
  return Object.entries(spec).map(([field, values]) => ({
    field,
    operator: "in",
    value: [...values],
  }));
}

function filterValues(filter: FieldFilter): unknown[] | null {
  if (filter.operator === "in" && Array.isArray(filter.value)) {
    return filter.value as unknown[];
  }
  if (filter.operator === "eq") {
    return [filter.value];
  }
  return null;
}

function sameValues(values: readonly unknown[], expected: readonly string[]): boolean {
  const seen = new Set(values);
  return seen.size === expected.length && expected.every((value) => seen.has(value));
}

function matches(filters: readonly FieldFilter[], spec: TableBucketSpec): boolean {
  const fields = Object.keys(spec);
  if (filters.length !== fields.length) {
    return false;
  }
  const byField = new Map(filters.map((filter) => [filter.field, filter]));
  return fields.every((field) => {
    const filter = byField.get(field);
    const values = filter ? filterValues(filter) : null;
    return values !== null && sameValues(values, spec[field]);
  });
}

export function activeTableBucket<K extends string>(
  filters: readonly FieldFilter[],
  buckets: Readonly<Record<K, TableBucketSpec>>,
): K | null {
  const entry = (Object.entries(buckets) as [K, TableBucketSpec][]).find(([, spec]) =>
    matches(filters, spec),
  );
  return entry ? entry[0] : null;
}
