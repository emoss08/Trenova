import { CopyIconButton } from "@/components/copy-icon-button";
import { JsonViewer, type JsonValue } from "@/components/elements/json-viewer";
import { darkTheme, lightTheme } from "@/components/formula-editor/editor-theme";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@trenova/shared/components/ui/collapsible";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { ShikiCodeBlock } from "@trenova/shared/components/ui/shiki-code-block";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { GraphQLRequestError, requestGraphQL } from "@trenova/shared/lib/graphql";
import { formatTimeAgo } from "@/lib/time-utils";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import type { CatalogOperation } from "@/types/graphql-catalog";
import { json } from "@codemirror/lang-json";
import { linter, type Diagnostic } from "@codemirror/lint";
import { Prec } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import {
  AlertTriangleIcon,
  BracesIcon,
  ChevronRightIcon,
  HistoryIcon,
  PlayIcon,
  RotateCcwIcon,
  Trash2Icon,
} from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { catalog, referencedTypeNames, scaffoldVariables } from "./catalog";
import { useRunHistory, type RunHistoryEntry } from "./use-run-history";

const LARGE_RESPONSE_THRESHOLD = 150_000;

type ResponseView = "tree" | "raw";

type RunState =
  | { status: "idle" }
  | { status: "running" }
  | { status: "success"; data: unknown; raw: string; bytes: number; elapsedMs: number }
  | {
      status: "error";
      message: string;
      details?: string;
      httpStatus?: number;
      elapsedMs?: number;
    };

function formatJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function formatElapsed(elapsedMs: number): string {
  if (elapsedMs >= 1000) {
    return `${(elapsedMs / 1000).toFixed(2)} s`;
  }
  return `${Math.round(elapsedMs)} ms`;
}

function ResponseJson({ view, data, raw }: { view: ResponseView; data: unknown; raw: string }) {
  if (view === "tree") {
    return (
      <JsonViewer data={data as JsonValue} collapsed={3} copyPath={false} className="p-2 text-xs" />
    );
  }
  if (raw.length > LARGE_RESPONSE_THRESHOLD) {
    return <pre className="p-2 font-mono text-xs leading-relaxed">{raw}</pre>;
  }
  return <ShikiCodeBlock code={raw} lang="json" darkTheme="vitesse-dark" />;
}

function historyPreview(variables: string): string {
  return variables.replace(/\s+/g, " ").trim();
}

