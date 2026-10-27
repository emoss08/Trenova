import type {
  LadingCheck,
  LadingDiagnostic,
  LadingEdit,
  LadingFile,
  LadingLineage,
  LadingRunInput,
  LadingRunResult,
  LadingRunValue,
  LadingSample,
  LadingSetting,
  LadingSource,
} from "@trenova/shared/edi/lading";
import type { LadingVersion } from "@trenova/shared/types/lading";

/**
 * The version the studio opens: the one the address names when the project has it, else the
 * open draft, else the active version, else the newest.
 */
export function pickVersion(
  versions: readonly LadingVersion[],
  requested: string | null,
  activeVersionId: string | null | undefined,
): LadingVersion | null {
  if (requested) {
    const named = versions.find((version) => version.id === requested);
    if (named) return named;
  }
  const draft = versions.find((version) => version.status === "Draft");
  if (draft) return draft;
  const active = versions.find((version) => version.id === activeVersionId);
  if (active) return active;
  return versions[0] ?? null;
}

/** Only a draft is written to; every other version is frozen. */
export function isEditable(version: Pick<LadingVersion, "status"> | null): boolean {
  return version?.status === "Draft";
}

function sameEntries<T>(a: readonly T[], b: readonly T[], key: (item: T) => string): boolean {
  if (a.length !== b.length) return false;
  const seen = new Map(a.map((item) => [key(item), item] as const));
  return b.every((item) => {
    const other = seen.get(key(item));
    return other !== undefined && JSON.stringify(other) === JSON.stringify(item);
  });
}

/** Whether the working copy differs from what was saved. */
export function isDirty(
  saved: { files: readonly LadingFile[]; samples: readonly LadingSample[] },
  working: { files: readonly LadingFile[]; samples: readonly LadingSample[] },
): boolean {
  return (
    !sameEntries(saved.files, working.files, (file) => file.path) ||
    !sameEntries(saved.samples, working.samples, (sample) => sample.path)
  );
}

export type NewPathProblem = "empty" | "notRelative" | "notLading" | "taken";

/**
 * Why a path cannot be added, or null when it can: a source file's path ends in `.lading`, every
 * path is relative and inside the project, and no two files or samples share one.
 */
export function newPathProblem(
  path: string,
  kind: "file" | "sample",
  taken: readonly string[],
): NewPathProblem | null {
  const trimmed = path.trim();
  if (trimmed === "") return "empty";
  if (
    trimmed.startsWith("/") ||
    trimmed.includes("\\") ||
    trimmed.split("/").some((part) => part === ".." || part === "." || part === "")
  ) {
    return "notRelative";
  }
  if (kind === "file" && !trimmed.endsWith(".lading")) return "notLading";
  if (taken.includes(trimmed)) return "taken";
  return null;
}

/** The words of a file's name: `tender-in.lading` is `tender` and `in`. */
function fileWords(path: string): string[] {
  const base = (path.split("/").pop() ?? "file").replace(/\.lading$/, "");
  const words = base.split(/[^A-Za-z0-9]+/).filter(Boolean);
  return words.length > 0 ? words : ["file"];
}

/** A Lading name from a file's name, in the camelCase names take: `tender-in` is `tenderIn`. */
function camelName(path: string): string {
  const [first = "file", ...rest] = fileWords(path);
  const head = /^\d/.test(first) ? `file${first}` : first.charAt(0).toLowerCase() + first.slice(1);
  return head + rest.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join("");
}

/**
 * The starter text of a new file, from the header its folder suggests. A guide is named with
 * dots, as maps name it (`guides/acme-204.lading` is `acme.204`); every other file takes a
 * camelCase name.
 */
export function newFileSource(path: string, partner: string): string {
  const name = camelName(path);
  if (path.startsWith("maps/")) return `map ${name}\n  from x12.004010.204\n  to   loadTender\n`;
  if (path.startsWith("flows/")) return `flow ${name} with ${partner}\n`;
  if (path.startsWith("guides/")) {
    const guide = fileWords(path)
      .map((word) => word.toLowerCase())
      .join(".");
    return `guide ${guide} from x12.004010.204 ""\n`;
  }
  if (path.startsWith("rules/")) return `rules ${name} for ${name}\n`;
  return `library ${name}\n`;
}

export type Problem = LadingDiagnostic & { path: string };

const SEVERITY_ORDER = { error: 0, warning: 1, info: 2 } as const;

/** Every finding in the project, errors first, then by file and position. */
export function projectProblems(check: LadingCheck): Problem[] {
  const problems = check.files.flatMap((file) =>
    file.diagnostics.map((diagnostic) => ({ ...diagnostic, path: file.path })),
  );
  return problems.sort(
    (a, b) =>
      SEVERITY_ORDER[a.severity] - SEVERITY_ORDER[b.severity] ||
      a.path.localeCompare(b.path) ||
      a.range.from - b.range.from,
  );
}

