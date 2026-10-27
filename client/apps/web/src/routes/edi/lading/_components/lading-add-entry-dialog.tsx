import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import { LadingCode } from "@/components/lading-editor/lading-code";
import { unlessRefused } from "@/components/lading-editor/lading-service";
import {
  ArrowLeftIcon,
  SearchSmIcon,
  Upload01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Label } from "@trenova/shared/components/ui/label";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  ladingCatalog,
  ladingDocText,
  ladingStarter,
  type LadingCheck,
} from "@trenova/shared/edi/lading";
import type { EdiRuntime } from "@trenova/shared/edi/runtime";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import {
  ADD_LABELS,
  CHOICE_DESCRIPTIONS,
  CHOICE_KEYS,
  CHOICE_LABELS,
  FILE_KINDS,
  SAMPLE_FORMATS,
  kindLook,
  usePathProblemText,
  type EntryChoice,
} from "./lading-file-kinds";
import { KindGlyph } from "./lading-kind-glyph";
import {
  documentStem,
  entryPath,
  freeName,
  kebabName,
  ladingName,
  newPathProblem,
  PARTNER_FILE,
  sampleFormat,
  starterSource,
  type FileStarter,
  type NewPathProblem,
  type SampleFormat,
} from "./studio-model";

/** An entry the dialog adds to the working copy. */
export type NewEntry = { kind: "file" | "sample"; path: string; content: string };

type MapDirection = "inbound" | "outbound";
type MapInput = "x12" | "csv" | "json";
type Upload = { name: string; content: string };

/** The built-in guide a partner's guide or map narrows for a set; one per document set. */
function baseGuide(set: string): string {
  return `x12.004010.${set}`;
}

function isSample(choice: EntryChoice): choice is SampleFormat {
  return (SAMPLE_FORMATS as readonly string[]).includes(choice);
}

/**
 * What to add to the project, in two steps. First a palette of every kind of source file Lading
 * has and every kind of sample: type to filter, the arrows and Enter to choose, or a kind's
 * letter while the filter is empty. Then that kind's form beside the starter it will open as;
 * samples are pasted or uploaded from the partner's own files. Opened on a kind, it starts at
 * the form.
 */
