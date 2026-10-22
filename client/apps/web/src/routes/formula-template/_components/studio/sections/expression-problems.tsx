import type { EditorRange } from "@/components/formula-editor/highlight-range";
import { EXPRESSION_SCOPE } from "@/lib/formula-preview";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormulaDiagnostic,
  FormulaFix,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { AlertTriangleIcon, InfoCircleIcon, XCircleIcon } from "@trenova/shared/components/icons";
import { useMemo, useState, type ReactNode } from "react";
import { useFormContext } from "react-hook-form";
import { NovaButton } from "../ai/nova-button";
import { TickedText } from "../ai/ticked-text";
import { scopeTarget, type ScopeTarget } from "../model/preview-scope";
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

/** The text of a fix's edits applied to `text`, last edit first so offsets hold. */
function applyChanges(text: string, fix: Extract<FormulaFix, { kind: "replace" }>): string {
  let next = text;
  for (const change of [...fix.changes].sort((a, b) => b.from - a.from)) {
    next = next.slice(0, change.from) + change.insert + next.slice(change.to);
  }
  return next;
}

function findingKey(scope: string, finding: FormulaDiagnostic): string {
  return `${scope}:${finding.code}:${finding.from}:${finding.to}:${finding.message}:${JSON.stringify(finding.args ?? [])}`;
}

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
        "ui-focus-ring flex h-5 items-center gap-1 rounded-badge px-2 text-2xs transition-colors",
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
 * The problems strip under the editor: every finding of `formula.check` on
 * the expression, what the engine said when it ran the preview load, and what
 * the server's check found on the breakdown lines, each on one row with its
 * position, its message and the fix beside it. A row selects what it is about
 * in the editor. A long list folds to its first rows, errors first, and can be
 * narrowed to one severity; the count in the header is always the whole.
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
  const breakdownDefinitions = form.watch("breakdownDefinitions");
  const result = preview.result;
  const request = preview.lastRequest;
  const [filter, setFilter] = useState<Filter>("all");
  const [expanded, setExpanded] = useState(false);

  const applyFix = (fix: FormulaFix, path: ScopeTarget["path"] = "expression") => {
    if (fix.kind === "declare") {
      declareInput(fix.name, fix.type);
      return;
    }
    const current = form.getValues(path);
    if (typeof current !== "string") return;
    form.setValue(path, applyChanges(current, fix), { shouldDirty: true, shouldValidate: true });
  };

  const fixWithNova = (message: string) =>
    askNova(t("Fix this problem in the formula: {0}", message));

  const rows = useMemo((): ProblemRow[] => {
    const list: ProblemRow[] = [];
    const order: Record<Tone, number> = { error: 0, warning: 1, info: 2 };
    const sorted = [...analysis.problems].sort(
      (a, b) => order[a.severity] - order[b.severity] || a.from - b.from,
    );
    // Two findings can share a span (a `let` missing both its name and its
    // value), so the key carries the message, and a repeat gets its index:
    // React keeps a stale row on screen when keys collide.
    const seen = new Map<string, number>();
    const keyed = (base: string) => {
      const repeat = seen.get(base) ?? 0;
      seen.set(base, repeat + 1);
      return repeat === 0 ? base : `${base}#${repeat}`;
    };
    for (const problem of sorted) {
      list.push(
        findingRow(problem, keyed(findingKey(EXPRESSION_SCOPE, problem)), {
          path: "expression",
          range: { from: problem.from, to: problem.to },
        }),
      );
    }

    if (analysis.pending && !analysis.empty) {
      list.push({
        key: "pending",
        tone: "info",
        range: null,
        message: t("Field and function names are checked once the shipment schema has loaded"),
        actions: null,
      });
    }

    const failure = result?.error;
    if (analysis.errorCount === 0 && failure && !analysis.empty) {
      const target = scopeTarget(failure.scope, request, breakdownDefinitions ?? []);
      const onExpression = target?.path === "expression" && target.checked === expression;
      const message = translateMessage(failure, t);
      list.push({
        key: "engine",
        tone: "error",
        range:
          onExpression && failure.from != null && failure.to != null
            ? { from: failure.from, to: failure.to }
            : onExpression
              ? { from: 0, to: expression.length }
              : null,
        message: <TickedText text={message} />,
        source:
          failure.scope === EXPRESSION_SCOPE ? t("on the preview load") : t("in a breakdown line"),
        actions: <NovaButton onClick={() => fixWithNova(message)}>{t("Fix with Nova")}</NovaButton>,
      });
    }

    // The editor checks the expression itself; the server's findings on the
    // breakdown lines are listed while the line still reads as it was checked.
    for (const finding of result?.diagnostics ?? []) {
      if (finding.scope === EXPRESSION_SCOPE) continue;
      const target = scopeTarget(finding.scope, request, breakdownDefinitions ?? []);
      if (!target || form.getValues(target.path) !== target.checked) continue;
      list.push(
        findingRow(finding, keyed(findingKey(finding.scope, finding)), {
          path: target.path,
          range: null,
          source: t("in a breakdown line"),
        }),
      );
    }
    return list;

    function findingRow(
      problem: FormulaDiagnostic,
      key: string,
      at: { path: ScopeTarget["path"]; range: EditorRange | null; source?: string },
    ): ProblemRow {
      const primary = problem.fixes[0];
      const message = translateMessage(problem, t);
      return {
        key,
        tone: problem.severity,
        range: at.range,
        message: <TickedText text={message} />,
        source: at.source,
        actions: (
          <>
            {problem.fixes.map((fix) => {
              const label = translateMessage(fix.label, t);
              return (
                <Button
                  key={label}
                  type="button"
                  variant={fix === primary ? "outline" : "ghost"}
                  size="xs"
                  className={problem.code === "nullable" ? "font-mono" : undefined}
                  onClick={() => applyFix(fix, at.path)}
                >
                  {label}
                </Button>
              );
            })}
            {problem.fixes.length === 0 && problem.severity === "error" && (
              <NovaButton onClick={() => fixWithNova(message)}>{t("Fix with Nova")}</NovaButton>
            )}
          </>
        ),
      };
    }
    // The handlers read the form and the expression at click time.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [analysis, result, request, expression, breakdownDefinitions, t]);

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
