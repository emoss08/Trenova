import { AlertTriangleIcon, InfoCircleIcon, XCircleIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  TariffDiagnostic,
  TariffFix,
  TariffRange,
  TariffSeverity,
} from "@trenova/shared/types/tariff";
import { useState, type ReactNode } from "react";
import { lineOf } from "@/lib/tariff-source-lines";
import { TickedText } from "./ticked-text";

/** How many rows show before the list folds behind "Show all". */
const VISIBLE_ROWS = 5;

const ORDER: Record<TariffSeverity, number> = { error: 0, warning: 1, info: 2 };

function ToneIcon({ severity }: { severity: TariffSeverity }) {
  if (severity === "error") return <XCircleIcon className="text-danger size-3.5 shrink-0" />;
  if (severity === "warning") {
    return <AlertTriangleIcon className="text-warning size-3.5 shrink-0" />;
  }
  return <InfoCircleIcon className="text-foreground-subtle size-3.5 shrink-0" />;
}

/** Findings in the order a person fixes them: errors first, then by where they are. */
export function sortFindings(diagnostics: readonly TariffDiagnostic[]): TariffDiagnostic[] {
  return [...diagnostics].sort(
    (a, b) => ORDER[a.severity] - ORDER[b.severity] || (a.range?.from ?? 0) - (b.range?.from ?? 0),
  );
}

export type ProblemsListProps = {
  diagnostics: readonly TariffDiagnostic[];
  source: string;
  onReveal: (range: TariffRange) => void;
  onApplyFix: (fix: TariffFix) => void;
  /** More actions on a row, such as asking Nova about it. */
  rowActions?: (diagnostic: TariffDiagnostic) => ReactNode;
  className?: string;
};

/**
 * Every finding of the check under the editor, one row each with its line, its message and its
 * fixes. A row selects what it is about; a finding inside an imported library names the library.
 * A long list folds to its first rows, errors first.
 */
export function ProblemsList({
  diagnostics,
  source,
  onReveal,
  onApplyFix,
  rowActions,
  className,
}: ProblemsListProps) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);
  const rows = sortFindings(diagnostics.filter((diagnostic) => diagnostic.path === null));
  if (rows.length === 0) return null;
  const shown = expanded ? rows : rows.slice(0, VISIBLE_ROWS);

  return (
    <div className={cn("flex flex-col border-t", className)}>
      <ul aria-label={t("Problems")} className="divide-border divide-y">
        {shown.map((diagnostic, index) => {
          const range = diagnostic.range;
          const message = translateMessage(diagnostic.message, t);
          return (
            <li
              // oxlint-disable-next-line react/no-array-index-key -- findings are recomputed per text and have no identity
              key={`${diagnostic.code}:${range?.from ?? "x"}:${index}`}
              data-severity={diagnostic.severity}
              className="flex flex-wrap items-start gap-x-2 gap-y-1 px-3 py-1.5 text-xs"
            >
              <ToneIcon severity={diagnostic.severity} />
              <button
                type="button"
                disabled={!range}
                onClick={() => range && onReveal(range)}
                className="ui-focus-ring hover:text-foreground min-w-0 flex-1 rounded-sm text-left disabled:cursor-default"
              >
                <TickedText
                  text={diagnostic.module ? t("In {0}: {1}", diagnostic.module, message) : message}
                />
              </button>
              {range && !diagnostic.module && (
                <span className="text-foreground-subtle shrink-0 font-mono text-2xs tabular-nums">
                  {t("Ln {0}", lineOf(source, range.from))}
                </span>
              )}
              {(diagnostic.fixes.length > 0 || rowActions) && (
                <span className="flex w-full flex-wrap gap-1 pl-5.5">
                  {diagnostic.fixes.map((fix, fixIndex) => (
                    <Button
                      // oxlint-disable-next-line react/no-array-index-key -- fixes have no identity but their order
                      key={fixIndex}
                      type="button"
                      variant="outline"
                      size="xs"
                      onClick={() => onApplyFix(fix)}
                    >
                      {translateMessage(fix.label, t)}
                    </Button>
                  ))}
                  {rowActions?.(diagnostic)}
                </span>
              )}
            </li>
          );
        })}
      </ul>
      {rows.length > VISIBLE_ROWS && (
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="m-1 self-start"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? t("Show fewer") : t("Show all {0}", rows.length)}
        </Button>
      )}
    </div>
  );
}