function HistoryPopover({
  entries,
  onReplay,
  onClear,
}: {
  entries: RunHistoryEntry[];
  onReplay: (entry: RunHistoryEntry) => void;
  onClear: () => void;
}) {
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={<Button size="sm" variant="outline" className="gap-1.5 text-muted-foreground" />}
      >
        <HistoryIcon className="size-3.5" />
        History
        {entries.length > 0 && (
          <span className="text-2xs text-muted-foreground/70 tabular-nums">{entries.length}</span>
        )}
      </PopoverTrigger>
      <PopoverContent align="start" className="w-88 p-0">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <span className="text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
            Run history
          </span>
          {entries.length > 0 && (
            <button
              type="button"
              onClick={onClear}
              className="flex items-center gap-1 text-2xs font-medium text-muted-foreground transition-colors hover:text-destructive"
            >
              <Trash2Icon className="size-3" />
              Clear
            </button>
          )}
        </div>
        {entries.length === 0 ? (
          <p className="px-3 py-6 text-center text-xs text-muted-foreground">
            No runs yet for this operation.
          </p>
        ) : (
          <ScrollArea className="flex max-h-72 flex-col [&_[data-slot=scroll-area-viewport]>div]:block!">
            <ul className="flex flex-col py-1">
              {entries.map((entry) => (
                <li key={entry.id}>
                  <button
                    type="button"
                    onClick={() => {
                      onReplay(entry);
                      setOpen(false);
                    }}
                    title={entry.variables}
                    className="flex w-full flex-col gap-0.5 px-3 py-1.5 text-left transition-colors hover:bg-muted/60"
                  >
                    <span className="flex w-full items-center gap-2 text-xs">
                      <span
                        className={cn(
                          "size-1.5 shrink-0 rounded-full",
                          entry.status === "success" ? "bg-emerald-500" : "bg-destructive",
                        )}
                      />
                      <span
                        className={
                          entry.status === "success"
                            ? "text-emerald-600 dark:text-emerald-400"
                            : "text-destructive"
                        }
                      >
                        {entry.status === "success" ? "OK" : (entry.httpStatus ?? "Error")}
                      </span>
                      <span className="text-muted-foreground">
                        {formatElapsed(entry.elapsedMs)}
                      </span>
                      {entry.bytes !== undefined && (
                        <span className="text-muted-foreground/60">
                          {formatFileSize(entry.bytes)}
                        </span>
                      )}
                      <span className="ml-auto shrink-0 text-2xs text-muted-foreground/60">
                        {formatTimeAgo(entry.at)}
                      </span>
                    </span>
                    <span className="w-full truncate font-mono text-2xs text-muted-foreground">
                      {historyPreview(entry.variables)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </ScrollArea>
        )}
      </PopoverContent>
    </Popover>
  );
}

function InputTypesReference({ typeNames }: { typeNames: string[] }) {
  const [open, setOpen] = useState(false);
  const sdl = useMemo(
    () =>
      typeNames
        .map((name) => catalog.types[name]?.sdl)
        .filter(Boolean)
        .join("\n\n"),
    [typeNames],
  );

  if (typeNames.length === 0) {
    return null;
  }

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        render={
          <button
            type="button"
            className="flex items-center gap-1 text-2xs font-medium text-muted-foreground transition-colors hover:text-foreground"
          />
        }
      >
        <ChevronRightIcon className={cn("size-3 transition-transform", open && "rotate-90")} />
        <BracesIcon className="size-3" />
        Input types ({typeNames.length})
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ScrollArea
          maskVariant="muted"
          className="mt-1.5 flex max-h-56 flex-col rounded-md border bg-muted/30 [&_[data-slot=scroll-area-viewport]>div]:block!"
        >
          <ShikiCodeBlock code={sdl} lang="graphql" darkTheme="vitesse-dark" />
        </ScrollArea>
      </CollapsibleContent>
    </Collapsible>
  );
}

export function RunPanel({ operation }: { operation: CatalogOperation }) {
  const { theme } = useTheme();
  const scaffold = useMemo(() => scaffoldVariables(operation.variables), [operation.variables]);
  const typeNames = useMemo(() => referencedTypeNames(operation.variables), [operation.variables]);
  const [variables, setVariables] = useState(scaffold);
  const [parseError, setParseError] = useState<string | null>(null);
  const [runState, setRunState] = useState<RunState>({ status: "idle" });
  const [responseView, setResponseView] = useState<ResponseView>("tree");
  const {
    entries: historyEntries,
    record: recordRun,
    clear: clearHistory,
  } = useRunHistory(operation.name);

  useEffect(() => {
    setVariables(scaffold);
    setParseError(null);
    setRunState({ status: "idle" });
  }, [operation.name, scaffold]);

  const jsonLinter = useMemo(
    () =>
      linter((view) => {
        const diagnostics: Diagnostic[] = [];
        const text = view.state.doc.toString().trim();
        if (!text) {
          return diagnostics;
        }
        try {
          JSON.parse(text);
        } catch (error) {
          diagnostics.push({
            from: 0,
            to: view.state.doc.length,
            severity: "error",
            message: error instanceof Error ? error.message : "Invalid JSON",
          });
        }
        return diagnostics;
      }),
    [],
  );

  const run = useCallback(async () => {
    let parsedVariables: Record<string, unknown> = {};
    const trimmed = variables.trim();
    if (trimmed) {
      try {
        parsedVariables = JSON.parse(trimmed) as Record<string, unknown>;
      } catch (error) {
        setParseError(error instanceof Error ? error.message : "Invalid JSON");
        return;
      }
    }
    setParseError(null);

    if (!operation.hash) {
      setRunState({
        status: "error",
        message: "This operation has no persisted hash and cannot be executed.",
      });
      return;
    }

    setRunState({ status: "running" });
    const startedAt = performance.now();
    const document = {
      __meta__: { hash: operation.hash },
      toString: () => operation.sdl,
    };

    try {
      const data = await requestGraphQL<unknown, Record<string, unknown>>({
        document,
        operationName: operation.name,
        variables: parsedVariables,
      });
      const raw = formatJson(data);
      const bytes = new Blob([raw]).size;
      const elapsedMs = performance.now() - startedAt;
      if (raw.length > LARGE_RESPONSE_THRESHOLD) {
        setResponseView("raw");
      }
      setRunState({ status: "success", data, raw, bytes, elapsedMs });
      recordRun({ status: "success", variables, elapsedMs, bytes });
    } catch (error) {
      const elapsedMs = performance.now() - startedAt;
      if (error instanceof GraphQLRequestError) {
        setRunState({
          status: "error",
          message: error.message,
          details: error.graphQLErrors.length > 0 ? formatJson(error.graphQLErrors) : undefined,
          httpStatus: error.status,
          elapsedMs,
        });
        recordRun({
          status: "error",
          variables,
          elapsedMs,
          httpStatus: error.status,
          message: error.message,
        });
        return;
      }
      const message = error instanceof Error ? error.message : "Request failed";
      setRunState({ status: "error", message, elapsedMs });
      recordRun({ status: "error", variables, elapsedMs, message });
    }
  }, [operation.hash, operation.name, operation.sdl, recordRun, variables]);

  const runRef = useRef(run);
  useEffect(() => {
    runRef.current = run;
  }, [run]);

  const extensions = useMemo(
    () => [
      json(),
      jsonLinter,
      EditorView.lineWrapping,
      Prec.highest(
        keymap.of([
          {
            key: "Mod-Enter",
            run: () => {
              void runRef.current();
              return true;
            },
          },
        ]),
      ),
    ],
    [jsonLinter],
  );

  const isRunning = runState.status === "running";
  const isDirty = variables !== scaffold;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={() => void run()} disabled={isRunning} className="gap-1.5">
            {isRunning ? <Spinner className="size-3.5" /> : <PlayIcon className="size-3.5" />}
            {isRunning ? "Running…" : "Run"}
            {!isRunning && (
              <Kbd className="bg-primary-foreground/20 text-primary-foreground">⌘⏎</Kbd>
            )}
          </Button>
          <HistoryPopover
            entries={historyEntries}
            onReplay={(entry) => {
              setVariables(entry.variables);
              setParseError(null);
            }}
            onClear={clearHistory}
          />
          {operation.kind === "mutation" && (
            <span className="flex items-center gap-1 text-xs text-orange-600 dark:text-orange-400">
              <AlertTriangleIcon className="size-3.5" />
              Mutation — executes against live data
            </span>
          )}
        </div>
        {runState.status === "success" && (
          <div className="flex items-center gap-2 text-xs">
            <span className="flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
              <span className="size-1.5 rounded-full bg-emerald-500" />
              OK
            </span>
            <span className="text-muted-foreground">{formatElapsed(runState.elapsedMs)}</span>
            <span className="text-muted-foreground/60">{formatFileSize(runState.bytes)}</span>
          </div>
        )}
        {runState.status === "error" && (
          <div className="flex items-center gap-2 text-xs">
            <span className="flex items-center gap-1.5 text-destructive">
              <span className="size-1.5 rounded-full bg-destructive" />
              {runState.httpStatus ?? "Error"}
            </span>
            {runState.elapsedMs !== undefined && (
              <span className="text-muted-foreground">{formatElapsed(runState.elapsedMs)}</span>
            )}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-1.5">
        <div className="flex items-center justify-between">
          <span className="text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
            Variables
          </span>
          {isDirty && (
            <button
              type="button"
              onClick={() => {
                setVariables(scaffold);
                setParseError(null);
              }}
              className="flex items-center gap-1 text-2xs font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              <RotateCcwIcon className="size-3" />
              Reset
            </button>
          )}
        </div>
        <div
          className={cn(
            "overflow-hidden rounded-md border transition-colors focus-within:border-primary/40",
            parseError && "border-destructive",
          )}
        >
          <CodeMirror
            value={variables}
            onChange={setVariables}
            extensions={extensions}
            theme={theme === "dark" ? darkTheme : lightTheme}
            height="160px"
            basicSetup={{ lineNumbers: true, foldGutter: false, bracketMatching: true }}
          />
        </div>
        {parseError && (
          <p className="flex items-center gap-1.5 text-xs text-destructive">
            <AlertTriangleIcon className="size-3.5 shrink-0" />
            {parseError}
          </p>
        )}
        <InputTypesReference typeNames={typeNames} />
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-1.5">
        <div className="flex items-center justify-between">
          <span className="text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
            Response
          </span>
          {runState.status === "success" && (
            <div className="flex items-center gap-1.5">
              <div className="flex items-center gap-0.5 rounded-md bg-muted/60 p-0.5">
                {(["tree", "raw"] as const).map((view) => (
                  <button
                    key={view}
                    type="button"
                    onClick={() => setResponseView(view)}
                    className={cn(
                      "rounded-sm px-1.5 py-0.5 text-2xs font-medium capitalize transition-colors",
                      responseView === view
                        ? "bg-background text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground",
                    )}
                  >
                    {view}
                  </button>
                ))}
              </div>
              <CopyIconButton value={runState.raw} label="Copy response" size="icon-xxs" />
            </div>
          )}
        </div>
        <ScrollArea
          maskVariant="muted"
          className="min-h-0 flex-1 rounded-md border bg-muted/30 [&_[data-slot=scroll-area-viewport]>div]:block!"
        >
          <AnimatePresence mode="wait" initial={false}>
            {runState.status === "idle" && (
              <m.div
                key="idle"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
                className="flex flex-col items-center justify-center gap-2 px-6 py-12 text-center"
              >
                <PlayIcon className="size-5 text-muted-foreground/50" />
                <p className="text-xs text-muted-foreground">
                  Run the operation to see the response
                </p>
              </m.div>
            )}
            {runState.status === "running" && (
              <m.div
                key="running"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
                className="flex flex-col items-center justify-center gap-2 px-6 py-12"
              >
                <Spinner className="size-4 text-muted-foreground" />
                <p className="text-xs text-muted-foreground">Executing…</p>
              </m.div>
            )}
            {runState.status === "success" && (
              <m.div
                key="success"
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
              >
                <ResponseJson view={responseView} data={runState.data} raw={runState.raw} />
              </m.div>
            )}
            {runState.status === "error" && (
              <m.div
                key="error"
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.15 }}
                className="flex flex-col gap-2 p-3"
              >
                <p className="flex items-center gap-1.5 text-xs font-medium text-destructive">
                  <AlertTriangleIcon className="size-3.5 shrink-0" />
                  {runState.message}
                </p>
                {runState.details !== undefined && (
                  <ShikiCodeBlock code={runState.details} lang="json" darkTheme="vitesse-dark" />
                )}
              </m.div>
            )}
          </AnimatePresence>
        </ScrollArea>
      </div>
    </div>
  );
}
