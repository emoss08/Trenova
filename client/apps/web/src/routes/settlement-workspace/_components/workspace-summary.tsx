import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { StatTile } from "@/components/stat-tile";
import type { SettlementWorkspaceSummary } from "@/lib/graphql/driver-settlement";
import { TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

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
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
        <StatTile
          label="In Pipeline"
          hint="Settlements created for this period, across every status except voided."
          value={<span className="tabular-nums">{pipelineTotal}</span>}
          sub={
            <span>
              {summary.draftCount} draft · {summary.pendingApprovalCount} pending ·{" "}
              {summary.approvedCount} approved
            </span>
          }
        />
        <StatTile
          label="Needs Review"
          clickable={summary.exceptionCount > 0}
          onClick={summary.exceptionCount > 0 ? onFilterAttention : undefined}
          tone={summary.exceptionCount > 0 ? "warn" : undefined}
          hint="Settlements flagged with exceptions — click to filter the queue to them."
          value={
            <span className="flex items-center gap-1.5 tabular-nums">
              {summary.exceptionCount > 0 && <TriangleAlert className="size-3.5" />}
              {summary.exceptionCount}
            </span>
          }
          sub={<span>exception-flagged settlements</span>}
        />
        <StatTile
          label="Posted / Paid"
          hint="Settlements posted to the GL and settlements already paid out."
          value={
            <span className="tabular-nums">
              {summary.postedCount} · {summary.paidCount}
            </span>
          }
          sub={<span>posted · paid</span>}
        />
        <StatTile
          label="Period Net Pay"
          hint="Total net pay across every non-voided settlement in this period."
          value={<AmountDisplay value={summary.totalNetMinor} currency="USD" />}
          sub={
            <span>
              gross <AmountDisplay value={summary.totalGrossMinor} currency="USD" />
            </span>
          }
        />
        <StatTile
          label="Unsettled Pay"
          clickable={summary.unsettledEventCount > 0 || summary.heldEventCount > 0}
          onClick={
            summary.unsettledEventCount > 0 || summary.heldEventCount > 0
              ? onShowUnsettled
              : undefined
          }
          hint="Accrued pay not yet on a settlement — click to review by driver and settle individuals off-cycle."
          value={<AmountDisplay value={summary.unsettledGrossMinor} currency="USD" />}
          sub={
            <span>
              {summary.unsettledEventCount} events · {summary.unsettledWorkerCount} drivers
            </span>
          }
        />
        <StatTile
          label="On Hold"
          tone={summary.heldEventCount > 0 ? "info" : undefined}
          hint="Pay events deliberately deferred — they skip generation until released."
          value={<AmountDisplay value={summary.heldGrossMinor} currency="USD" />}
          sub={<span>{summary.heldEventCount} held events</span>}
        />
      </div>
    </div>
  );
}
