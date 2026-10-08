import { formatUsd } from "@/lib/ai-usage-format";
import { centsToDecimal, decimalToCents } from "@/lib/decimal-cents";
import type { AIRetrievalSettings } from "@/lib/graphql/ai-retrieval";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { Ic } from "../kit/ic";
import { SecH, Switch } from "../kit/layout";
import { parseBudgetCents } from "./retrieval-model";

type RetrievalSettingsProps = {
  settings: AIRetrievalSettings;
  canUpdate: boolean;
  busy: boolean;
  onBudget: (budgetUsd: string) => void;
  onPause: (paused: boolean) => void;
};

/**
 * What indexing may spend and whether it runs. The budget covers indexing for the
 * calendar month (UTC); searches count against each agent's own budget instead.
 */
export function RetrievalSettingsPanel({
  settings,
  canUpdate,
  busy,
  onBudget,
  onPause,
}: RetrievalSettingsProps) {
  const t = useT();
  const savedCents = decimalToCents(settings.monthlyIndexingBudgetUsd);
  const [draft, setDraft] = useState<{ from: number; text: string } | null>(null);
  const text = draft && draft.from === savedCents ? draft.text : (savedCents / 100).toFixed(2);

  const commit = () => {
    const cents = parseBudgetCents(text);
    setDraft(null);
    if (cents !== null && cents !== savedCents) {
      onBudget(centsToDecimal(cents));
    }
  };

  return (
    <section className="sec">
      <SecH
        t={t("Settings")}
        r={<span className="sh2-n">{t("From the indexer's next round")}</span>}
      />
      <div className="pol">
        <div className="po">
          <div className="po-h">
            <Ic n="dollar" s={14} />
            <b>{t("Monthly indexing budget")}</b>
          </div>
          <div className="bud-i">
            <span>$</span>
            <input
              className="mono"
              inputMode="decimal"
              aria-label={t("Monthly indexing budget")}
              value={text}
              disabled={!canUpdate || busy}
              onChange={(event) => setDraft({ from: savedCents, text: event.target.value })}
              onBlur={commit}
              onKeyDown={(event) => {
                if (event.key === "Enter") event.currentTarget.blur();
                if (event.key === "Escape") setDraft(null);
              }}
            />
            <em>{t("per month")}</em>
          </div>
          <p>
            {t(
              "Indexing stops for the month when it's reached. Searches count against each agent's own budget, not this one.",
            )}
          </p>
          {settings.pausedReason === "Budget" && (
            <p className="po-m t-w">
              {t(
                "This month's {0} is spent. Raise it to resume now.",
                formatUsd(settings.monthlyIndexingBudgetUsd) ?? "",
              )}
            </p>
          )}
        </div>
        <div className="po">
          <div className="po-h">
            <Ic n="pause" s={14} />
            <b>{t("Pause indexing")}</b>
            <span className="sp" />
            <Switch
              on={settings.paused && settings.pausedReason === "Manual"}
              label={t("Pause indexing")}
              disabled={!canUpdate || busy}
              onChange={onPause}
            />
          </div>
          <p>{t("Nothing new is indexed, and agents search by keyword until it's resumed.")}</p>
        </div>
        {!canUpdate && (
          <p className="po-f">{t("Changing these needs the right to update AI providers.")}</p>
        )}
      </div>
    </section>
  );
}
