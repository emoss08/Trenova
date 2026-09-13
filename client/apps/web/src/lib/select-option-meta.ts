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

export function selectOptionMetaBoolean(option: SelectOption, key: string): boolean {
  return option.meta?.[key] === true;
}
