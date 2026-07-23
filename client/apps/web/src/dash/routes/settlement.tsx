import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { DriverSettlementStatusBadge } from "@trenova/shared/components/status-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatRange } from "@trenova/shared/lib/date";
import {
  fetchMyDisputes,
  fetchMySettlement,
  type PortalSettlementLine,
} from "@trenova/shared/lib/graphql/driver-portal";
import type { DriverSettlementStatus } from "@trenova/shared/types/driver-pay";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon, FlagIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useParams } from "react-router";
import { DisputeDrawer } from "../_components/dispute-drawer";
import { useDashFeatures } from "../_components/use-dash-features";
import { DisputeStatusBadge } from "../_components/portal-badges";

const categoryOrder = [
  "Earning",
  "GuaranteeTopUp",
  "Reimbursement",
  "Adjustment",
  "CarryForward",
  "Deduction",
  "EscrowContribution",
  "AdvanceRecovery",
] as const;

const categoryLabels: Record<string, string> = {
  Earning: "Earnings",
  GuaranteeTopUp: "Guarantee top-up",
  Reimbursement: "Reimbursements",
  Adjustment: "Adjustments",
  CarryForward: "Carry forward",
  Deduction: "Deductions",
  EscrowContribution: "Escrow",
  AdvanceRecovery: "Advance recovery",
};

const deductionCategories = new Set(["Deduction", "EscrowContribution", "AdvanceRecovery"]);

