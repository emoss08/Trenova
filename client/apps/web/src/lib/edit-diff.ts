import { translate } from "@trenova/shared/i18n/runtime";

/** How long a value may run in a change review before it is cut. */
const VALUE_PREVIEW_LENGTH = 42;

/** What was added to and removed from a list of plain values. */
export function listDelta<T extends string | number>(
  before: readonly T[],
  after: readonly T[],
): { added: T[]; removed: T[] } {
  const was = new Set(before);
  const now = new Set(after);
  return {
    added: after.filter((value) => !was.has(value)),
    removed: before.filter((value) => !now.has(value)),
  };
}

/** Whether a value is a list a review can show item by item. */
export function isPlainList(value: unknown): value is (string | number)[] {
  return (
    Array.isArray(value) &&
    value.every((item) => typeof item === "string" || typeof item === "number")
  );
}

/** A value as one short line of a change review. */
export function previewValue(value: unknown): string {
  if (typeof value === "boolean") {
    return value ? translate("On") : translate("Off");
  }
  if (Array.isArray(value)) {
    return translate("{0} selected", value.length);
  }
  if (value === null || value === undefined || value === "") {
    return translate("Empty");
  }
  if (typeof value === "object") {
    return translate("Changed");
  }
  const text = typeof value === "string" ? value : String(value as number | bigint | symbol);
  return text.length > VALUE_PREVIEW_LENGTH ? `${text.slice(0, VALUE_PREVIEW_LENGTH - 2)}…` : text;
}
