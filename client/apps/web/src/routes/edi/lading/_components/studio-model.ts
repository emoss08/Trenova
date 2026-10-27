import type {
  LadingCheck,
  LadingDiagnostic,
  LadingEdit,
  LadingFile,
  LadingLineage,
  LadingMap,
  LadingRunInput,
  LadingRunResult,
  LadingRunValue,
  LadingSample,
  LadingSetting,
  LadingSource,
} from "@trenova/shared/edi/lading";
import type { LadingReplay, LadingVersion } from "@trenova/shared/types/lading";

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

/** The kinds of source file, each named by its header's first keyword. */
export type LadingFileKind = "partner" | "guide" | "map" | "rules" | "flow" | "library";

/** The folder the studio puts each kind of new file in; the partner file sits at the root. */
export const FILE_FOLDERS: Record<LadingFileKind, string> = {
  partner: "",
  guide: "guides/",
  map: "maps/",
  rules: "rules/",
  flow: "flows/",
  library: "lib/",
};

/** The one file that holds the partner's envelope, zone, currency and settings. */
export const PARTNER_FILE = "partner.lading";

export const SAMPLE_FOLDER = "samples/";

/** How a sample is read: X12, CSV, JSON, or a Trenova document for an outbound test. */
export type SampleFormat = "x12" | "csv" | "json" | "document";

/** The ending a new sample of each format takes. */
export const SAMPLE_ENDINGS: Record<SampleFormat, string> = {
  x12: ".x12",
  csv: ".csv",
  json: ".json",
  document: ".document.json",
};

/**
 * A sample's format from its name: `.x12`, `.edi` and `.txt` are X12, `.document.json` a
 * Trenova document, `.json` and `.csv` as named. Null for any other name.
 */
export function sampleFormat(path: string): SampleFormat | null {
  const lower = path.toLowerCase();
  if (lower.endsWith(".document.json")) return "document";
  if (lower.endsWith(".json")) return "json";
  if (lower.endsWith(".csv")) return "csv";
  if (lower.endsWith(".x12") || lower.endsWith(".edi") || lower.endsWith(".txt")) return "x12";
  return null;
}

export type NewPathProblem = "empty" | "notRelative" | "notLading" | "notSample" | "taken";

/**
 * Why a path cannot be added, or null when it can: a source file's path ends in `.lading`, a
 * sample's in an ending a sample is read by, every path is relative and inside the project, and
 * no two files or samples share one.
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
  if (kind === "sample" && sampleFormat(trimmed) === null) return "notSample";
  if (taken.includes(trimmed)) return "taken";
  return null;
}

/**
 * The path a new entry is saved at: its kind's folder, the name as typed, and the kind's ending,
 * which a name that already carries it does not repeat.
 */
export function entryPath(kind: LadingFileKind | SampleFormat, name: string): string {
  const trimmed = name.trim().replace(/^\/+/, "");
  if (trimmed === "") return "";
  if (kind in SAMPLE_ENDINGS) {
    const ending = SAMPLE_ENDINGS[kind as SampleFormat];
    const base = trimmed.toLowerCase().endsWith(ending) ? trimmed : `${trimmed}${ending}`;
    return `${SAMPLE_FOLDER}${base}`;
  }
  if (kind === "partner") return PARTNER_FILE;
  const base = trimmed.endsWith(".lading") ? trimmed : `${trimmed}.lading`;
  return `${FILE_FOLDERS[kind as LadingFileKind]}${base}`;
}

/** The words of a name: `tender-in.lading` is `tender` and `in`, `tenderIn` is the same. */
function nameWords(text: string): string[] {
  return text
    .replace(/\.lading$/, "")
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .split(/[^A-Za-z0-9]+/)
    .filter(Boolean)
    .map((word) => word.toLowerCase());
}

/** The words of a file's name, without its folders. */
function fileWords(path: string): string[] {
  const words = nameWords(path.split("/").pop() ?? "");
  return words.length > 0 ? words : ["file"];
}

/**
 * A Lading name from a file's name or any text, in the camelCase names take: `tender-in` is
 * `tenderIn`, `Acme Manufacturing` is `acmeManufacturing`.
 */
export function ladingName(path: string): string {
  const [first = "file", ...rest] = fileWords(path);
  const head = /^\d/.test(first) ? `file${first}` : first;
  return head + rest.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join("");
}

