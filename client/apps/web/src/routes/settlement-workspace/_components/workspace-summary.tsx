import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { KpiStrip } from "@/components/kpi/kpi-strip";
import { StatTile } from "@/components/stat-tile";
import type { SettlementWorkspaceSummary } from "@/lib/graphql/driver-settlement";
import type { ReactNode } from "react";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useRichT } from "@trenova/shared/i18n/rich";

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix);
}

export function WorkspaceSummaryStrip({
  summary,
  actions,
  onFilterAttention,
  onShowUnsettled,
}: {
  summary: SettlementWorkspaceSummary;
  actions: ReactNode;
  onFilterAttention: () => void;
  onShowUnsettled: () => void;
}) {
  const t = useT();
  const rt = useRichT();

  const pipelineTotal =
    summary.draftCount +
    summary.pendingApprovalCount +
    summary.approvedCount +
    summary.postedCount +
    summary.paidCount;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-muted-foreground text-xs">
          {rt(
            "Pay period <b>{0} – {1}</b> · pays <b>{2}</b>",
            { b: (c) => <span className="text-foreground font-medium">{c}</span> },
            formatDate(summary.periodStart),
            formatDate(summary.periodEnd - 86400),
            formatDate(summary.payDate),
          )}
        </p>
        {actions}
      </div>
      <KpiStrip>
        <StatTile
          label={t("In pipeline")}
          hint={t("Settlements created for this period, across every status except voided.")}
          value={<span className="tabular-nums">{pipelineTotal}</span>}
          sub={
            <span>
              {t(
                "{0} draft · {1} pending · {2} approved",
                summary.draftCount,
                summary.pendingApprovalCount,
                summary.approvedCount,
              )}
            </span>
          }
        />
        <StatTile
          label={t("Needs review")}
          clickable={summary.exceptionCount > 0}
          onClick={summary.exceptionCount > 0 ? onFilterAttention : undefined}
          tone={summary.exceptionCount > 0 ? "warn" : undefined}
          hint={t("Settlements flagged with exceptions — click to filter the queue to them.")}
          value={<span className="tabular-nums">{summary.exceptionCount}</span>}
          sub={<span>exception-flagged settlements</span>}
        />
        <StatTile
          label={t("Posted / paid")}
          hint={t("Settlements posted to the GL and settlements already paid out.")}
          value={
            <span className="tabular-nums">
              {summary.postedCount} · {summary.paidCount}
            </span>
          }
          sub={<span>{t("posted · paid")}</span>}
        />
        <StatTile
          label={t("Period net pay")}
          hint={t("Total net pay across every non-voided settlement in this period.")}
          value={<AmountDisplay value={summary.totalNetMinor} currency="USD" />}
          sub={
            <span>
              gross <AmountDisplay value={summary.totalGrossMinor} currency="USD" />
            </span>
          }
        />
        <StatTile
          label={t("Unsettled pay")}
          clickable={summary.unsettledEventCount > 0 || summary.heldEventCount > 0}
          onClick={
            summary.unsettledEventCount > 0 || summary.heldEventCount > 0
              ? onShowUnsettled
              : undefined
          }
          hint={t(
            "Accrued pay not yet on a settlement — click to review by driver and settle individuals off-cycle.",
          )}
          value={<AmountDisplay value={summary.unsettledGrossMinor} currency="USD" />}
          sub={
            <span>
              {t(
                "{0} events · {1} drivers",
                summary.unsettledEventCount,
                summary.unsettledWorkerCount,
              )}
            </span>
          }
        />
        <StatTile
          label={t("On hold")}
          tone={summary.heldEventCount > 0 ? "info" : undefined}
          hint={t("Pay events deliberately deferred — they skip generation until released.")}
          value={<AmountDisplay value={summary.heldGrossMinor} currency="USD" />}
          sub={<span>{t("{0} held events", summary.heldEventCount)}</span>}
        />
      </KpiStrip>
    </div>
  );
}
