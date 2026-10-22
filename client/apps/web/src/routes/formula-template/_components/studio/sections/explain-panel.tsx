import type { EditorRange } from "@/components/formula-editor/highlight-range";
import { describeApiError } from "@/lib/api-error-message";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaExplainResponse } from "@trenova/shared/types/formula-template";
import { ArrowRightIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { TickedText } from "../ai/ticked-text";
import { formatMoney, numericAmount } from "../model/format";

const MAX_CODE = 70;

export type ExplainState =
  | { status: "loading" }
  | { status: "ready"; data: FormulaExplainResponse }
  | { status: "error"; error: unknown };

/** The lead line, priced on the preview load by the engine; never the model's arithmetic. */
function leadLine(data: FormulaExplainResponse, t: ReturnType<typeof useT>): string {
  const raw = numericAmount(data.rawAmount);
  const billed = numericAmount(data.billedAmount);
  if (raw === null || billed === null) {
    return t("Here's what each line of the formula does.");
  }
  const guardrail = data.guardrail;
  if (guardrail?.applied) {
    return guardrail.bound === "max"
      ? t(
          "On the preview load the formula comes to {0}, over the maximum, so it bills {1}. Here's how each line gets there.",
          formatMoney(raw),
          formatMoney(billed),
        )
      : t(
          "On the preview load the formula comes to {0}, under the minimum, so it bills {1}. Here's how each line gets there.",
          formatMoney(raw),
          formatMoney(billed),
        );
  }
  return t(
    "On the preview load the formula comes to {0} and bills {1}. Here's how each line gets there.",
    formatMoney(raw),
    formatMoney(billed),
  );
}

/**
 * What the formula charges, in Nova's words: one line priced on the preview
 * load, then one row per statement down to the final expression. Pointing at
 * a row lights its statement in the editor.
 */
export function ExplainPanel({
  state,
  onHover,
  onClose,
  onFollowUp,
  onRetry,
}: {
  state: ExplainState;
  onHover: (range: EditorRange | null) => void;
  onClose: () => void;
  onFollowUp: () => void;
  onRetry: () => void;
}) {
  const t = useT();
  const loading = state.status === "loading";
  const data = state.status === "ready" ? state.data : null;
  const rows = data
    ? data.statements.map((statement) => ({
        statement,
        says: data.items.find((item) => item.statementIndex === statement.index)?.says ?? "",
      }))
    : [];

  return (
    <div className="bg-card overflow-hidden rounded-lg border" aria-busy={loading || undefined}>
      <div className="flex flex-col gap-2 px-3.5 pt-3 pb-3">
        <div className="flex items-center gap-2">
          <NovaBadge size="sm" spin={loading ? "always" : "none"} />
          <span className="text-sm font-semibold">{t("What this charges")}</span>
          <span className="flex-1" />
          {data && (
            <Button type="button" variant="quiet" size="xs" className="gap-1" onClick={onFollowUp}>
              {t("Ask a follow-up")}
              <ArrowRightIcon className="size-3" />
            </Button>
          )}
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={t("Close the explanation")}
            onClick={onClose}
          >
            <XCloseIcon className="size-3.5" />
          </Button>
        </div>
        {loading && (
          <p className="ui-shimmer text-foreground-muted w-fit rounded-sm text-base">
            {t("Reading the formula and its inputs…")}
          </p>
        )}
        {state.status === "error" && (
          <div className="flex items-center gap-2">
            <p className="text-danger text-sm">
              {describeApiError(state.error, t("Nova could not explain the formula."))}
            </p>
            <Button type="button" variant="outline" size="xs" onClick={onRetry}>
              {t("Try again")}
            </Button>
          </div>
        )}
        {data && <p className="text-base leading-relaxed">{leadLine(data, t)}</p>}
      </div>
      {rows.length > 0 && (
        <div className="bg-sunken mx-2 mb-2 flex flex-col rounded-md py-1">
          {rows.map(({ statement, says }) => {
            const final = statement.kind === "result";
            return (
              <div
                key={statement.index}
                onMouseEnter={() => onHover({ from: statement.from, to: statement.to })}
                onMouseLeave={() => onHover(null)}
                className="hover:bg-surface-hover flex gap-2.5 rounded-sm px-2.5 py-1.5"
              >
                <span
                  className={cn(
                    "mt-px flex size-4.5 shrink-0 items-center justify-center rounded-full font-mono text-2xs",
                    final ? "bg-ink text-ink-foreground" : "bg-card text-foreground-muted border",
                  )}
                >
                  {final ? "=" : statement.index + 1}
                </span>
                <div className="flex min-w-0 flex-col gap-0.5">
                  <code className="text-foreground-muted truncate font-mono text-xs">
                    {statement.code.length > MAX_CODE
                      ? `${statement.code.slice(0, MAX_CODE - 2)}…`
                      : statement.code}
                  </code>
                  <span className="text-sm leading-relaxed">
                    <TickedText text={says} />
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