/** A name as a file name: `tenderIn` is `tender-in`. */
export function kebabName(name: string): string {
  return nameWords(name).join("-");
}

/** Text as it may sit inside a Lading string: no closing quote and no interpolation braces. */
function quoted(text: string): string {
  return `"${text.replaceAll('"', "'").replace(/[{}]/g, "")}"`;
}

/** What a new source file starts as, beyond the name its path gives it. */
export type FileStarter =
  | { kind: "guide"; base: string; title: string }
  | { kind: "map"; direction: "inbound"; input: "x12"; guide: string; document: string }
  | { kind: "map"; direction: "inbound"; input: "csv" | "json"; document: string }
  | { kind: "map"; direction: "outbound"; document: string; guide: string }
  | { kind: "rules"; map: string }
  | { kind: "flow"; partner: string }
  | { kind: "library" };

/**
 * The starter text of a new source file: its header, named from its path. A guide is named with
 * dots, as maps name it (`guides/acme-204.lading` is `acme.204`); every other file takes a
 * camelCase name. A map reading CSV starts with one column to show the shape.
 */
export function starterSource(path: string, starter: FileStarter): string {
  const name = ladingName(path);
  switch (starter.kind) {
    case "guide":
      // i18n-ignore: Lading source written into the new file
      return `guide ${fileWords(path).join(".")} from ${starter.base} ${quoted(starter.title)}\n`;
    case "map": {
      if (starter.direction === "outbound") {
        // i18n-ignore: Lading source written into the new file
        return `map ${name}\n  from ${starter.document}\n  to   ${starter.guide}\n`;
      }
      if (starter.input === "csv") {
        // i18n-ignore: Lading source written into the new file
        return `map ${name}\n  from csv {\n    reference: Text\n  }\n  to   ${starter.document}\n`;
      }
      const from = starter.input === "json" ? "json" : starter.guide;
      // i18n-ignore: Lading source written into the new file
      return `map ${name}\n  from ${from}\n  to   ${starter.document}\n`;
    }
    case "rules":
      // i18n-ignore: Lading source written into the new file
      return `rules ${name} for ${starter.map}\n`;
    case "flow":
      // i18n-ignore: Lading source written into the new file
      return `flow ${name} with ${starter.partner}\n`;
    case "library":
      // i18n-ignore: Lading source written into the new file
      return `library ${name}\n`;
  }
}

/** The stem a document's files are named by: `loadTender` is `tender`. */
export function documentStem(document: string): string {
  return nameWords(document).pop() ?? document;
}

/** A name not yet taken: the name itself, else the name with the first free number after it. */
export function freeName(
  kind: LadingFileKind | SampleFormat,
  name: string,
  taken: readonly string[],
): string {
  if (!taken.includes(entryPath(kind, name))) return name;
  for (let n = 2; ; n += 1) {
    const next = `${name}-${n}`;
    if (!taken.includes(entryPath(kind, next))) return next;
  }
}

/**
 * The project with an entry moved to a new path. A sample's tests read it by its quoted path, so
 * every file's `"old/path"` becomes `"new/path"`; a source file is referred to by its declared
 * name, which a move leaves alone.
 */
export function renameEntry(
  project: { files: readonly LadingFile[]; samples: readonly LadingSample[] },
  from: string,
  to: string,
): { files: LadingFile[]; samples: LadingSample[] } {
  const isSample = project.samples.some((sample) => sample.path === from);
  const files = project.files.map((file) => {
    const path = file.path === from ? to : file.path;
    const source = isSample ? file.source.replaceAll(`"${from}"`, `"${to}"`) : file.source;
    return path === file.path && source === file.source ? file : { path, source };
  });
  const samples = project.samples.map((sample) =>
    sample.path === from ? { ...sample, path: to } : sample,
  );
  return { files, samples };
}

/** What a row in the file tree is: a kind of source file, a file without a header, or a sample. */
export type TreeKind = LadingFileKind | "other" | "sample";

/** One row of the file tree, with what the check knows about it. */
export type TreeEntry = {
  path: string;
  kind: TreeKind;
  /** The path as the row shows it: without its kind's folder, nor `.lading`. */
  label: string;
  map?: LadingMap;
  rulesFor?: string;
  guideBase?: string | null;
  format?: SampleFormat;
};

