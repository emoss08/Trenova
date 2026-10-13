import type { ExprDiagnostic, ExprFix } from "@/components/formula-editor/expr-check";
import { guardNullableField, scopeToFormPath } from "@/components/formula-editor/guard-nullable";
import type { EditorRange } from "@/components/formula-editor/highlight-range";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  ExpressionWarning,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { AlertTriangleIcon, InfoCircleIcon, XCircleIcon } from "@trenova/shared/components/icons";
import { useMemo, useState, type ReactNode } from "react";
import { useFormContext, type Path } from "react-hook-form";
import { NovaButton } from "../ai/nova-button";
import { TickedText } from "../ai/ticked-text";
import { engineProblem } from "../model/expression-analysis";
import { useStudio } from "../studio-context";

type Tone = "error" | "warning" | "info";
type Filter = "all" | Tone;

/** How many rows show before the list folds behind "Show all". */
const VISIBLE_ROWS = 5;

type ProblemRow = {
  key: string;
  tone: Tone;
  range: EditorRange | null;
  message: ReactNode;
  /** Where the finding came from, for the eye: the editor, the engine, a breakdown line. */
  source?: string;
  actions: ReactNode;
};

function lineColumn(expression: string, offset: number): { line: number; column: number } {
  const before = expression.slice(0, Math.max(0, Math.min(offset, expression.length)));
  const line = before.split("\n").length;
  const column = before.length - before.lastIndexOf("\n");
  return { line, column };
}

function ToneIcon({ tone }: { tone: Tone }) {
  if (tone === "error") return <XCircleIcon className="text-danger size-3.5 shrink-0" />;
  if (tone === "warning") return <AlertTriangleIcon className="text-warning size-3.5 shrink-0" />;
  return <InfoCircleIcon className="text-foreground-subtle size-3.5 shrink-0" />;
}

function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "ui-focus-ring flex h-5 items-center gap-1 rounded-full px-2 text-2xs transition-colors",
        active
          ? "bg-foreground text-background"
          : "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
      )}
    >
      {children}
    </button>
  );
}

/**
 * The problems strip under the editor: every finding of the language
 * service, what the engine said on the preview load, and the fields the
 * server says can be empty, each on one row with its position, its message
 * and the fix beside it. A row selects what it is about in the editor. A
 * long list folds to its first rows, errors first, and can be narrowed to
 * one severity; the count in the header is always the whole.
 */
