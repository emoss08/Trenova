import { useT } from "@trenova/shared/i18n/use-t";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  ChevronRightIcon,
  Copy01Icon,
  DotsHorizontalIcon,
  Edit03Icon,
  PlusIcon,
  SearchSmIcon,
  Trash01Icon,
  Upload01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { LadingCheck, LadingFile, LadingSample } from "@trenova/shared/edi/lading";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState, type FocusEvent, type ReactNode } from "react";
import { LadingAddEntryDialog, type NewEntry } from "./lading-add-entry-dialog";
import { SECTION_LABELS, usePathProblemText, type EntryChoice } from "./lading-file-kinds";
import { KindGlyph } from "./lading-kind-glyph";
import {
  newPathProblem,
  problemCounts,
  projectMap,
  type FlowBranch,
  type TreeEntry,
} from "./studio-model";

type Counts = { errors: number; warnings: number };

/**
 * The project as Lading reads it: the partner file, each flow with the maps and rules it uses,
 * the guides and libraries every file shares, the files no flow reaches, and the samples. Each
 * file shows what the check knows of it and its errors and warnings; whether the whole project
 * checks clean is the version toolbar's and the editor status bar's to say. A draft adds,
 * renames and removes entries here.
 */
export function LadingFileTree({
  files,
  samples,
  check,
  openPath,
  editable,
  partnerLabel,
  zone,
  onOpen,
  onAdd,
  onRename,
  onRemove,
}: {
  files: readonly LadingFile[];
  samples: readonly LadingSample[];
  check: LadingCheck;
  openPath: string;
  editable: boolean;
  partnerLabel: string;
  zone: string;
  onOpen: (path: string) => void;
  onAdd: (entries: NewEntry[]) => void;
  onRename: (from: string, to: string) => void;
  onRemove: (path: string) => void;
}) {
  const t = useT();
  const filterRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [closed, setClosed] = useState<ReadonlySet<string>>(() => new Set());
  const [renaming, setRenaming] = useState<string | null>(null);
  const [adding, setAdding] = useState<{
    open: boolean;
    session: number;
    choice: EntryChoice | null;
  }>({ open: false, session: 0, choice: null });

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (
        (event.metaKey || event.ctrlKey) &&
        !event.shiftKey &&
        !event.altKey &&
        event.key.toLowerCase() === "p"
      ) {
        event.preventDefault();
        filterRef.current?.focus();
        filterRef.current?.select();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const taken = [...files.map((file) => file.path), ...samples.map((sample) => sample.path)];
  const counts = problemCounts(check);
  const tree = projectMap(files, samples, check);
  const needle = query.trim().toLowerCase();
  const matches = (entry: TreeEntry) => entry.path.toLowerCase().includes(needle);
  const keep = (entries: readonly TreeEntry[]) => (needle ? entries.filter(matches) : entries);
  const flows: FlowBranch[] = needle
    ? tree.flows.flatMap((branch) => {
        const children = branch.children.filter(matches);
        return matches(branch.flow) || children.length > 0 ? [{ ...branch, children }] : [];
      })
    : tree.flows;
  const partner = keep(tree.partner);
  const shared = keep(tree.shared);
  const loose = keep(tree.loose);
  const shownSamples = keep(tree.samples);
  const nothing =
    needle !== "" &&
    partner.length + flows.length + shared.length + loose.length + shownSamples.length === 0;

  const openAdd = (choice: EntryChoice | null) =>
    setAdding((current) => ({ open: true, session: current.session + 1, choice }));
  const isOpen = (key: string) => needle !== "" || !closed.has(key);
  const toggle = (key: string) =>
    setClosed((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });

  const row = (entry: TreeEntry, id: string, depth = 1) =>
    renaming === id ? (
      <RenameRow
        key={id}
        entry={entry}
        depth={depth}
        taken={taken.filter((path) => path !== entry.path)}
        onDone={(to) => {
          setRenaming(null);
          if (to !== null && to !== entry.path) onRename(entry.path, to);
        }}
      />
    ) : (
      <TreeRow
        key={id}
        entry={entry}
        depth={depth}
        counts={counts.get(entry.path)}
        open={entry.path === openPath}
        editable={editable}
        onOpen={onOpen}
        onRename={() => setRenaming(id)}
        onRemove={onRemove}
      />
    );

  return (
    <nav aria-label={t("Project files")} className="flex h-full min-h-0 flex-col">
      <div className="flex flex-col gap-2 border-b px-3 pt-2.5 pb-2">
        <div className="flex items-center gap-2">
          <span className="min-w-0 flex-1 truncate text-sm font-semibold">
            {check.partner?.label ?? partnerLabel}
          </span>
          {editable ? (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={t("Add to the project")}
                    onClick={() => openAdd(null)}
                  />
                }
              >
                <PlusIcon className="size-3.5" />
              </TooltipTrigger>
              <TooltipContent>{t("Add to the project")}</TooltipContent>
            </Tooltip>
          ) : null}
        </div>
        <Input
          ref={filterRef}
          type="search"
          value={query}
          aria-label={t("Filter files")}
          placeholder={t("Filter files")}
          leftElement={<SearchSmIcon className="text-muted-foreground size-3.5" />}
          rightElement={query ? undefined : <Kbd className="mr-0.5">{formatShortcut("P")}</Kbd>}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape" && query) {
              event.preventDefault();
              setQuery("");
            }
          }}
        />
      </div>
      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-1 py-1.5">
          {nothing ? (
            <p className="text-muted-foreground px-3 py-2 text-xs">
              {t("No file or sample matches {0}", query.trim())}
            </p>
          ) : null}
          {partner.length > 0 ? (
            <ul>{partner.map((entry) => row(entry, `partner:${entry.path}`, 0))}</ul>
          ) : null}
          {flows.length > 0 ? (
            <TreeSection
              title={SECTION_LABELS.flows}
              count={flows.length}
              open={isOpen("flows")}
              onToggle={() => toggle("flows")}
              onAdd={editable ? () => openAdd("flow") : undefined}
              addLabel={t("Add flow")}
            >
              {flows.map((branch) => {
                const key = `flow:${branch.flow.path}`;
                const expanded = isOpen(key);
                return (
                  <li key={branch.flow.path}>
                    <ul>
                      {renaming === key ? (
                        row(branch.flow, key)
                      ) : (
                        <TreeRow
                          entry={branch.flow}
                          depth={1}
                          counts={counts.get(branch.flow.path)}
                          open={branch.flow.path === openPath}
                          editable={editable}
                          expanded={branch.children.length > 0 ? expanded : undefined}
                          onToggle={() => toggle(key)}
                          onOpen={onOpen}
                          onRename={() => setRenaming(key)}
                          onRemove={onRemove}
                        />
                      )}
                    </ul>
                    {expanded && branch.children.length > 0 ? (
                      <Guided depth={1}>
                        {branch.children.map((entry) => row(entry, `${key}>${entry.path}`, 2))}
                      </Guided>
                    ) : null}
                  </li>
                );
              })}
            </TreeSection>
          ) : null}
          {shared.length > 0 ? (
            <TreeSection
              title={SECTION_LABELS.shared}
              count={shared.length}
              open={isOpen("shared")}
              onToggle={() => toggle("shared")}
              onAdd={editable ? () => openAdd(null) : undefined}
              addLabel={t("Add a guide or library")}
            >
              {shared.map((entry) => row(entry, `shared:${entry.path}`))}
            </TreeSection>
          ) : null}
          {loose.length > 0 ? (
            <TreeSection
              title={SECTION_LABELS.loose}
              count={loose.length}
              hint={t("No flow's clause names these files")}
              open={isOpen("loose")}
              onToggle={() => toggle("loose")}
            >
              {loose.map((entry) => row(entry, `loose:${entry.path}`))}
            </TreeSection>
          ) : null}
          {!needle || shownSamples.length > 0 ? (
            <TreeSection
              title={SECTION_LABELS.samples}
              count={shownSamples.length}
              open={isOpen("samples")}
              onToggle={() => toggle("samples")}
              onAdd={editable ? () => openAdd("upload") : undefined}
              addLabel={t("Add samples")}
            >
              {shownSamples.length === 0 ? (
                <li className="flex flex-col items-start gap-2 py-1 pr-3" style={{ paddingLeft: glyphInset(1) }}>
                  <p className="text-muted-foreground text-xs">
                    {t("No samples yet. Add the partner's messages to run maps against them.")}
                  </p>
                  {editable ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="xs"
                      onClick={() => openAdd("upload")}
                    >
                      <Upload01Icon className="size-3" />
                      {t("Add samples")}
                    </Button>
                  ) : null}
                </li>
              ) : (
                shownSamples.map((entry) => row(entry, `sample:${entry.path}`))
              )}
            </TreeSection>
          ) : null}
        </div>
      </ScrollArea>
      {editable ? (
        <LadingAddEntryDialog
          open={adding.open}
          onOpenChange={(open) => setAdding((current) => ({ ...current, open }))}
          session={adding.session}
          choice={adding.choice}
          check={check}
          taken={taken}
          partnerLabel={partnerLabel}
          zone={zone}
          onAdd={onAdd}
        />
      ) : null}
    </nav>
  );
}

