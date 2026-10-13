import type { SelectOption } from "@/lib/graphql/select-options";

/**
 * A select option's `meta` is an untyped JSON bag the server fills per
 * resource, so every read has to narrow before use. These readers return the
 * empty value rather than throwing: a picker that renders without one optional
 * detail beats a picker that renders nothing.
 */
export function selectOptionMetaString(option: SelectOption, key: string): string {
  const value = option.meta?.[key];
  return typeof value === "string" ? value : "";
}

export function selectOptionMetaNumber(option: SelectOption, key: string): number | null {
  const value = option.meta?.[key];
  return typeof value === "number" ? value : null;
}

/** The meta value when it is one of `values`, so a renamed or missing enum reads as absent rather than as a guess. */
export function selectOptionMetaEnum<T extends string>(
  option: SelectOption,
  key: string,
  values: readonly T[],
): T | null {
  const value = option.meta?.[key];
  return values.find((candidate) => candidate === value) ?? null;
}

export function selectOptionMetaBoolean(option: SelectOption, key: string): boolean {
  return option.meta?.[key] === true;
}

/** How a record reads wherever it is named in a list: its code before its name when it has one. */
export function recordOptionLabel(option: SelectOption): string {
  const code = selectOptionMetaString(option, "code");
  return code ? `${code} - ${option.label}` : option.label;
}
