import { InputField } from "@/components/fields/input-field";
import { formatUsd } from "@/lib/ai-usage-format";
import { centsToDecimal, decimalToCents } from "@/lib/decimal-cents";
import type { AIRetrievalSettings } from "@/lib/graphql/ai-retrieval";
import { useT } from "@trenova/shared/i18n/use-t";
import { useForm } from "react-hook-form";
import { aicFieldTrigger } from "../edit/field-trigger";
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
  const savedText = (savedCents / 100).toFixed(2);
  const form = useForm<{ budget: string }>({ values: { budget: savedText } });

  const commit = () => {
    const cents = parseBudgetCents(form.getValues("budget"));
    form.reset({ budget: savedText });
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
          <InputField
            control={form.control}
            name="budget"
            inputMode="decimal"
            aria-label={t("Monthly indexing budget")}
            disabled={!canUpdate || busy}
            leftElement={<span className="text-xs text-muted-foreground">$</span>}
            sideText={t("per month")}
            inputClassProps={aicFieldTrigger}
            onBlur={commit}
            onKeyDown={(event) => {
              if (event.key === "Enter") event.currentTarget.blur();
              if (event.key === "Escape") form.reset({ budget: savedText });
            }}
          />
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