// The tree's geometry: a row at depth d starts its chevron slot one rem in per level past the
// section headings' half-rem; the slot is 0.875rem and the gap after it 0.375rem, so a row's
// glyph sits at its indent plus 1.25rem, and the guide under a parent runs down the middle of
// the parent's chevron slot.
const indent = (depth: number) => `${0.5 + depth}rem`;
const glyphInset = (depth: number) => `${1.75 + depth}rem`;
const guideInset = (depth: number) => `${0.9375 + depth}rem`;

/** Rows one level in, with a guide line down from their parent's chevron. */
function Guided({ depth, children }: { depth: number; children: ReactNode }) {
  return (
    <div className="relative">
      <span
        aria-hidden
        className="bg-border absolute inset-y-0 w-px"
        style={{ left: guideInset(depth) }}
      />
      <ul className="flex flex-col">{children}</ul>
    </div>
  );
}

/** A collapsible section of the tree with its count, a hint on hover and an add button. */
function TreeSection({
  title,
  count,
  hint,
  open,
  onToggle,
  onAdd,
  addLabel,
  children,
}: {
  title: string;
  count: number;
  hint?: string;
  open: boolean;
  onToggle: () => void;
  onAdd?: () => void;
  addLabel?: string;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col">
      <div className="group/section flex h-7 items-center gap-1 pr-1.5 pl-2">
        <button
          type="button"
          aria-expanded={open}
          title={hint}
          onClick={onToggle}
          className="ui-focus-ring text-muted-foreground hover:text-foreground flex min-w-0 flex-1 items-center gap-1.5 rounded-sm text-xs"
        >
          <ChevronRightIcon
            className={cn("size-3.5 shrink-0 transition-transform", open && "rotate-90")}
          />
          <span className="truncate font-medium">{title}</span>
          <span className="tabular-nums">{count}</span>
        </button>
        {onAdd ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={addLabel}
            className="opacity-0 group-hover/section:opacity-100 focus-visible:opacity-100 pointer-coarse:opacity-100"
            onClick={onAdd}
          >
            <PlusIcon className="size-3.5" />
          </Button>
        ) : null}
      </div>
      {open ? <Guided depth={0}>{children}</Guided> : null}
    </section>
  );
}

