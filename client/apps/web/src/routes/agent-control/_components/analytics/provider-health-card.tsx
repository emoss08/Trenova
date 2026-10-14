import type { AIAnalyticsProviderHealth } from "@/lib/graphql/ai-analytics";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { AiAnalyticsCompareBy } from "@trenova/graphql/generated/graphql";
import { memo } from "react";
import { Mark } from "../kit/marks";
import { formatLatency, percent, PROVIDER_ERRORS } from "./analytics-model";
import { AnalyticsCard } from "./card";

/** Above this share of calls failing before an answer, the rate reads as a problem. */
const ERROR_RATE_ALARM = 0.05;
/** Above this many calls passed to the next provider, the count reads as a warning. */
const FELL_TO_NEXT_ALARM = 50;
/** A deployment quiet for longer than this says when it last ran. */
const QUIET_SECONDS = 86_400;

type ProviderHealthCardProps = {
  providers: readonly AIAnalyticsProviderHealth[];
  hidden: readonly string[];
  compareBy: AiAnalyticsCompareBy;
  until: number;
  onOpenProvider: (providerId: string) => void;
};

/** Errors that happened before a model could answer, per model on each provider. */
export const ProviderHealthCard = memo(function ProviderHealthCard({
  providers,
  hidden,
  compareBy,
  until,
  onOpenProvider,
}: ProviderHealthCardProps) {
  const t = useT();
  const rows = providers.filter(
    (row) => !hidden.includes(compareBy === "Provider" ? (row.provider.id ?? "") : row.modelKey),
  );
  const worst = Math.max(0.0001, ...rows.map((row) => row.errorRate ?? 0));

  return (
    <AnalyticsCard
      title={t("Provider health")}
      description={t(
        "Errors that happened before the model could answer. Always shown by provider.",
      )}
      className="an-ph"
    >
      {rows.length === 0 ? (
        <p className="an-nil">{t("No provider was called in this scope.")}</p>
      ) : (
        <>
          <div className="an-phh" aria-hidden>
            <span>{t("Provider")}</span>
            <span>{t("Errors")}</span>
            <span className="r">{t("Rate")}</span>
            <span className="r">{t("Fell to next")}</span>
            <span className="r">{t("First token")}</span>
          </div>
          {rows.map((row, index) => {
            const quiet =
              row.lastCalledAt !== null &&
              row.lastCalledAt !== undefined &&
              until - row.lastCalledAt > QUIET_SECONDS;
            const open = row.provider.id && !row.provider.removed ? row.provider.id : null;
            const rate = row.errorRate ?? 0;
            return (
              <button
                key={row.key}
                type="button"
                className="an-phr ui-focus-ring"
                disabled={!open}
                onClick={() => open && onOpenProvider(open)}
              >
                <span className="an-phn">
                  <Mark
                    provider={{
                      name: row.provider.name,
                      kind: row.provider.kind ?? undefined,
                      baseUrl: row.provider.baseUrl,
                    }}
                    s={22}
                  />
                  <span className="tl">
                    <b>{row.provider.name}</b>
                    <span>
                      {row.model}
                      {quiet && row.lastCalledAt
                        ? ` · ${t("last ran {0}", formatUnixMonthDay(row.lastCalledAt))}`
                        : ""}
                      {!row.provider.enabled && !row.provider.removed ? ` · ${t("off")}` : ""}
                    </span>
                  </span>
                </span>
                <span className="an-pe">
                  {row.errors > 0 ? (
                    <span
                      style={{ width: `${Math.max(3, (rate / worst) * 100)}%` }}
                      role="img"
                      aria-label={row.errorsByKind
                        .map(
                          (item) =>
                            `${t(PROVIDER_ERRORS.find((kind) => kind.kind === item.kind)?.label ?? item.kind)} ${item.calls}`,
                        )
                        .join(", ")}
                    >
                      {row.errorsByKind.map((item) => (
                        <i
                          key={item.kind}
                          className={`e-${item.kind}`}
                          style={{ flexGrow: item.calls, animationDelay: `${index * 50}ms` }}
                          title={`${t(PROVIDER_ERRORS.find((kind) => kind.kind === item.kind)?.label ?? item.kind)}: ${item.calls.toLocaleString()}`}
                        />
                      ))}
                    </span>
                  ) : (
                    <em className="dim">{t("None")}</em>
                  )}
                </span>
                <span className={cn("mono r", rate > ERROR_RATE_ALARM && "t-d")}>
                  {row.errors > 0 ? percent(rate, 1) : "—"}
                </span>
                <span
                  className={cn(
                    "mono r",
                    row.fellToNext > FELL_TO_NEXT_ALARM ? "t-w" : row.fellToNext === 0 && "dim",
                  )}
                >
                  {row.fellToNext > 0 ? row.fellToNext.toLocaleString() : "—"}
                </span>
                <span className="mono r">{formatLatency(row.firstTokenMedianMs)}</span>
              </button>
            );
          })}
          <div className="an-lgd">
            {PROVIDER_ERRORS.map((kind) => (
              <span key={kind.kind}>
                <i className={`e-${kind.kind}`} />
                {t(kind.label)}
              </span>
            ))}
          </div>
        </>
      )}
    </AnalyticsCard>
  );
});