/** How many errors and warnings each file has, for the file tree. */
export function problemCounts(
  check: LadingCheck,
): Map<string, { errors: number; warnings: number }> {
  const counts = new Map<string, { errors: number; warnings: number }>();
  for (const file of check.files) {
    let errors = 0;
    let warnings = 0;
    for (const diagnostic of file.diagnostics) {
      if (diagnostic.severity === "error") errors += 1;
      if (diagnostic.severity === "warning") warnings += 1;
    }
    counts.set(file.path, { errors, warnings });
  }
  return counts;
}

/** A sample as a run's input, read by its extension: X12 unless it is CSV or JSON. */
export function sampleInput(sample: LadingSample): LadingRunInput {
  const path = sample.path.toLowerCase();
  if (path.endsWith(".csv")) return { csv: sample.content };
  if (path.endsWith(".json")) {
    try {
      return { json: JSON.parse(sample.content) as unknown };
    } catch {
      return { json: sample.content };
    }
  }
  return { x12: sample.content };
}

/** The 1-based line a UTF-16 offset of a source falls on. */
export function lineOf(source: string, offset: number): number {
  let line = 1;
  const end = Math.min(offset, source.length);
  for (let index = 0; index < end; index += 1) {
    if (source.charCodeAt(index) === 10) line += 1;
  }
  return line;
}

/** Where a value was read from, as hover names it: `B204` for an element, else the path. */
export function sourceLabel(source: LadingSource): string {
  if (source.segment && source.element !== undefined) {
    return `${source.segment}${String(source.element).padStart(2, "0")}`;
  }
  return source.path;
}

/** Every value a run produced, with where it came from, for the editor's hover. */
export function runValues(result: LadingRunResult): LadingRunValue[] {
  const lineage: LadingLineage[] = [
    ...result.transactions.flatMap((transaction) => transaction.lineage),
    ...(result.output?.lineage ?? []),
  ];
  return lineage.map((entry) => ({
    path: entry.path,
    value: entry.value,
    sources: entry.sources.map(sourceLabel),
    steps: entry.steps,
  }));
}

/** A setting as its form field holds it: a checkbox's state, or the text typed. */
export type SettingDraft = boolean | string;

const NUMERIC_SETTINGS = new Set(["Int", "Duration"]);
const TEXT_SETTINGS = new Set(["Decimal", "Percent", "Money", "Text"]);

function isMeasure(type: string): boolean {
  return type !== "Bool" && !NUMERIC_SETTINGS.has(type) && !TEXT_SETTINGS.has(type);
}

/** The unit a measure setting is entered in: the stored value's, else its default's. */
export function settingUnit(setting: LadingSetting, stored: unknown): string {
  if (stored && typeof stored === "object" && "unit" in stored) {
    return String((stored as { unit: unknown }).unit);
  }
  return setting.default?.trim().match(/\s([A-Za-z]+)$/)?.[1] ?? "";
}

/** The field's starting state: the stored value, else blank, which keeps the default. */
export function settingDraft(setting: LadingSetting, stored: unknown): SettingDraft {
  if (setting.type === "Bool") {
    return typeof stored === "boolean" ? stored : setting.default?.trim() === "true";
  }
  if (stored === undefined || stored === null) return "";
  if (typeof stored === "object") {
    const record = stored as { value?: unknown; amount?: unknown };
    return String(record.value ?? record.amount ?? "");
  }
  return String(stored);
}

/**
 * The value saved for a field, in the shape the server takes for the setting's type: null keeps
 * the default, and undefined means the text is not a value of that type.
 */
export function settingValue(setting: LadingSetting, draft: SettingDraft, unit: string): unknown {
  if (setting.type === "Bool") return draft === true;
  const text = String(draft).trim().replaceAll("_", "");
  if (text === "") return null;
  if (NUMERIC_SETTINGS.has(setting.type)) {
    const number = Number(text);
    return Number.isInteger(number) ? number : undefined;
  }
  if (setting.type === "Text") return String(draft);
  if (TEXT_SETTINGS.has(setting.type)) {
    return Number.isFinite(Number(text)) ? text : undefined;
  }
  if (isMeasure(setting.type)) {
    const number = Number(text);
    return Number.isFinite(number) && unit !== "" ? { value: number, unit } : undefined;
  }
  return undefined;
}

/**
 * The files with a rename's or a refactoring's edits applied: each file's edits from the end
 * back, so earlier offsets still hold. An edit to a path the project lacks creates that file, as
 * Move to library does.
 */
export function applyFileEdits(
  files: readonly LadingFile[],
  edits: readonly LadingEdit[],
): LadingFile[] {
  const byPath = new Map<string, LadingEdit[]>();
  for (const edit of edits) byPath.set(edit.path, [...(byPath.get(edit.path) ?? []), edit]);
  const edit = (source: string, changes: readonly LadingEdit[]) =>
    [...changes]
      .sort((a, b) => b.range.from - a.range.from)
      .reduce(
        (text, change) =>
          text.slice(0, change.range.from) + change.insert + text.slice(change.range.to),
        source,
      );
  const next = files.map((file) => {
    const changes = byPath.get(file.path);
    return changes ? { path: file.path, source: edit(file.source, changes) } : file;
  });
  for (const [path, changes] of byPath) {
    if (!files.some((file) => file.path === path)) next.push({ path, source: edit("", changes) });
  }
  return next;
}
