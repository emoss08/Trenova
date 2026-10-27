import {
  BookOpen01Icon,
  BracketsIcon,
  Building02Icon,
  Dataflow03Icon,
  File02Icon,
  File06Icon,
  FileCode02Icon,
  GitBranch01Icon,
  PuzzlePiece01Icon,
  ShieldTickIcon,
  TableIcon,
  Upload01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import type { LadingFileKind, NewPathProblem, SampleFormat } from "./studio-model";

/** What the add dialog offers: a kind of source file, a sample format, or files to upload. */
export type EntryChoice = LadingFileKind | SampleFormat | "upload";

export const FILE_KINDS: readonly LadingFileKind[] = [
  "map",
  "rules",
  "flow",
  "guide",
  "library",
  "partner",
];

export const SAMPLE_FORMATS: readonly SampleFormat[] = ["x12", "csv", "json", "document"];

type KindLook = { icon: IconComponent; ink: string; tile: string };

// Kinds have no order of severity, so each wears a categorical accent: the icon alone in the
// tree, on its tinted tile in the add dialog.
const LOOKS: Record<LadingFileKind | "other" | SampleFormat | "upload", KindLook> = {
  partner: {
    icon: Building02Icon,
    ink: "text-accent-indigo",
    tile: "bg-accent-indigo-subtle text-accent-indigo-on-subtle",
  },
  guide: {
    icon: BookOpen01Icon,
    ink: "text-accent-amber",
    tile: "bg-accent-amber-subtle text-accent-amber-on-subtle",
  },
  map: {
    icon: Dataflow03Icon,
    ink: "text-accent-teal",
    tile: "bg-accent-teal-subtle text-accent-teal-on-subtle",
  },
  rules: {
    icon: ShieldTickIcon,
    ink: "text-accent-rose",
    tile: "bg-accent-rose-subtle text-accent-rose-on-subtle",
  },
  flow: {
    icon: GitBranch01Icon,
    ink: "text-accent-violet",
    tile: "bg-accent-violet-subtle text-accent-violet-on-subtle",
  },
  library: {
    icon: PuzzlePiece01Icon,
    ink: "text-accent-emerald",
    tile: "bg-accent-emerald-subtle text-accent-emerald-on-subtle",
  },
  other: {
    icon: File02Icon,
    ink: "text-muted-foreground",
    tile: "bg-accent-slate-subtle text-accent-slate-on-subtle",
  },
  x12: {
    icon: File06Icon,
    ink: "text-accent-sky",
    tile: "bg-accent-sky-subtle text-accent-sky-on-subtle",
  },
  csv: {
    icon: TableIcon,
    ink: "text-accent-sky",
    tile: "bg-accent-sky-subtle text-accent-sky-on-subtle",
  },
  json: {
    icon: BracketsIcon,
    ink: "text-accent-sky",
    tile: "bg-accent-sky-subtle text-accent-sky-on-subtle",
  },
  document: {
    icon: FileCode02Icon,
    ink: "text-accent-sky",
    tile: "bg-accent-sky-subtle text-accent-sky-on-subtle",
  },
  upload: {
    icon: Upload01Icon,
    ink: "text-accent-slate",
    tile: "bg-accent-slate-subtle text-accent-slate-on-subtle",
  },
};

export function kindLook(kind: keyof typeof LOOKS): KindLook {
  return LOOKS[kind];
}

/** The headings of the file tree's sections. */
export const SECTION_LABELS = defineLabels({
  flows: "Flows",
  shared: "Shared",
  loose: "Not in any flow",
  samples: "Samples",
});

/** The key that picks each choice in the add palette while its filter is empty. */
export const CHOICE_KEYS: Record<EntryChoice, string> = {
  map: "m",
  rules: "r",
  flow: "f",
  guide: "g",
  library: "l",
  partner: "p",
  x12: "x",
  csv: "c",
  json: "j",
  document: "d",
  upload: "u",
};

/** A choice's name in the add dialog. */
export const CHOICE_LABELS: Record<EntryChoice, string> = defineLabels({
  map: "Map",
  rules: "Rules",
  flow: "Flow",
  guide: "Guide",
  library: "Library",
  partner: "Partner file",
  x12: "X12 message",
  csv: "CSV file",
  json: "JSON file",
  document: "Trenova document",
  upload: "Upload from your computer",
});

/** What each choice holds, under its name in the add dialog. */
export const CHOICE_DESCRIPTIONS: Record<EntryChoice, string> = defineLabels({
  map: "How one message becomes a Trenova document, or the reverse",
  rules: "What must be true of a map's document, and what happens when it is not",
  flow: "The conversation: what arrives, what Trenova answers and sends, and what it waits for",
  guide: "How this partner narrows a standard X12 guide",
  library: "Functions, crosswalks and constants other files use",
  partner: "The envelope, time zone, currency and settings; one per project",
  x12: "A message the partner sent, for maps and tests to run against",
  csv: "Rows under a header row, for a map that reads CSV",
  json: "A JSON document, for a map that reads JSON",
  document: "A document as Trenova holds it, for an outbound map's tests",
  upload: "The partner's real files, each added as a sample by its name",
});

/** The add button's label for each choice. */
export const ADD_LABELS: Record<Exclude<EntryChoice, "upload">, string> = defineLabels({
  map: "Add map",
  rules: "Add rules",
  flow: "Add flow",
  guide: "Add guide",
  library: "Add library",
  partner: "Add partner file",
  x12: "Add sample",
  csv: "Add sample",
  json: "Add sample",
  document: "Add sample",
});

/** Why a path cannot be used, in the words the tree's rename and the add palette both show. */
export function usePathProblemText(): Record<NewPathProblem, string> {
  const t = useT();
  return {
    empty: t("Name the file"),
    notRelative: t("Use a path inside the project, such as maps/status.lading"),
    notLading: t("A source file's name ends in .lading"),
    notSample: t("A sample's name ends in .x12, .edi, .txt, .csv or .json"),
    taken: t("The project already has a file at this path"),
  };
}
