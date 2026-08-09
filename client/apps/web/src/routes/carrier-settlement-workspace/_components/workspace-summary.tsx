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
  const pipelineTotal =
    summary.draftCount +
    summary.pendingApprovalCount +
    summary.approvedCount +
    summary.postedCount +
    summary.paidCount;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs text-muted-foreground">
          Pay period{" "}
          <span className="font-medium text-foreground">
            {formatDate(summary.periodStart)} – {formatDate(summary.periodEnd - 86400)}
          </span>{" "}
          · pays <span className="font-medium text-foreground">{formatDate(summary.payDate)}</span>
        </p>
        {actions}
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
        <StatTile
          label="In Pipeline"
          hint="Carrier settlements created for this period, across every status except voided."
          value={<span className="tabular-nums">{pipelineTotal}</span>}
          sub={
            <span>
              {summary.draftCount} draft · {summary.pendingApprovalCount} pending ·{" "}
              {summary.approvedCount} approved
            </span>
          }
        />
        <StatTile
          label="Posted / Paid"
          hint="Settlements posted to the GL and settlements already remitted."
          value={
            <span className="tabular-nums">
              {summary.postedCount} · {summary.paidCount}
            </span>
          }
          sub={<span>posted · paid</span>}
        />
        <StatTile
          label="Period Net Payable"
          hint="Total net payable across every non-voided settlement in this period."
          value={<AmountDisplay value={summary.totalNetMinor} currency="USD" />}
          sub={
            <span>
              gross <AmountDisplay value={summary.totalGrossMinor} currency="USD" />
            </span>
          }
        />
        <StatTile
          label="Unsettled Cost"
          tone={summary.pendingEventCount > 0 ? "info" : undefined}
          hint="Accrued purchased-transportation cost not yet on a settlement."
          value={<AmountDisplay value={summary.pendingAmountMinor} currency="USD" />}
          sub={
            <span>
              {summary.pendingEventCount} events · {summary.pendingCarrierCount} carriers
            </span>
          }
        />
        <StatTile
          label="Open Batch"
          hint="Whether an AP run for this period is already open."
          value={<span>{summary.openBatchId ? "Open" : "None"}</span>}
          sub={<span>{summary.openBatchId ? "generation tops it up" : "generate to start"}</span>}
        />
      </div>
    </div>
  );
}
