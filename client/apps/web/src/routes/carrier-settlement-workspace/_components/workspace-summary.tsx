import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { StatTile } from "@/components/stat-tile";
import type { CarrierSettlementWorkspaceSummary } from "@/lib/graphql/carrier-settlement";
import type { ReactNode } from "react";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix);
}

export function WorkspaceSummaryStrip({
  summary,
  actions,
}: {
  summary: CarrierSettlementWorkspaceSummary;
  actions: ReactNode;
}) {
  const t = useT();

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
          {t("Pay period")}{" "}
          <span className="text-foreground font-medium">
            {formatDate(summary.periodStart)} – {formatDate(summary.periodEnd - 86400)}
          </span>{" "}
          {t("· pays")} <span className="text-foreground font-medium">{formatDate(summary.payDate)}</span>
        </p>
        {actions}
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
        <StatTile
          label={t("In Pipeline")}
          hint={t("Carrier settlements created for this period, across every status except voided.")}
          value={<span className="tabular-nums">{pipelineTotal}</span>}
          sub={
            <span>
              {t("{0} draft · {1} pending · {2} approved", summary.draftCount, summary.pendingApprovalCount, summary.approvedCount)}
            </span>
          }
        />
        <StatTile
          label={t("Posted / Paid")}
          hint={t("Settlements posted to the GL and settlements already remitted.")}
          value={
            <span className="tabular-nums">
              {summary.postedCount} · {summary.paidCount}
            </span>
          }
          sub={<span>{t("posted · paid")}</span>}
        />
        <StatTile
          label={t("Period Net Payable")}
          hint={t("Total net payable across every non-voided settlement in this period.")}
          value={<AmountDisplay value={summary.totalNetMinor} currency="USD" />}
          sub={
            <span>
              gross <AmountDisplay value={summary.totalGrossMinor} currency="USD" />
            </span>
          }
        />
        <StatTile
          label={t("Unsettled Cost")}
          tone={summary.pendingEventCount > 0 ? "info" : undefined}
          hint={t("Accrued purchased-transportation cost not yet on a settlement.")}
          value={<AmountDisplay value={summary.pendingAmountMinor} currency="USD" />}
          sub={
            <span>
              {t("{0} events · {1} carriers", summary.pendingEventCount, summary.pendingCarrierCount)}
            </span>
          }
        />
        <StatTile
          label={t("Open Batch")}
          hint={t("Whether an AP run for this period is already open.")}
          value={<span>{summary.openBatchId ? t("Open") : t("None")}</span>}
          sub={<span>{summary.openBatchId ? t("generation tops it up") : t("generate to start")}</span>}
        />
      </div>
    </div>
  );
}