/** A file's errors, else its warnings, as a count with the words for a screen reader. */
function ProblemCount({ counts }: { counts: Counts }) {
  const t = useT();
  if (counts.errors === 0 && counts.warnings === 0) return null;
  const errors = counts.errors > 0;
  const count = errors ? counts.errors : counts.warnings;
  return (
    <span
      className={cn(
        "flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full px-1 text-2xs font-medium tabular-nums",
        errors
          ? "bg-danger-subtle text-danger-subtle-foreground"
          : "bg-warning-subtle text-warning-subtle-foreground",
      )}
    >
      <span aria-hidden="true">{count}</span>
      <span className="sr-only">
        {errors
          ? t("{0, plural, one {# error} other {# errors}}", count)
          : t("{0, plural, one {# warning} other {# warnings}}", count)}
      </span>
    </span>
  );
}

/** What the check knows of an entry, in a few words: `204 in`, `for tenderIn`, `guide`. */
function EntryDetail({ entry }: { entry: TreeEntry }) {
  const t = useT();
  let text: string | null = null;
  if (entry.map) {
    const read = entry.map.set ?? entry.map.source.toUpperCase();
    text = entry.map.direction === "inbound" ? t("{0} in", read) : t("{0} out", read);
  } else if (entry.rulesFor) {
    text = t("for {0}", entry.rulesFor);
  } else if (entry.kind === "guide") {
    text = t("guide");
  } else if (entry.kind === "library") {
    text = t("library");
  }
  if (!text) return null;
  return <span className="text-muted-foreground shrink-0 truncate text-2xs">{text}</span>;
}

