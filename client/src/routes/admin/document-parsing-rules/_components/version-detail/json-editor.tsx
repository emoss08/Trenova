import { useTheme } from "@/components/theme-provider";
import {
  darkTheme,
  lightTheme,
} from "@/components/formula-editor/editor-theme";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  ruleDocumentSchema,
  type RuleDocument,
  type RuleVersionFormValues,
} from "@/types/document-parsing-rule";
import { json } from "@codemirror/lang-json";
import { linter, type Diagnostic } from "@codemirror/lint";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import {
  CheckIcon,
  AlertTriangleIcon,
  RefreshCwIcon,
  UploadIcon,
  InfoIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useFormContext } from "react-hook-form";

function parseJsonErrorPosition(message: string): number | null {
  const posMatch = message.match(/position\s+(\d+)/i);
  if (posMatch) return parseInt(posMatch[1], 10);
  const lineMatch = message.match(/line\s+(\d+)/i);
  if (lineMatch) return null;
  return null;
}

function getLineFromPosition(text: string, position: number): number {
  let line = 1;
  for (let i = 0; i < position && i < text.length; i++) {
    if (text[i] === "\n") line++;
  }
  return line;
}

export function JsonEditor() {
  const { theme } = useTheme();
  const { getValues, setValue } = useFormContext<RuleVersionFormValues>();
  const [localValue, setLocalValue] = useState(() =>
    JSON.stringify(getValues("ruleDocument"), null, 2),
  );
  const [parseError, setParseError] = useState<string | null>(null);
  const [applied, setApplied] = useState(false);

  useEffect(() => {
    setLocalValue(JSON.stringify(getValues("ruleDocument"), null, 2));
    setParseError(null);
  }, [getValues]);

  const handleApply = useCallback(() => {
    try {
      const parsed = JSON.parse(localValue);
      const result = ruleDocumentSchema.safeParse(parsed);
      if (!result.success) {
        const errors = result.error.issues
          .map((i) => `${i.path.join(".")}: ${i.message}`)
          .join("; ");
        setParseError(errors);
        return;
      }
      setValue("ruleDocument", result.data as RuleDocument, {
        shouldDirty: true,
      });
      setParseError(null);
      setApplied(true);
      setTimeout(() => setApplied(false), 2000);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Invalid JSON";
      const position = e instanceof Error ? parseJsonErrorPosition(msg) : null;
      if (position !== null) {
        const line = getLineFromPosition(localValue, position);
        setParseError(`Invalid JSON at line ${line}: ${msg}`);
      } else {
        setParseError(`Invalid JSON: ${msg}`);
      }
    }
  }, [localValue, setValue]);

  const handleRefresh = useCallback(() => {
    setLocalValue(JSON.stringify(getValues("ruleDocument"), null, 2));
    setParseError(null);
  }, [getValues]);

  const jsonLinter = useMemo(
    () =>
      linter((view) => {
        const diagnostics: Diagnostic[] = [];
        try {
          JSON.parse(view.state.doc.toString());
        } catch (e) {
          const msg = e instanceof Error ? e.message : "Invalid JSON";
          diagnostics.push({
            from: 0,
            to: view.state.doc.length,
            severity: "error",
            message: msg,
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

  const lineCount = localValue.split("\n").length;
  const charCount = localValue.length;

  return (
    <div className="space-y-3">
      <div className="flex items-start gap-2 rounded-md border border-muted bg-muted/30 p-2.5">
        <InfoIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
        <p className="text-xs text-muted-foreground">
          The JSON editor and the Rule Builder share the same underlying data.
          Edits made here must be applied to take effect in the builder, and
          vice versa. Use &ldquo;Refresh from Builder&rdquo; to pull the latest builder
          state into this editor.
        </p>
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  size="sm"
                  onClick={handleApply}
                  className="gap-1"
                >
                  {applied ? (
                    <>
                      <CheckIcon className="size-3.5" />
                      Applied
                    </>
                  ) : (
                    <>
                      <UploadIcon className="size-3.5" />
                      Apply Changes
                    </>
                  )}
                </Button>
              }
            />
            <TooltipContent>
              Validate the JSON and push it into the form. This overwrites the
              Rule Builder state.
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={handleRefresh}
                  className="gap-1"
                >
                  <RefreshCwIcon className="size-3.5" />
                  Refresh from Builder
                </Button>
              }
            />
            <TooltipContent>
              Discard any unapplied JSON edits and reload from the current Rule
              Builder state.
            </TooltipContent>
          </Tooltip>
        </div>
        <span className="text-xs text-muted-foreground">
          {lineCount} line{lineCount !== 1 ? "s" : ""} · {charCount.toLocaleString()} chars
        </span>
      </div>

      {parseError && (
        <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-2.5">
          <AlertTriangleIcon className="mt-0.5 size-3.5 shrink-0 text-destructive" />
          <p className="text-xs text-destructive">{parseError}</p>
        </div>
      )}

      <div
        className={cn(
          "overflow-hidden rounded-md border",
          parseError && "border-destructive",
        )}
      >
        <CodeMirror
          value={localValue}
          onChange={setLocalValue}
          extensions={extensions}
          theme={theme === "dark" ? darkTheme : lightTheme}
          height="500px"
          basicSetup={{
            lineNumbers: true,
            foldGutter: true,
            bracketMatching: true,
            indentOnInput: true,
          }}
        />
      </div>
    </div>
  );
}
