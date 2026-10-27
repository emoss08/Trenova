import { describeApiError } from "@/lib/api-error-message";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaExplainResponse } from "@trenova/shared/types/formula-template";
import { ArrowRightIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import type { TariffRange } from "@trenova/shared/types/tariff";
import { TickedText } from "@/components/tariff-studio/ticked-text";

const MAX_CODE = 70;

export type ExplainState =
  | { status: "loading" }
  | { status: "ready"; data: FormulaExplainResponse }
  | { status: "error"; error: unknown };

/** The lead line, from the bill the server rated; never the model's arithmetic. */
function leadLine(data: FormulaExplainResponse, t: ReturnType<typeof useT>): string {
  const total = data.preview?.bill?.total;
  if (!total) return t("Here's what each statement of the tariff does.");
  return t(
    "On the preview load the tariff bills {0}. Here's how each statement gets there.",
    formatTariffValue(total, t),
  );
}

/**
 * What the tariff bills, in Nova's words: one line from the bill the preview
 * load gets, then one row per statement the trace recorded. Pointing at a row
 * lights its statement in the editor.
 */
export function ExplainPanel({
  source,
  state,
  onHover,
  onClose,
  onFollowUp,
  onRetry,
}: {
  source: string;
  state: ExplainState;
  onHover: (range: TariffRange | null) => void;
  onClose: () => void;
  onFollowUp: () => void;
  onRetry: () => void;
}) {
  const t = useT();
  const loading = state.status === "loading";
  const data = state.status === "ready" ? state.data : null;
  const rows = data?.items ?? [];

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
            {t("Reading the tariff and its trace…")}
          </p>
        )}
        {state.status === "error" && (
          <div className="flex items-center gap-2">
            <p className="text-danger text-sm">
              {describeApiError(state.error, t("Nova could not explain the tariff."))}
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
          {rows.map((item, index) => {
            const { range } = item;
            const code = range ? (source.slice(range.from, range.to).split("\n")[0] ?? "") : "";
            return (
              <div
                key={item.index}
                onMouseEnter={() => onHover(range)}
                onMouseLeave={() => onHover(null)}
                className="hover:bg-surface-hover flex gap-2.5 rounded-sm px-2.5 py-1.5"
              >
                <span className="bg-card text-foreground-muted mt-px flex size-4.5 shrink-0 items-center justify-center rounded-full border font-mono text-2xs">
                  {index + 1}
                </span>
                <div className="flex min-w-0 flex-col gap-0.5">
                  {code !== "" && (
                    <code className="text-foreground-muted truncate font-mono text-xs">
                      {code.length > MAX_CODE ? `${code.slice(0, MAX_CODE - 2)}…` : code}
                    </code>
                  )}
                  <span className="text-sm leading-relaxed">
                    <TickedText text={item.says} />
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