/** One flow in the file tree, with the maps and rules its clauses use under it. */
export type FlowBranch = { flow: TreeEntry; children: TreeEntry[] };

/**
 * The project as the file tree shows it: the partner file, each flow with the maps and rules it
 * uses, the guides and libraries every file shares, the files no flow reaches (a map or rules
 * file no clause names, a file without a header), and the samples.
 */
export type ProjectMap = {
  partner: TreeEntry[];
  flows: FlowBranch[];
  shared: TreeEntry[];
  loose: TreeEntry[];
  samples: TreeEntry[];
};

const FILE_KINDS = new Set<string>(["partner", "guide", "map", "rules", "flow", "library"]);

function rowLabel(path: string, folder: string): string {
  const inside = folder !== "" && path.startsWith(folder) ? path.slice(folder.length) : path;
  return inside.replace(/\.lading$/, "");
}

const byPath = (a: TreeEntry, b: TreeEntry) => a.path.localeCompare(b.path);

/**
 * The project arranged by what uses what, from the check. A file's kind is the one its header
 * names, wherever the file sits. Under a flow come the map and rules of each clause in clause
 * order, each map followed by any other rules file written for it; a map two flows use appears
 * under both. Guides come before libraries in the shared files.
 */
export function projectMap(
  files: readonly LadingFile[],
  samples: readonly LadingSample[],
  check: LadingCheck,
): ProjectMap {
  const kinds = new Map(check.files.map((file) => [file.path, file.kind] as const));
  const maps = new Map(check.maps.map((map) => [map.path, map] as const));
  const rules = new Map(check.rules.map((rule) => [rule.path, rule.map] as const));
  const guides = new Map(
    check.guides.flatMap((guide) => (guide.path ? [[guide.path, guide.base] as const] : [])),
  );
  const entries = new Map<string, TreeEntry>();
  for (const file of files) {
    const found = kinds.get(file.path) ?? "";
    const kind: TreeKind = FILE_KINDS.has(found) ? (found as LadingFileKind) : "other";
    entries.set(file.path, {
      path: file.path,
      kind,
      label: rowLabel(file.path, kind === "other" ? "" : FILE_FOLDERS[kind]),
      map: maps.get(file.path),
      rulesFor: rules.get(file.path),
      guideBase: guides.get(file.path),
    });
  }
  const all = [...entries.values()];
  const ofKind = (kind: TreeKind) => all.filter((entry) => entry.kind === kind).sort(byPath);

  const mapPath = new Map(check.maps.map((map) => [map.name, map.path] as const));
  const rulesPath = new Map(check.rules.map((rule) => [rule.name, rule.path] as const));
  const rulesOf = (mapName: string) =>
    check.rules.filter((rule) => rule.map === mapName).map((rule) => rule.path);

  const reached = new Set<string>();
  const flows = ofKind("flow").map((flow) => {
    const clauses = check.flows.find((item) => item.path === flow.path)?.clauses ?? [];
    const paths: string[] = [];
    const take = (path: string | undefined) => {
      if (path && entries.has(path) && !paths.includes(path)) paths.push(path);
    };
    for (const clause of clauses) {
      if (clause.map) {
        take(mapPath.get(clause.map));
        if (clause.rules) take(rulesPath.get(clause.rules));
        for (const path of rulesOf(clause.map)) take(path);
      } else if (clause.rules) {
        take(rulesPath.get(clause.rules));
      }
    }
    for (const path of paths) reached.add(path);
    return { flow, children: paths.map((path) => entries.get(path) as TreeEntry) };
  });

  const loose = all
    .filter(
      (entry) =>
        entry.kind === "other" ||
        ((entry.kind === "map" || entry.kind === "rules") && !reached.has(entry.path)),
    )
    .sort(byPath);

  return {
    partner: ofKind("partner"),
    flows,
    shared: [...ofKind("guide"), ...ofKind("library")],
    loose,
    samples: samples
      .map((sample): TreeEntry => ({
        path: sample.path,
        kind: "sample",
        label: rowLabel(sample.path, SAMPLE_FOLDER),
        format: sampleFormat(sample.path) ?? "x12",
      }))
      .sort(byPath),
  };
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

/**
 * Entries under the file they belong to. The files keep the order of their first entry, so with
 * `projectProblems`' order a file with an error comes before one with only warnings.
 */
export function byFile<T extends { path: string }>(
  entries: readonly T[],
): { path: string; entries: T[] }[] {
  const byPath = new Map<string, T[]>();
  for (const entry of entries) byPath.set(entry.path, [...(byPath.get(entry.path) ?? []), entry]);
  return [...byPath].map(([path, inFile]) => ({ path, entries: inFile }));
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

/**
 * A sample as a run's input, read by its format: a Trenova document for an outbound map, CSV,
 * JSON, and X12 for any other name.
 */
export function sampleInput(sample: LadingSample): LadingRunInput {
  const format = sampleFormat(sample.path);
  if (format === "csv") return { csv: sample.content };
  if (format === "json" || format === "document") {
    let value: unknown = sample.content;
    try {
      value = JSON.parse(sample.content);
    } catch {
      value = sample.content;
    }
    return format === "document" ? { document: value } : { json: value };
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

/** Where a version stands on one step of its way to live traffic. */
export type StageState = "done" | "running" | "attention" | "failed" | "pending" | "skipped";

export type ReleaseStage =
  | { key: "saved"; state: StageState }
  | {
      key: "replayed";
      state: StageState;
      clean: number;
      total: number;
      reason?: "noActive" | "isActive" | "noInbound";
    }
  | { key: "certified"; state: StageState }
  | { key: "shadow"; state: StageState }
  | { key: "live"; state: StageState; superseded: boolean };

const CERTIFIED_OR_LATER = new Set<LadingVersion["status"]>([
  "Certified",
  "Shadow",
  "Active",
  "Superseded",
]);

/**
 * The steps between a version and live traffic, each as far as this version has come: saved,
 * replayed (each inbound map with a finished replay and no new rejection, against the active
 * version), certified, shadowed (optional) and live. A step that cannot apply is skipped.
 */
export function releaseStages(input: {
  version: Pick<LadingVersion, "id" | "status" | "shadowUntil">;
  activeVersionId: string | null | undefined;
  dirty: boolean;
  inboundMaps: readonly string[];
  replays: readonly Pick<LadingReplay, "map" | "status" | "summary">[];
}): ReleaseStage[] {
  const { version, activeVersionId, dirty, inboundMaps, replays } = input;
  const certified = CERTIFIED_OR_LATER.has(version.status);

  const replayed = (): ReleaseStage => {
    const total = inboundMaps.length;
    const base = { key: "replayed" as const, total };
    if (!activeVersionId) return { ...base, state: "skipped", clean: 0, reason: "noActive" };
    if (activeVersionId === version.id) {
      return { ...base, state: "skipped", clean: 0, reason: "isActive" };
    }
    if (total === 0) return { ...base, state: "skipped", clean: 0, reason: "noInbound" };
    const finished = (map: string) =>
      replays.filter((replay) => replay.map === map && replay.status === "Completed");
    const clean = inboundMaps.filter((map) =>
      finished(map).some((replay) => (replay.summary?.newlyRejected ?? 0) === 0),
    ).length;
    const rejecting = inboundMaps.some(
      (map) =>
        finished(map).length > 0 &&
        finished(map).every((replay) => (replay.summary?.newlyRejected ?? 0) > 0),
    );
    const running = replays.some((replay) => replay.status === "Running");
    const state: StageState =
      clean === total
        ? "done"
        : running
          ? "running"
          : rejecting
            ? "failed"
            : clean > 0
              ? "attention"
              : "pending";
    return { ...base, state, clean };
  };

  return [
    { key: "saved", state: version.status !== "Draft" || !dirty ? "done" : "attention" },
    replayed(),
    { key: "certified", state: certified ? "done" : "pending" },
    {
      key: "shadow",
      state:
        version.status === "Shadow"
          ? "running"
          : version.shadowUntil != null
            ? "done"
            : version.status === "Active" || version.status === "Superseded"
              ? "skipped"
              : "pending",
    },
    {
      key: "live",
      state: version.status === "Active" || version.status === "Superseded" ? "done" : "pending",
      superseded: version.status === "Superseded",
    },
  ];
}