function TreeRow({
  entry,
  depth,
  counts,
  open,
  editable,
  expanded,
  onToggle,
  onOpen,
  onRename,
  onRemove,
}: {
  entry: TreeEntry;
  /** How far in the row sits: 0 beside the section headings, one step in per level. */
  depth: number;
  counts?: Counts;
  open: boolean;
  editable: boolean;
  /** Whether the row's children show; undefined for a row without children. */
  expanded?: boolean;
  onToggle?: () => void;
  onOpen: (path: string) => void;
  onRename: () => void;
  onRemove: (path: string) => void;
}) {
  const t = useT();
  const { copy } = useCopyToClipboard();
  return (
    <li
      className={cn(
        "group/entry relative flex h-7 items-center gap-1.5 pr-1",
        open ? "bg-surface-active" : "hover:bg-surface-hover",
      )}
      style={{ paddingLeft: indent(depth) }}
    >
      {open ? (
        <span aria-hidden className="bg-foreground absolute inset-y-1 left-0 w-0.5 rounded-full" />
      ) : null}
      {expanded === undefined ? (
        <span aria-hidden className="w-3.5 shrink-0" />
      ) : (
        <button
          type="button"
          aria-expanded={expanded}
          aria-label={expanded ? t("Collapse {0}", entry.label) : t("Expand {0}", entry.label)}
          onClick={onToggle}
          className="ui-focus-ring text-muted-foreground hover:text-foreground flex h-full w-3.5 shrink-0 items-center rounded-sm"
        >
          <ChevronRightIcon
            className={cn("size-3.5 transition-transform", expanded && "rotate-90")}
          />
        </button>
      )}
      <button
        type="button"
        title={entry.path}
        aria-current={open ? "true" : undefined}
        onClick={() => onOpen(entry.path)}
        onKeyDown={(event) => {
          if (event.key === "F2" && editable) {
            event.preventDefault();
            onRename();
          }
        }}
        className="ui-focus-ring flex h-full min-w-0 flex-1 items-center gap-2 pr-1 text-left"
      >
        <KindGlyph kind={entry.format ?? (entry.kind === "sample" ? "x12" : entry.kind)} />
        <span
          className={cn(
            "min-w-0 flex-1 truncate font-mono text-xs",
            open ? "text-foreground" : "text-foreground-muted",
          )}
        >
          {entry.label}
        </span>
        <EntryDetail entry={entry} />
        {counts ? <ProblemCount counts={counts} /> : null}
      </button>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("Actions for {0}", entry.path)}
              className="opacity-0 group-hover/entry:opacity-100 focus-visible:opacity-100 data-[popup-open]:opacity-100 pointer-coarse:opacity-100"
            >
              <DotsHorizontalIcon className="size-3.5" />
            </Button>
          }
        />
        <DropdownMenuContent align="end" className="w-44">
          {editable ? (
            <DropdownMenuItem
              title={t("Rename")}
              startContent={<Edit03Icon className="size-3.5" />}
              endContent={<Kbd>F2</Kbd>}
              onClick={onRename}
            />
          ) : null}
          <DropdownMenuItem
            title={t("Copy path")}
            startContent={<Copy01Icon className="size-3.5" />}
            onClick={() => void copy(entry.path, { withToast: true })}
          />
          {editable ? (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                title={t("Remove")}
                startContent={<Trash01Icon className="size-3.5" />}
                color="danger"
                onClick={() => onRemove(entry.path)}
              />
            </>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    </li>
  );
}

/**
 * An entry's path in place, as it would be saved: Enter or leaving the field moves it when the
 * new path is free and of the entry's kind, Escape keeps it where it is.
 */
function RenameRow({
  entry,
  depth,
  taken,
  onDone,
}: {
  entry: TreeEntry;
  depth: number;
  taken: readonly string[];
  onDone: (to: string | null) => void;
}) {
  const t = useT();
  const problemText = usePathProblemText();
  const [value, setValue] = useState(entry.path);
  // Enter and Escape unmount the field, and a browser may blur it as it goes: finish once.
  const finished = useRef(false);
  const finish = (to: string | null) => {
    if (finished.current) return;
    finished.current = true;
    onDone(to);
  };
  const problem = newPathProblem(value, entry.kind === "sample" ? "sample" : "file", taken);
  const commit = () => finish(problem ? null : value.trim());

  return (
    <li className="flex flex-col gap-1 py-1 pr-2" style={{ paddingLeft: glyphInset(depth) }}>
      <Input
        value={value}
        autoFocus
        spellCheck={false}
        aria-label={t("New path for {0}", entry.path)}
        aria-invalid={problem && value !== entry.path ? true : undefined}
        className="font-mono text-xs"
        onFocus={(event: FocusEvent<HTMLInputElement>) => {
          const slash = entry.path.lastIndexOf("/") + 1;
          const dot = entry.path.indexOf(".", slash);
          event.currentTarget.setSelectionRange(slash, dot > slash ? dot : entry.path.length);
        }}
        onChange={(event) => setValue(event.target.value)}
        onBlur={commit}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            if (!problem) commit();
          } else if (event.key === "Escape") {
            event.preventDefault();
            finish(null);
          }
        }}
      />
      {problem && value !== entry.path ? (
        <p className="text-danger text-xs">{problemText[problem]}</p>
      ) : null}
    </li>
  );
}