export function DashSettlementPage() {
  const { settlementId = "" } = useParams();
  const features = useDashFeatures();
  const [disputeOpen, setDisputeOpen] = useState(false);
  const [disputedLine, setDisputedLine] = useState<PortalSettlementLine | null>(null);

  const settlement = useQuery({
    queryKey: ["dash-settlement", settlementId],
    queryFn: () => fetchMySettlement(settlementId),
    enabled: settlementId.length > 0,
  });
  const disputes = useQuery({
    queryKey: ["dash-disputes"],
    queryFn: fetchMyDisputes,
  });

  const settlementDisputes = useMemo(
    () => (disputes.data ?? []).filter((dispute) => dispute.settlementId === settlementId),
    [disputes.data, settlementId],
  );

  const groupedLines = useMemo(() => {
    const lines = settlement.data?.lines ?? [];
    return categoryOrder
      .map((category) => ({
        category,
        lines: lines.filter((line) => line.category === category),
      }))
      .filter((group) => group.lines.length > 0);
  }, [settlement.data]);

  const openDispute = (line: PortalSettlementLine | null) => {
    setDisputedLine(line);
    setDisputeOpen(true);
  };

  if (settlement.isPending) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-36 w-full rounded-2xl" />
        <Skeleton className="h-64 w-full rounded-2xl" />
      </div>
    );
  }

  const data = settlement.data;
  if (!data) {
    return (
      <div className="flex flex-col items-start gap-4">
        <Link to="/dash/pay" className="flex items-center gap-1 text-sm text-muted-foreground">
          <ArrowLeftIcon className="size-4" /> Back to pay
        </Link>
        <div className="w-full rounded-2xl border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
          We couldn&apos;t find that settlement.
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <Link to="/dash/pay" className="flex items-center gap-1 text-sm text-muted-foreground">
        <ArrowLeftIcon className="size-4" /> Back to pay
      </Link>

      <div className="rounded-2xl border border-border bg-card p-4">
        <div className="flex items-center justify-between gap-2">
          <p className="text-sm font-semibold">{data.settlementNumber}</p>
          <DriverSettlementStatusBadge status={data.status as DriverSettlementStatus} />
        </div>
        <p className="mt-2 text-3xl font-semibold tracking-tight">
          <AmountDisplay value={data.netPayMinor} currency={data.currencyCode} />
        </p>
        <p className="mt-1 text-sm text-muted-foreground">
          Net pay for {formatRange(data.periodStart, data.periodEnd)}
        </p>
        <p className="text-sm text-muted-foreground">
          {data.paidAt
            ? `Paid ${formatRange(data.paidAt, data.paidAt)}${data.paymentMethod ? ` via ${data.paymentMethod}` : ""}${data.paymentReference ? ` (${data.paymentReference})` : ""}`
            : `Pay date ${formatRange(data.payDate, data.payDate)}`}
        </p>

        <dl className="mt-4 flex flex-col gap-1.5 border-t border-border pt-3 text-sm">
          <SummaryRow
            label="Gross earnings"
            valueMinor={data.grossEarningsMinor}
            currency={data.currencyCode}
          />
          <SummaryRow
            label="Reimbursements"
            valueMinor={data.reimbursementsMinor}
            currency={data.currencyCode}
          />
          <SummaryRow
            label="Deductions"
            valueMinor={-data.deductionsMinor}
            currency={data.currencyCode}
          />
          {data.carryForwardInMinor !== 0 ? (
            <SummaryRow
              label="Carried in"
              valueMinor={data.carryForwardInMinor}
              currency={data.currencyCode}
            />
          ) : null}
          {data.carryForwardOutMinor !== 0 ? (
            <SummaryRow
              label="Carried to next period"
              valueMinor={-data.carryForwardOutMinor}
              currency={data.currencyCode}
            />
          ) : null}
        </dl>
      </div>

      {settlementDisputes.length > 0 ? (
        <div className="rounded-2xl border border-border bg-card p-4">
          <p className="text-sm font-semibold">Your disputes on this statement</p>
          <ul className="mt-2 flex flex-col gap-2">
            {settlementDisputes.map((dispute) => (
              <li key={dispute.id} className="flex items-center justify-between gap-2 text-sm">
                <span className="truncate text-muted-foreground">
                  {dispute.settlementLine?.description ?? "Whole statement"}
                </span>
                <DisputeStatusBadge status={dispute.status} />
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {groupedLines.map((group) => (
        <section key={group.category} className="rounded-2xl border border-border bg-card">
          <h2 className="px-4 pt-3 text-xs font-medium text-muted-foreground uppercase">
            {categoryLabels[group.category] ?? group.category}
          </h2>
          <ul className="divide-y divide-border">
            {group.lines.map((line) => (
              <li key={line.id} className="flex items-center gap-2 px-4 py-3">
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm">{line.description}</p>
                  <p className="text-xs text-muted-foreground">
                    {[
                      line.proNumber || null,
                      Number(line.quantity) > 0 && Number(line.rate) > 0
                        ? `${line.quantity} × ${line.rate}`
                        : null,
                    ]
                      .filter(Boolean)
                      .join(" · ")}
                  </p>
                </div>
                <span className="text-sm font-medium tabular-nums">
                  <AmountDisplay
                    value={
                      deductionCategories.has(line.category) ? -line.amountMinor : line.amountMinor
                    }
                    currency={data.currencyCode}
                    variant={deductionCategories.has(line.category) ? "negative" : "neutral"}
                  />
                </span>
                {features.allowSettlementDisputes ? (
                  <button
                    type="button"
                    aria-label={`Dispute ${line.description}`}
                    onClick={() => openDispute(line)}
                    className="-mr-1 p-1 text-muted-foreground hover:text-foreground"
                  >
                    <FlagIcon className="size-3.5" />
                  </button>
                ) : null}
              </li>
            ))}
          </ul>
        </section>
      ))}

      {features.allowSettlementDisputes ? (
        <Button variant="outline" className="h-11" onClick={() => openDispute(null)}>
          <FlagIcon className="size-4" />
          Something looks wrong
        </Button>
      ) : null}

      <DisputeDrawer
        settlementId={settlementId}
        line={disputedLine}
        open={disputeOpen}
        onOpenChange={setDisputeOpen}
      />
    </div>
  );
}

function SummaryRow({
  label,
  valueMinor,
  currency,
}: {
  label: string;
  valueMinor: number;
  currency: string;
}) {
  return (
    <div className="flex items-center justify-between">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">
        <AmountDisplay value={valueMinor} currency={currency} />
      </dd>
    </div>
  );
}