export function ExpressionProblems({
  onJump,
  className,
}: {
  onJump: (range: EditorRange) => void;
  className?: string;
}) {
  const t = useT();
  const { analysis, preview, askNova, declareInput } = useStudio();
  const form = useFormContext<FormulaTemplateFormValues>();
  const expression = form.watch("expression") ?? "";
  const result = preview.result;
  const [filter, setFilter] = useState<Filter>("all");
  const [expanded, setExpanded] = useState(false);

  const applyFix = (fix: ExprFix) => {
    if (fix.kind === "declare") {
      declareInput(fix.name, fix.type);
      return;
    }
    let next = expression;
    for (const change of [...fix.changes].sort((a, b) => b.from - a.from)) {
      next = next.slice(0, change.from) + change.insert + next.slice(change.to);
    }
    form.setValue("expression", next, { shouldDirty: true, shouldValidate: true });
  };

  const guard = (warning: ExpressionWarning) => {
    const path = scopeToFormPath(warning.scope) as Path<FormulaTemplateFormValues>;
    const current = form.getValues(path);
    if (typeof current !== "string") return;
    const guarded = guardNullableField(current, warning.field, warning.suggestion);
    if (guarded !== current) {
      form.setValue(path, guarded as never, { shouldDirty: true, shouldValidate: true });
    }
  };

  const fixWithNova = (message: string) =>
    askNova(t("Fix this problem in the formula: {0}", message));

  const rows = useMemo((): ProblemRow[] => {
    const list: ProblemRow[] = [];
    const flaggedFields = new Set(
      analysis.problems.filter((p) => p.code === "nullable").map((p) => p.name ?? ""),
    );
    const order: Record<Tone, number> = { error: 0, warning: 1, info: 2 };
    const sorted = [...analysis.problems].sort(
      (a, b) => order[a.severity] - order[b.severity] || a.from - b.from,
    );
    for (const problem of sorted) {
      list.push(diagnosticRow(problem, `${problem.code}:${problem.from}:${problem.to}`));
    }

    if (analysis.errorCount === 0 && result && !result.valid && !analysis.empty) {
      const engine = engineProblem(
        expression,
        result.error ?? result.message ?? "",
        result.statements,
      );
      list.push({
        key: "engine",
        tone: "error",
        range: { from: engine.from, to: engine.to },
        message: <TickedText text={engine.message} />,
        source: t("on the preview load"),
        actions: (
          <NovaButton onClick={() => fixWithNova(engine.message)}>{t("Fix with Nova")}</NovaButton>
        ),
      });
    }

    for (const warning of result?.valid ? (result.warnings ?? []) : []) {
      if (warning.scope === "expression" && flaggedFields.has(warning.field)) continue;
      list.push({
        key: `${warning.scope}:${warning.field}`,
        tone: "warning",
        range: null,
        message: warning.message,
        source: warning.scope === "expression" ? undefined : t("in a breakdown line"),
        actions: (
          <Button
            type="button"
            variant="outline"
            size="xs"
            className="font-mono"
            onClick={() => guard(warning)}
          >
            {t("Use {0}", warning.suggestion)}
          </Button>
        ),
      });
    }
    return list;

    function diagnosticRow(problem: ExprDiagnostic, key: string): ProblemRow {
      const primary = problem.fixes[0];
      return {
        key,
        tone: problem.severity,
        range: { from: problem.from, to: problem.to },
        message: <TickedText text={problem.message} />,
        actions: (
          <>
            {problem.fixes.map((fix) => (
              <Button
                key={fix.label}
                type="button"
                variant={fix === primary ? "outline" : "ghost"}
                size="xs"
                onClick={() => applyFix(fix)}
              >
                {fix.label}
              </Button>
            ))}
            {problem.fixes.length === 0 && problem.severity === "error" && (
              <NovaButton onClick={() => fixWithNova(problem.message)}>
                {t("Fix with Nova")}
              </NovaButton>
            )}
          </>
        ),
      };
    }
    // The handlers read the form and the expression at click time.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [analysis, result, expression, t]);

  if (rows.length === 0) return null;

  const counts = {
    error: rows.filter((row) => row.tone === "error").length,
    warning: rows.filter((row) => row.tone === "warning").length,
    info: rows.filter((row) => row.tone === "info").length,
  };
  const filtered = filter === "all" ? rows : rows.filter((row) => row.tone === filter);
  const long = rows.length > VISIBLE_ROWS;
  const shown = expanded || !long ? filtered : filtered.slice(0, VISIBLE_ROWS);
  const hidden = filtered.length - shown.length;

  return (
    <div className={cn("bg-card flex flex-col border-t", className)} aria-label={t("Problems")}>
      {long && (
        <div className="flex items-center gap-1 border-b px-2 py-1">
          <FilterChip active={filter === "all"} onClick={() => setFilter("all")}>
            {t("All {0}", rows.length)}
          </FilterChip>
          {counts.error > 0 && counts.error < rows.length && (
            <FilterChip active={filter === "error"} onClick={() => setFilter("error")}>
              {t("{0, plural, one {# error} other {# errors}}", counts.error)}
            </FilterChip>
          )}
          {counts.warning > 0 && counts.warning < rows.length && (
            <FilterChip active={filter === "warning"} onClick={() => setFilter("warning")}>
              {t("{0, plural, one {# warning} other {# warnings}}", counts.warning)}
            </FilterChip>
          )}
          <span className="flex-1" />
          <span className="text-foreground-subtle text-2xs">{t("Errors first, in order")}</span>
        </div>
      )}
      <ScrollArea maskVariant="card" className="max-h-64">
        <ul className="flex flex-col">
          {shown.map((row) => {
            const position = row.range ? lineColumn(expression, row.range.from) : null;
            return (
              <li
                key={row.key}
                className="hover:bg-surface-hover flex items-start gap-2 border-b px-3 py-1.5 text-xs last:border-b-0"
              >
                <button
                  type="button"
                  disabled={!row.range}
                  onClick={() => row.range && onJump(row.range)}
                  className="ui-focus-ring flex min-w-0 flex-1 items-start gap-2 rounded-sm text-left disabled:cursor-default"
                  title={row.range ? t("Select in the editor") : undefined}
                >
                  <span className="mt-px">
                    <ToneIcon tone={row.tone} />
                  </span>
                  {position && (
                    <span className="text-foreground-subtle mt-px shrink-0 font-mono text-2xs tabular-nums">
                      {position.line}:{position.column}
                    </span>
                  )}
                  <span className="min-w-0 flex-1 leading-snug">
                    {row.message}
                    {row.source && <span className="text-foreground-subtle"> · {row.source}</span>}
                  </span>
                </button>
                <span className="flex shrink-0 items-center gap-1">{row.actions}</span>
              </li>
            );
          })}
        </ul>
      </ScrollArea>
      {long && (
        <div className="flex items-center justify-between gap-2 border-t px-3 py-1">
          <span className="text-foreground-subtle text-2xs">
            {hidden > 0
              ? t("{0} of {1} shown", shown.length, filtered.length)
              : t("{0} shown", filtered.length)}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={() => setExpanded((current) => !current)}
          >
            {expanded ? t("Show fewer") : t("Show all {0}", filtered.length)}
          </Button>
        </div>
      )}
    </div>
  );
}
