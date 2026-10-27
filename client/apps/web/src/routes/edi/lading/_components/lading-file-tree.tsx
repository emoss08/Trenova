import { useT } from "@trenova/shared/i18n/use-t";
import {
  FileCode01Icon,
  File06Icon,
  PlusIcon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@trenova/shared/components/ui/popover";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import type { LadingFile, LadingSample } from "@trenova/shared/edi/lading";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { newFileSource, newPathProblem, type NewPathProblem } from "./studio-model";

type EntryKind = "file" | "sample";

type Counts = ReadonlyMap<string, { errors: number; warnings: number }>;

/**
 * The project's source files and samples, each source file with the number of errors and
 * warnings the check found in it. A draft adds and removes entries here.
 */
export function LadingFileTree({
  files,
  samples,
  counts,
  openPath,
  editable,
  partnerName,
  onOpen,
  onAdd,
  onRemove,
}: {
  files: readonly LadingFile[];
  samples: readonly LadingSample[];
  counts: Counts;
  openPath: string;
  editable: boolean;
  partnerName: string;
  onOpen: (path: string) => void;
  onAdd: (kind: EntryKind, path: string, content: string) => void;
  onRemove: (path: string) => void;
}) {
  const t = useT();
  const taken = [...files.map((file) => file.path), ...samples.map((sample) => sample.path)];
  const sortedFiles = [...files].sort((a, b) => a.path.localeCompare(b.path));
  const sortedSamples = [...samples].sort((a, b) => a.path.localeCompare(b.path));

  return (
    <nav aria-label={t("Project files")} className="flex h-full min-h-0 flex-col">
      <div className="flex items-center justify-between border-b px-3 py-2">
        <span className="text-sm font-medium">{t("Files")}</span>
        {editable ? (
          <AddEntry
            taken={taken}
            onAdd={(kind, path) =>
              onAdd(kind, path, kind === "file" ? newFileSource(path, partnerName) : "")
            }
          />
        ) : null}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto py-1">
        <ul className="flex flex-col">
          {sortedFiles.map((file) => (
            <TreeEntry
              key={file.path}
              path={file.path}
              icon={<FileCode01Icon className="size-4 shrink-0" />}
              counts={counts.get(file.path)}
              open={file.path === openPath}
              editable={editable}
              onOpen={onOpen}
              onRemove={onRemove}
            />
          ))}
        </ul>
        <p className="text-muted-foreground mt-3 px-3 pb-1 text-xs">{t("Samples")}</p>
        {sortedSamples.length === 0 ? (
          <p className="text-muted-foreground px-3 text-xs">
            {t("No samples yet. Add one to run a map against it.")}
          </p>
        ) : (
          <ul className="flex flex-col">
            {sortedSamples.map((sample) => (
              <TreeEntry
                key={sample.path}
                path={sample.path}
                icon={<File06Icon className="size-4 shrink-0" />}
                open={sample.path === openPath}
                editable={editable}
                onOpen={onOpen}
                onRemove={onRemove}
              />
            ))}
          </ul>
        )}
      </div>
    </nav>
  );
}

function TreeEntry({
  path,
  icon,
  counts,
  open,
  editable,
  onOpen,
  onRemove,
}: {
  path: string;
  icon: ReactNode;
  counts?: { errors: number; warnings: number };
  open: boolean;
  editable: boolean;
  onOpen: (path: string) => void;
  onRemove: (path: string) => void;
}) {
  const t = useT();
  return (
    <li className="group/entry relative">
      <button
        type="button"
        aria-current={open ? "true" : undefined}
        onClick={() => onOpen(path)}
        className={cn(
          "ui-focus-ring flex w-full items-center gap-2 px-3 py-1 text-left text-sm",
          open ? "bg-surface-active" : "hover:bg-surface-hover",
        )}
      >
        {icon}
        <span className="min-w-0 flex-1 truncate font-mono text-xs">{path}</span>
        {counts && counts.errors > 0 ? (
          <span className="text-danger text-xs tabular-nums">
            {t("{0, plural, one {# error} other {# errors}}", counts.errors)}
          </span>
        ) : counts && counts.warnings > 0 ? (
          <span className="text-warning text-xs tabular-nums">
            {t("{0, plural, one {# warning} other {# warnings}}", counts.warnings)}
          </span>
        ) : null}
      </button>
      {editable ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="absolute top-0.5 right-1 opacity-0 group-hover/entry:opacity-100 focus-visible:opacity-100"
          aria-label={t("Remove {0}", path)}
          onClick={() => onRemove(path)}
        >
          <Trash01Icon className="size-3.5" />
        </Button>
      ) : null}
    </li>
  );
}

function AddEntry({
  taken,
  onAdd,
}: {
  taken: readonly string[];
  onAdd: (kind: EntryKind, path: string) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<EntryKind>("file");
  const [path, setPath] = useState("maps/");
  const [touched, setTouched] = useState(false);
  const problem = newPathProblem(path, kind, taken);

  const problemText: Record<NewPathProblem, string> = {
    empty: t("Name the file"),
    notRelative: t("Use a path inside the project, such as maps/status.lading"),
    notLading: t("A source file's name ends in .lading"),
    taken: t("The project already has a file at this path"),
  };

  const submit = () => {
    setTouched(true);
    if (problem) return;
    onAdd(kind, path.trim());
    setOpen(false);
    setTouched(false);
    setPath(kind === "file" ? "maps/" : "samples/");
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button type="button" variant="ghost" size="icon-xs" aria-label={t("Add a file")}>
            <PlusIcon className="size-3.5" />
          </Button>
        }
      />
      <PopoverContent className="w-80">
        <PopoverHeader>
          <PopoverTitle>{t("Add to the project")}</PopoverTitle>
          <PopoverDescription>
            {t("Maps, guides, rules and flows go in their folders; samples are messages to run.")}
          </PopoverDescription>
        </PopoverHeader>
        <form
          className="flex flex-col gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            submit();
          }}
        >
          <SegmentedControl<EntryKind>
            value={kind}
            aria-label={t("What to add")}
            fullWidth
            onValueChange={(next) => {
              setKind(next);
              setPath(next === "file" ? "maps/" : "samples/");
              setTouched(false);
            }}
            items={[
              { value: "file", label: t("Source file") },
              { value: "sample", label: t("Sample") },
            ]}
          />
          <div className="flex flex-col gap-1">
            <Input
              aria-label={t("Path")}
              value={path}
              autoFocus
              aria-invalid={touched && problem !== null ? true : undefined}
              onChange={(event) => setPath(event.target.value)}
            />
            {touched && problem ? (
              <p className="text-danger text-xs">{problemText[problem]}</p>
            ) : null}
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" size="sm" onClick={() => setOpen(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" size="sm">
              {t("Add")}
            </Button>
          </div>
        </form>
      </PopoverContent>
    </Popover>
  );
}
