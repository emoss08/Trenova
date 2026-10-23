import type { FieldChange } from "@trenova/shared/types/formula-template";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type ChangedFieldRow = {
  path: string;
  label: string;
  summary: string;
};

const FIELD_LABELS: Record<string, string> = defineLabels({
  name: "Name",
  description: "Description",
  imports: "Pinned libraries",
});

const SEGMENT_LABELS: Record<string, string> = {
  imports: "Pinned library",
  path: "path",
  version: "version",
};

function formatSide(value: unknown): string {
  if (value === null || value === undefined || value === "") return "empty";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  return JSON.stringify(value);
}

function labelFor(path: string): string {
  if (FIELD_LABELS[path]) return FIELD_LABELS[path];

  const segments = path.split(".");
  const words: string[] = [];
  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i];
    if (/^\d+$/.test(segment)) {
      words.push(String(Number(segment) + 1));
      continue;
    }
    words.push(SEGMENT_LABELS[segment] ?? segment);
  }
  const [head, ...rest] = words;
  return [head, ...rest].join(" ");
}

/**
 * Turns the server's change map into rows a reviewer can scan. The source is
 * left out because it gets a line diff of its own; everything else is one line
 * of "was → now".
 */
export function describeChangedFields(changes: Record<string, FieldChange>): ChangedFieldRow[] {
  return Object.entries(changes)
    .filter(([path]) => path !== "source")
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([path, change]) => ({
      path,
      label: labelFor(path),
      summary: `${formatSide(change.from)} → ${formatSide(change.to)}`,
    }));
}
