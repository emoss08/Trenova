import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatRange } from "@trenova/shared/lib/date";
import {
  fetchMyAdvances,
  fetchMyDisputes,
  fetchMyEscrow,
  withdrawSettlementDispute,
} from "@trenova/shared/lib/graphql/driver-portal";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { ExpensesSection } from "../_components/expenses-section";
import { useDashFeatures } from "../_components/use-dash-features";
import { DisputeStatusBadge, disputeCategoryLabels } from "../_components/portal-badges";

export function DashMoneyPage() {
  const features = useDashFeatures();
  const escrow = useQuery({ queryKey: ["dash-escrow"], queryFn: fetchMyEscrow });
  const advances = useQuery({ queryKey: ["dash-advances"], queryFn: fetchMyAdvances });
  const disputes = useQuery({ queryKey: ["dash-disputes"], queryFn: fetchMyDisputes });

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Money</h1>
        <p className="text-sm text-muted-foreground">
          Expenses, escrow, advances, and open questions.
        </p>
      </div>

      {features.allowExpenseSubmission ? <ExpensesSection /> : null}

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Escrow</h2>
        {escrow.isPending ? (
          <Skeleton className="h-32 w-full rounded-2xl" />
        ) : escrow.data?.account ? (
          <div className="rounded-2xl border border-border bg-card p-4">
            <div className="flex items-baseline justify-between">
              <p className="text-2xl font-semibold tracking-tight">
                <AmountDisplay
                  value={escrow.data.account.balanceMinor}
                  currency={escrow.data.account.currencyCode}
                />
              </p>
              {escrow.data.account.targetAmountMinor > 0 ? (
                <p className="text-xs text-muted-foreground">
                  of{" "}
                  <AmountDisplay
                    value={escrow.data.account.targetAmountMinor}
                    currency={escrow.data.account.currencyCode}
                  />{" "}
                  target
                </p>
              ) : null}
            </div>
            {escrow.data.account.targetAmountMinor > 0 ? (
              <Progress
                className="mt-3"
                value={Math.min(
                  100,
                  (escrow.data.account.balanceMinor / escrow.data.account.targetAmountMinor) * 100,
                )}
              />
            ) : null}
            {escrow.data.transactions.length > 0 ? (
              <ul className="mt-4 divide-y divide-border border-t border-border">
                {escrow.data.transactions.slice(0, 8).map((transaction) => (
                  <li
                    key={transaction.id}
                    className="flex items-center justify-between gap-3 py-2.5 text-sm"
                  >
                    <div className="min-w-0">
                      <p className="truncate">{transaction.description || transaction.type}</p>
                      <p className="text-xs text-muted-foreground">
                        {formatRange(transaction.occurredDate, transaction.occurredDate)}
                      </p>
                    </div>
                    <span className="shrink-0 font-medium tabular-nums">
                      <AmountDisplay value={transaction.amountMinor} variant="auto" />
                    </span>
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
        ) : (
          <div className="rounded-2xl border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            You don&apos;t have an escrow account.
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Advances</h2>
        {advances.isPending ? (
          <Skeleton className="h-20 w-full rounded-2xl" />
        ) : advances.data && advances.data.length > 0 ? (
          <ul className="divide-y divide-border rounded-2xl border border-border bg-card">
            {advances.data.map((advance) => (
              <li key={advance.id} className="px-4 py-3">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-sm font-medium">
                    {advance.reference || `${advance.source} advance`}
                  </p>
                  <Badge variant="warning">
                    <AmountDisplay
                      value={advance.outstandingMinor}
                      currency={advance.currencyCode}
                    />{" "}
                    left
                  </Badge>
                </div>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Issued {formatRange(advance.issuedDate, advance.issuedDate)} ·{" "}
                  <AmountDisplay value={advance.recoveredMinor} currency={advance.currencyCode} />{" "}
                  of <AmountDisplay value={advance.amountMinor} currency={advance.currencyCode} />{" "}
                  repaid
                </p>
              </li>
            ))}
          </ul>
        ) : (
          <div className="rounded-2xl border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            No outstanding advances. 🎉
          </div>
        )}
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold">Pay disputes</h2>
        {disputes.isPending ? (
          <Skeleton className="h-20 w-full rounded-2xl" />
        ) : disputes.data && disputes.data.length > 0 ? (
          <ul className="divide-y divide-border rounded-2xl border border-border bg-card">
            {disputes.data.map((dispute) => (
              <DisputeItem key={dispute.id} dispute={dispute} />
            ))}
          </ul>
        ) : (
          <div className="rounded-2xl border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            If something on a settlement looks wrong, flag it from the statement and it will show up
            here.
          </div>
        )}
      </section>
    </div>
  );
}

type DisputeItemProps = {
  dispute: Awaited<ReturnType<typeof fetchMyDisputes>>[number];
};

function DisputeItem({ dispute }: DisputeItemProps) {
  const queryClient = useQueryClient();
  const [pending, setPending] = useState(false);
  const canWithdraw = dispute.status === "Open" || dispute.status === "InReview";

  const handleWithdraw = async () => {
    setPending(true);
    try {
      await withdrawSettlementDispute(dispute.id);
      toast.success("Dispute withdrawn.");
      await queryClient.invalidateQueries({ queryKey: ["dash-disputes"] });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "We couldn't withdraw the dispute.");
    } finally {
      setPending(false);
    }
  };

  return (
    <li className="px-4 py-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium">
          {disputeCategoryLabels[dispute.category] ?? dispute.category}
          {dispute.settlement ? (
            <span className="font-normal text-muted-foreground">
              {" "}
              · {dispute.settlement.settlementNumber}
            </span>
          ) : null}
        </p>
        <DisputeStatusBadge status={dispute.status} />
      </div>
      <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{dispute.description}</p>
      {dispute.resolutionNote ? (
        <p className="mt-2 rounded-md border-l-0 border-border text-xs text-foreground">
          <span className="text-muted-foreground">Carrier response:</span> {dispute.resolutionNote}
        </p>
      ) : null}
      {canWithdraw ? (
        <Button
          variant="ghost"
          size="sm"
          className="mt-1 h-7 px-2 text-xs text-muted-foreground"
          disabled={pending}
          onClick={handleWithdraw}
        >
          {pending ? "Withdrawing..." : "Withdraw"}
        </Button>
      ) : null}
    </li>
  );
}