export function LadingAddEntryDialog({
  open,
  onOpenChange,
  session,
  choice,
  check,
  taken,
  partnerLabel,
  zone,
  onAdd,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Changes on every opening, so each opening starts from a clean palette and form. */
  session: number;
  /** The kind to open the form on, or null for the palette. */
  choice: EntryChoice | null;
  check: LadingCheck;
  taken: readonly string[];
  partnerLabel: string;
  zone: string;
  onAdd: (entries: NewEntry[]) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <AddEntrySteps
        key={session}
        initial={choice}
        check={check}
        taken={taken}
        partnerLabel={partnerLabel}
        zone={zone}
        onCancel={() => onOpenChange(false)}
        onAdd={(entries) => {
          onAdd(entries);
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}

function AddEntrySteps({
  initial,
  check,
  taken,
  partnerLabel,
  zone,
  onCancel,
  onAdd,
}: {
  initial: EntryChoice | null;
  check: LadingCheck;
  taken: readonly string[];
  partnerLabel: string;
  zone: string;
  onCancel: () => void;
  onAdd: (entries: NewEntry[]) => void;
}) {
  const t = useT();
  const unavailable: Partial<Record<EntryChoice, string>> = {};
  if (check.partner !== null || taken.includes(PARTNER_FILE)) {
    unavailable.partner = t("The project has its partner file");
  }
  if (check.maps.length === 0) unavailable.rules = t("Add a map first");
  const [picked, setPicked] = useState<EntryChoice | null>(
    initial !== null && unavailable[initial] === undefined ? initial : null,
  );

  if (picked === null) {
    return (
      <DialogContent size="lg" showCloseButton={false} className="gap-0 overflow-hidden p-0">
        <DialogTitle className="sr-only">{t("Add to the project")}</DialogTitle>
        <Palette unavailable={unavailable} onPick={setPicked} />
      </DialogContent>
    );
  }
  return (
    <DialogContent size="xl" className="gap-0 overflow-hidden p-0">
      <AddEntryForm
        choice={picked}
        check={check}
        taken={taken}
        partnerLabel={partnerLabel}
        zone={zone}
        onBack={() => setPicked(null)}
        onCancel={onCancel}
        onAdd={onAdd}
      />
    </DialogContent>
  );
}

function AddEntryForm({
  choice: initial,
  check,
  taken,
  partnerLabel,
  zone,
  onBack,
  onCancel,
  onAdd,
}: {
  choice: EntryChoice;
  check: LadingCheck;
  taken: readonly string[];
  partnerLabel: string;
  zone: string;
  onBack: () => void;
  onCancel: () => void;
  onAdd: (entries: NewEntry[]) => void;
}) {
  const t = useT();
  const edi = useEdi();
  const nameId = useId();
  const documents = ladingCatalog(edi).documents.flatMap((record) =>
    record.set === null ? [] : [{ name: record.name, set: record.set, doc: record.doc }],
  );
  const hasPartner = check.partner !== null || taken.includes(PARTNER_FILE);
  const partner = check.partner?.name ?? ladingName(partnerLabel);
  const inboundMaps = check.maps.filter((map) => map.direction === "inbound");

  const problemText = usePathProblemText();
  const [choice, setChoice] = useState<EntryChoice>(initial);
  const [typed, setTyped] = useState<Partial<Record<EntryChoice, string>>>({});
  const [submitted, setSubmitted] = useState(false);
  const [direction, setDirection] = useState<MapDirection>("inbound");
  const [input, setInput] = useState<MapInput>("x12");
  const [docName, setDocName] = useState(documents[0]?.name ?? "loadTender");
  const [mapGuide, setMapGuide] = useState<string | null>(null);
  const [rulesMap, setRulesMap] = useState(inboundMaps[0]?.name ?? check.maps[0]?.name ?? "");
  const [guideSet, setGuideSet] = useState(documents[0]?.set ?? "204");
  const [guideBase, setGuideBase] = useState<string | null>(null);
  const [guideTitle, setGuideTitle] = useState("");
  const [content, setContent] = useState<Partial<Record<SampleFormat, string>>>({});
  const [uploads, setUploads] = useState<Upload[]>([]);

  const guidesFor = (set: string) => [
    ...check.guides.filter((item) => item.path !== null && item.set === set).map((g) => g.name),
    baseGuide(set),
  ];
  const picked = documents.find((item) => item.name === docName);
  const set = picked?.set ?? "204";
  const guides = guidesFor(set);
  const guide =
    mapGuide !== null && guides.includes(mapGuide) ? mapGuide : (guides[0] ?? baseGuide(set));
  const guideBases = guidesFor(guideSet);
  const base =
    guideBase !== null && guideBases.includes(guideBase)
      ? guideBase
      : (guideBases[0] ?? baseGuide(guideSet));

  const suggested = (kind: Exclude<EntryChoice, "upload">): string => {
    const stem = documentStem(docName);
    const name =
      kind === "map"
        ? direction === "outbound"
          ? `${stem}-out`
          : input === "x12"
            ? `${stem}-in`
            : `${stem}-${input}`
        : kind === "rules"
          ? rulesMap
            ? `${kebabName(rulesMap).replace(/-(in|out)$/, "")}-checks`
            : ""
          : kind === "guide"
            ? `${kebabName(partner)}-${guideSet}`
            : kind === "library"
              ? "common"
              : "";
    return name === "" || kind === "partner" ? name : freeName(kind, name, taken);
  };

  const name = choice === "upload" ? "" : (typed[choice] ?? suggested(choice));
  const path = choice === "upload" ? "" : entryPath(choice, name);
  const problem: NewPathProblem | null =
    choice === "upload" || choice === "partner"
      ? null
      : newPathProblem(path, isSample(choice) ? "sample" : "file", taken);
  const blocked =
    (choice === "rules" && check.maps.length === 0) || (choice === "partner" && hasPartner);

  const starter = (): FileStarter | null => {
    switch (choice) {
      case "map":
        return direction === "outbound"
          ? { kind: "map", direction, document: docName, guide }
          : input === "x12"
            ? { kind: "map", direction, input, document: docName, guide }
            : { kind: "map", direction, input, document: docName };
      case "rules":
        return { kind: "rules", map: rulesMap };
      case "flow":
        return { kind: "flow", partner };
      case "guide":
        return { kind: "guide", base, title: guideTitle };
      case "library":
        return { kind: "library" };
      default:
        return null;
    }
  };
  const fileStarter = starter();
  const source =
    choice === "partner"
      ? partnerSource(edi, partnerLabel, zone)
      : fileStarter && path !== "" && !blocked
        ? starterSource(path, fileStarter)
        : "";

  const uploadEntries = uploads.map((upload, index) => {
    const uploadPath = `samples/${upload.name}`;
    const earlier = uploads.slice(0, index).map((item) => `samples/${item.name}`);
    return {
      ...upload,
      path: uploadPath,
      problem: newPathProblem(uploadPath, "sample", [...taken, ...earlier]),
    };
  });

  const submit = () => {
    setSubmitted(true);
    if (choice === "upload") {
      const ready = uploadEntries.filter((upload) => upload.problem === null);
      if (ready.length === 0 || ready.length !== uploadEntries.length) return;
      onAdd(
        ready.map((upload) => ({ kind: "sample", path: upload.path, content: upload.content })),
      );
      return;
    }
    if (blocked || problem) return;
    if (isSample(choice)) {
      onAdd([{ kind: "sample", path: path.trim(), content: content[choice] ?? "" }]);
      return;
    }
    onAdd([
      {
        kind: "file",
        path: choice === "partner" ? PARTNER_FILE : path.trim(),
        content: source,
      },
    ]);
  };

  const readUploads = async (list: FileList | null) => {
    const read = await Promise.all(
      [...(list ?? [])].map(async (file) => ({ name: file.name, content: await file.text() })),
    );
    setUploads((current) => [
      ...current,
      ...read.filter((file) => !current.some((item) => item.name === file.name)),
    ]);
  };

  const hasPreview = source !== "";

  return (
    <form
      className="flex max-h-[calc(100dvh-8rem)] min-h-0 flex-col sm:h-[34rem]"
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
      onKeyDown={(event) => {
        if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
          event.preventDefault();
          submit();
        }
      }}
    >
      <div className="flex items-center gap-3 border-b py-2.5 pr-12 pl-2.5">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t("Back to every kind")}
          onClick={onBack}
        >
          <ArrowLeftIcon className="size-4" />
        </Button>
        <KindGlyph kind={choice} size="md" />
        <div className="flex min-w-0 flex-col">
          <DialogTitle className="text-sm font-semibold">{CHOICE_LABELS[choice]}</DialogTitle>
          <p className="text-muted-foreground truncate text-xs">{CHOICE_DESCRIPTIONS[choice]}</p>
        </div>
      </div>
      <div
        className={cn(
          "grid min-h-0 flex-1",
          hasPreview && "grid-rows-[minmax(0,1fr)_minmax(8rem,auto)] sm:grid-cols-2 sm:grid-rows-1",
        )}
      >
        <ScrollArea className="min-h-0">
          <div className="flex flex-col gap-4 p-4">
            {choice === "map" ? (
              <>
                <Field label={t("Direction")}>
                  <SegmentedControl<MapDirection>
                    value={direction}
                    aria-label={t("Direction")}
                    fullWidth
                    onValueChange={setDirection}
                    items={[
                      { value: "inbound", label: t("Partner to Trenova") },
                      { value: "outbound", label: t("Trenova to partner") },
                    ]}
                  />
                </Field>
                {direction === "inbound" ? (
                  <Field label={t("Reads")}>
                    <SegmentedControl<MapInput>
                      value={input}
                      aria-label={t("Reads")}
                      fullWidth
                      onValueChange={setInput}
                      items={[
                        { value: "x12", label: t("X12") },
                        { value: "csv", label: t("CSV") },
                        { value: "json", label: t("JSON") },
                      ]}
                    />
                  </Field>
                ) : null}
                <div className="grid gap-4 sm:grid-cols-2">
                  <Field label={direction === "inbound" ? t("Makes") : t("Reads")}>
                    <Select
                      value={docName}
                      items={documents.map((item) => ({ value: item.name, label: item.name }))}
                      onValueChange={(next) => {
                        if (next) setDocName(next);
                      }}
                    >
                      <SelectTrigger className="w-full" aria-label={t("Document")}>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {documents.map((item) => (
                          <SelectItem key={item.name} value={item.name}>
                            <span className="font-mono text-xs">{item.name}</span>
                            <span className="text-muted-foreground ml-auto pl-3 text-xs tabular-nums">
                              {item.set}
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </Field>
                  {direction === "outbound" || input === "x12" ? (
                    <Field label={direction === "inbound" ? t("From the guide") : t("Writes with")}>
                      <Select
                        value={guide}
                        items={guides.map((item) => ({ value: item, label: item }))}
                        onValueChange={(next) => setMapGuide(next)}
                      >
                        <SelectTrigger className="w-full" aria-label={t("Guide")}>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {guides.map((item) => (
                            <SelectItem key={item} value={item}>
                              <span className="font-mono text-xs">{item}</span>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </Field>
                  ) : null}
                </div>
                <p className="text-muted-foreground text-xs">
                  {ladingDocText(picked?.doc, (message) => translateMessage(message))}
                </p>
              </>
            ) : null}

            {choice === "rules" ? (
              check.maps.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                  {t("Rules check the document a map makes. Add a map first.")}
                </p>
              ) : (
                <Field label={t("Checks the document of")}>
                  <Select
                    value={rulesMap}
                    items={check.maps.map((map) => ({ value: map.name, label: map.name }))}
                    onValueChange={(next) => {
                      if (next) setRulesMap(next);
                    }}
                  >
                    <SelectTrigger className="w-full" aria-label={t("Map")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {check.maps.map((map) => (
                        <SelectItem key={map.name} value={map.name}>
                          <span className="font-mono text-xs">{map.name}</span>
                          <span className="text-muted-foreground ml-auto pl-3 text-xs">
                            {map.document}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              )
            ) : null}

            {choice === "guide" ? (
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label={t("Set")}>
                  <Select
                    value={guideSet}
                    items={documents.map((item) => ({ value: item.set, label: item.set }))}
                    onValueChange={(next) => {
                      if (next) setGuideSet(next);
                    }}
                  >
                    <SelectTrigger className="w-full" aria-label={t("Set")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {documents.map((item) => (
                        <SelectItem key={item.set} value={item.set}>
                          <span className="tabular-nums">{item.set}</span>
                          <span className="text-muted-foreground ml-auto pl-3 font-mono text-xs">
                            {item.name}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
                <Field label={t("Narrows")}>
                  <Select
                    value={base}
                    items={guideBases.map((item) => ({ value: item, label: item }))}
                    onValueChange={(next) => setGuideBase(next)}
                  >
                    <SelectTrigger className="w-full" aria-label={t("Narrows")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {guideBases.map((item) => (
                        <SelectItem key={item} value={item}>
                          <span className="font-mono text-xs">{item}</span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
                <Field label={t("Title")} className="sm:col-span-2">
                  <Input
                    value={guideTitle}
                    placeholder={t(
                      "As the partner's guide is titled, such as ACME Load Tender Guide rev 3.2",
                    )}
                    onChange={(event) => setGuideTitle(event.target.value)}
                  />
                </Field>
              </div>
            ) : null}

            {choice === "upload" ? (
              <UploadZone
                uploads={uploadEntries}
                problemText={problemText}
                onPick={(list) => void readUploads(list)}
                onRemove={(remove) =>
                  setUploads((current) => current.filter((item) => item.name !== remove))
                }
              />
            ) : choice === "partner" ? (
              <p className="text-muted-foreground font-mono text-xs">
                {t("Saved as {0}", PARTNER_FILE)}
              </p>
            ) : blocked ? null : (
              <Field
                label={t("Name")}
                htmlFor={nameId}
                hint={path ? t("Saved as {0}", path) : undefined}
                error={
                  (submitted || typed[choice] !== undefined) && problem
                    ? problemText[problem]
                    : undefined
                }
              >
                <Input
                  id={nameId}
                  value={name}
                  autoFocus
                  spellCheck={false}
                  className="font-mono"
                  placeholder={
                    isSample(choice) ? t("such as tender-original") : t("such as tender")
                  }
                  aria-invalid={submitted && problem !== null ? true : undefined}
                  onChange={(event) =>
                    setTyped((current) => ({ ...current, [choice]: event.target.value }))
                  }
                />
              </Field>
            )}

            {isSample(choice) ? (
              <SampleContent
                format={choice}
                value={content[choice] ?? ""}
                onChange={(next) => setContent((current) => ({ ...current, [choice]: next }))}
                onFile={(file, text) => {
                  const format = sampleFormat(file) ?? choice;
                  const stem =
                    format === "document"
                      ? file.replace(/\.document\.json$/i, "")
                      : file.replace(/\.[^.]+$/, "");
                  setChoice(format);
                  setContent((current) => ({ ...current, [format]: text }));
                  setTyped((current) => ({ ...current, [format]: current[format] ?? stem }));
                }}
              />
            ) : null}
          </div>
        </ScrollArea>
        {hasPreview ? (
          <div className="bg-muted flex min-h-0 flex-col border-t sm:border-t-0 sm:border-l">
            <div className="text-muted-foreground flex items-center gap-2 px-4 pt-3 pb-1 text-xs">
              <span className="flex-1">{t("Starts as")}</span>
              <span className="truncate font-mono">
                {choice === "partner" ? PARTNER_FILE : path}
              </span>
            </div>
            <ScrollArea maskVariant="muted" className="min-h-0 flex-1">
              <LadingCode code={source.trimEnd()} className="w-max min-w-full px-4 py-2" />
            </ScrollArea>
          </div>
        ) : null}
      </div>
      <div className="flex items-center gap-2 border-t px-4 py-2.5">
        <span className="text-muted-foreground hidden flex-1 items-center gap-1.5 text-xs sm:flex">
          <Kbd>{formatShortcut("↵")}</Kbd>
          {t("to add")}
        </span>
        <Button type="button" variant="outline" size="sm" className="ml-auto" onClick={onCancel}>
          {t("Cancel")}
        </Button>
        <Button
          type="submit"
          size="sm"
          disabled={blocked || (choice === "upload" && uploads.length === 0)}
        >
          {choice === "upload"
            ? t("{0, plural, one {Add # sample} other {Add # samples}}", uploads.length)
            : ADD_LABELS[choice]}
        </Button>
      </div>
    </form>
  );
}

/** The partner file the module's starter writes for a partner of this name. */
function partnerSource(edi: EdiRuntime, label: string, zone: string): string {
  const starter = unlessRefused(
    () => ladingStarter(edi, { name: label, label, role: "carrier", zone }),
    { files: [], samples: [] },
  );
  return starter.files.find((file) => file.path === PARTNER_FILE)?.source ?? "";
}

/** Where a palette row sits: in the source files or among the samples. */
const PALETTE_GROUPS: { key: "source" | "samples"; choices: readonly EntryChoice[] }[] = [
  { key: "source", choices: FILE_KINDS },
  { key: "samples", choices: [...SAMPLE_FORMATS, "upload"] },
];

/**
 * Every kind there is to add, as a filterable list: the arrows move, Enter chooses, and while
 * the filter is empty a kind's letter chooses it. A kind that cannot be added says why.
 */
function Palette({
  unavailable,
  onPick,
}: {
  unavailable: Partial<Record<EntryChoice, string>>;
  onPick: (choice: EntryChoice) => void;
}) {
  const t = useT();
  const listId = useId();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const needle = query.trim().toLowerCase();
  const groups = PALETTE_GROUPS.map((group) => ({
    ...group,
    choices: group.choices.filter(
      (choice) =>
        needle === "" ||
        CHOICE_LABELS[choice].toLowerCase().includes(needle) ||
        CHOICE_DESCRIPTIONS[choice].toLowerCase().includes(needle),
    ),
  })).filter((group) => group.choices.length > 0);
  const order = groups.flatMap((group) => group.choices);
  const pickable = order.filter((choice) => unavailable[choice] === undefined);
  const current = pickable[Math.min(active, pickable.length - 1)];
  const optionId = (choice: EntryChoice) => `${listId}-${choice}`;

  const choose = (choice: EntryChoice | undefined) => {
    if (choice && unavailable[choice] === undefined) onPick(choice);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (pickable.length === 0) return;
      const step = event.key === "ArrowDown" ? 1 : -1;
      const at = current ? pickable.indexOf(current) : 0;
      setActive((at + step + pickable.length) % pickable.length);
      return;
    }
    if (event.key === "Enter") {
      event.preventDefault();
      choose(current);
      return;
    }
    if (
      query === "" &&
      event.key.length === 1 &&
      !event.metaKey &&
      !event.ctrlKey &&
      !event.altKey
    ) {
      const byKey = order.find((choice) => CHOICE_KEYS[choice] === event.key.toLowerCase());
      if (byKey && unavailable[byKey] === undefined) {
        event.preventDefault();
        onPick(byKey);
      }
    }
  };

  return (
    <div className="flex max-h-[calc(100dvh-8rem)] flex-col">
      <div className="flex items-center gap-2 border-b px-4">
        <SearchSmIcon className="text-muted-foreground size-4 shrink-0" />
        <input
          autoFocus
          role="combobox"
          aria-expanded
          aria-controls={listId}
          aria-activedescendant={current ? optionId(current) : undefined}
          aria-label={t("What to add")}
          placeholder={t("What would you like to add?")}
          value={query}
          onChange={(event) => {
            setQuery(event.target.value);
            setActive(0);
          }}
          onKeyDown={onKeyDown}
          className="placeholder:text-muted-foreground h-12 min-w-0 flex-1 bg-transparent text-sm outline-none"
        />
        <Kbd>{t("Esc")}</Kbd>
      </div>
      <ScrollArea className="min-h-0 flex-1">
        <div id={listId} role="listbox" aria-label={t("Kinds to add")} className="p-2">
          {groups.length === 0 ? (
            <p className="text-muted-foreground px-2 py-6 text-center text-sm">
              {t("Nothing to add matches {0}", query.trim())}
            </p>
          ) : null}
          {groups.map((group) => (
            <div
              key={group.key}
              role="group"
              aria-label={group.key === "source" ? t("Source files") : t("Samples")}
            >
              <p className="text-muted-foreground px-2 pt-2 pb-1 text-xs font-medium">
                {group.key === "source" ? t("Source files") : t("Samples")}
              </p>
              {group.choices.map((choice) => {
                const reason = unavailable[choice];
                const selected = choice === current;
                return (
                  <div
                    key={choice}
                    id={optionId(choice)}
                    role="option"
                    aria-selected={selected}
                    aria-disabled={reason !== undefined}
                    onMouseMove={() => {
                      if (reason === undefined) setActive(pickable.indexOf(choice));
                    }}
                    onClick={() => choose(choice)}
                    className={cn(
                      "flex cursor-default items-center gap-3 rounded-md px-2 py-1.5",
                      selected && "bg-surface-active",
                      reason !== undefined && "opacity-50",
                    )}
                  >
                    <KindGlyph kind={choice} size="md" />
                    <span className="flex min-w-0 flex-1 flex-col">
                      <span className="text-sm">{CHOICE_LABELS[choice]}</span>
                      <span className="text-muted-foreground truncate text-xs">
                        {reason ?? CHOICE_DESCRIPTIONS[choice]}
                      </span>
                    </span>
                    <Kbd>{CHOICE_KEYS[choice].toUpperCase()}</Kbd>
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      </ScrollArea>
      <div className="text-muted-foreground flex items-center gap-4 border-t px-4 py-2 text-xs">
        <span className="flex items-center gap-1.5">
          <Kbd>↑</Kbd>
          <Kbd>↓</Kbd>
          {t("to move")}
        </span>
        <span className="flex items-center gap-1.5">
          <Kbd>↵</Kbd>
          {t("to choose")}
        </span>
        <span className="hidden items-center gap-1.5 sm:flex">{t("Or press a kind's letter")}</span>
      </div>
    </div>
  );
}

function Field({
  label,
  htmlFor,
  hint,
  error,
  className,
  children,
}: {
  label: string;
  htmlFor?: string;
  hint?: string;
  error?: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <Label htmlFor={htmlFor}>{label}</Label>
      {children}
      {error ? (
        <p className="text-danger text-xs">{error}</p>
      ) : hint ? (
        <p className="text-muted-foreground truncate font-mono text-xs">{hint}</p>
      ) : null}
    </div>
  );
}

/** A sample's text: pasted, or read from a file whose name also names the sample. */
function SampleContent({
  format,
  value,
  onChange,
  onFile,
}: {
  format: SampleFormat;
  value: string;
  onChange: (value: string) => void;
  onFile: (name: string, text: string) => void;
}) {
  const t = useT();
  const id = useId();
  const picker = useRef<HTMLInputElement>(null);
  const placeholder: Record<SampleFormat, string> = {
    x12: t("Paste the partner's message, from ISA to IEA"),
    csv: t("Paste the file, its header row first"),
    json: t("Paste the JSON document"),
    document: t("Paste the document as JSON"),
  };
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between gap-2">
        <Label htmlFor={id}>{t("Content")}</Label>
        <Button type="button" variant="ghost" size="xs" onClick={() => picker.current?.click()}>
          <Upload01Icon className="size-3.5" />
          {t("Read from a file")}
        </Button>
        <input
          ref={picker}
          type="file"
          hidden
          accept=".x12,.edi,.txt,.csv,.json"
          onChange={async (event) => {
            const file = event.target.files?.[0];
            event.target.value = "";
            if (file) onFile(file.name, await file.text());
          }}
        />
      </div>
      <Textarea
        id={id}
        value={value}
        rows={6}
        spellCheck={false}
        className="font-mono text-xs"
        placeholder={placeholder[format]}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  );
}

/** Files dropped or picked, each a sample at `samples/<its name>`, with why any cannot be. */
function UploadZone({
  uploads,
  problemText,
  onPick,
  onRemove,
}: {
  uploads: readonly (Upload & { path: string; problem: NewPathProblem | null })[];
  problemText: Record<NewPathProblem, string>;
  onPick: (files: FileList | null) => void;
  onRemove: (name: string) => void;
}) {
  const t = useT();
  const picker = useRef<HTMLInputElement>(null);
  const [over, setOver] = useState(false);
  return (
    <div className="flex flex-col gap-3">
      <button
        type="button"
        onClick={() => picker.current?.click()}
        onDragOver={(event) => {
          event.preventDefault();
          setOver(true);
        }}
        onDragLeave={() => setOver(false)}
        onDrop={(event) => {
          event.preventDefault();
          setOver(false);
          onPick(event.dataTransfer.files);
        }}
        className={cn(
          "ui-focus-ring flex flex-col items-center gap-1.5 rounded-lg border border-dashed px-4 py-8 text-center",
          over ? "border-primary bg-surface-active" : "hover:bg-surface-hover",
        )}
      >
        <Upload01Icon className="text-muted-foreground size-5" />
        <span className="text-sm font-medium">{t("Drop files here, or choose them")}</span>
        <span className="text-muted-foreground text-xs">
          {t(".x12, .edi and .txt are read as X12; .csv and .json as named")}
        </span>
      </button>
      <input
        ref={picker}
        type="file"
        hidden
        multiple
        accept=".x12,.edi,.txt,.csv,.json"
        onChange={(event) => {
          onPick(event.target.files);
          event.target.value = "";
        }}
      />
      {uploads.length > 0 ? (
        <ul className="flex flex-col divide-y rounded-md border">
          {uploads.map((upload) => {
            const look = kindLook(sampleFormat(upload.path) ?? "x12");
            const Icon = look.icon;
            return (
              <li key={upload.name} className="flex items-center gap-2 px-2 py-1.5">
                <Icon className={cn("size-4 shrink-0", look.ink)} />
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate font-mono text-xs">{upload.path}</span>
                  {upload.problem ? (
                    <span className="text-danger text-xs">{problemText[upload.problem]}</span>
                  ) : null}
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-label={t("Remove {0}", upload.name)}
                  onClick={() => onRemove(upload.name)}
                >
                  <XCloseIcon className="size-3.5" />
                </Button>
              </li>
            );
          })}
        </ul>
      ) : null}
    </div>
  );
}
