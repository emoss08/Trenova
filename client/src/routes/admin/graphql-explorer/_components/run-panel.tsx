import { darkTheme, lightTheme } from "@/components/formula-editor/editor-theme";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { ShikiCodeBlock } from "@/components/ui/shiki-code-block";
import { GraphQLRequestError, requestGraphQL } from "@/lib/graphql";
import { cn } from "@/lib/utils";
import type { CatalogOperation } from "@/types/graphql-catalog";
import { json } from "@codemirror/lang-json";
import { linter, type Diagnostic } from "@codemirror/lint";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { AlertTriangleIcon, PlayIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { scaffoldVariables } from "./catalog";

type RunState =
  | { status: "idle" }
  | { status: "running" }
  | { status: "success"; data: unknown; elapsedMs: number }
  | { status: "error"; message: string; details?: unknown; elapsedMs?: number };

function formatJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function RunPanel({ operation }: { operation: CatalogOperation }) {
  const { theme } = useTheme();
  const [variables, setVariables] = useState(() => scaffoldVariables(operation.variables));
  const [parseError, setParseError] = useState<string | null>(null);
  const [runState, setRunState] = useState<RunState>({ status: "idle" });

  useEffect(() => {
    setVariables(scaffoldVariables(operation.variables));
    setParseError(null);
    setRunState({ status: "idle" });
  }, [operation.name, operation.variables]);

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

  const extensions = useMemo(
    () => [json(), jsonLinter, EditorView.lineWrapping],
    [jsonLinter],
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
      setRunState({ status: "success", data, elapsedMs: performance.now() - startedAt });
    } catch (error) {
      const elapsedMs = performance.now() - startedAt;
      if (error instanceof GraphQLRequestError) {
        setRunState({
          status: "error",
          message: error.message,
          details: error.graphQLErrors,
          elapsedMs,
        });
        return;
      }
      setRunState({
        status: "error",
        message: error instanceof Error ? error.message : "Request failed",
        elapsedMs,
      });
    }
  }, [operation.hash, operation.name, operation.sdl, variables]);

  const isRunning = runState.status === "running";

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={run} disabled={isRunning} className="gap-1.5">
            <PlayIcon className={cn("size-3.5", isRunning && "animate-pulse")} />
            {isRunning ? "Running…" : "Run"}
          </Button>
          {operation.kind === "mutation" && (
            <span className="flex items-center gap-1 text-xs text-orange-600 dark:text-orange-400">
              <AlertTriangleIcon className="size-3.5" />
              Mutation — executes against live data
            </span>
          )}
        </div>
        {runState.status === "success" && (
          <span className="text-xs text-emerald-600 dark:text-emerald-400">
            {Math.round(runState.elapsedMs)} ms
          </span>
        )}
        {runState.status === "error" && runState.elapsedMs !== undefined && (
          <span className="text-xs text-muted-foreground">{Math.round(runState.elapsedMs)} ms</span>
        )}
      </div>

      <div>
        <span className="text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
          Variables
        </span>
        <div
          className={cn(
            "mt-1 overflow-hidden rounded-md border",
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
          <p className="mt-1 flex items-center gap-1.5 text-xs text-destructive">
            <AlertTriangleIcon className="size-3.5 shrink-0" />
            {parseError}
          </p>
        )}
      </div>

      <div className="flex min-h-0 flex-1 flex-col">
        <span className="text-2xs font-medium tracking-wider text-muted-foreground/70 uppercase">
          Response
        </span>
        <div className="mt-1 min-h-0 flex-1 overflow-auto rounded-md border bg-muted/30">
          {runState.status === "idle" && (
            <p className="p-3 text-xs text-muted-foreground">
              Run the operation to see the response.
            </p>
          )}
          {runState.status === "running" && (
            <p className="p-3 text-xs text-muted-foreground">Executing…</p>
          )}
          {runState.status === "success" && (
            <ShikiCodeBlock code={formatJson(runState.data)} lang="json" darkTheme="vitesse-dark" />
          )}
          {runState.status === "error" && (
            <div className="flex flex-col gap-2 p-3">
              <p className="flex items-center gap-1.5 text-xs font-medium text-destructive">
                <AlertTriangleIcon className="size-3.5 shrink-0" />
                {runState.message}
              </p>
              {runState.details !== undefined && (
                <ShikiCodeBlock
                  code={formatJson(runState.details)}
                  lang="json"
                  darkTheme="vitesse-dark"
                />
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
