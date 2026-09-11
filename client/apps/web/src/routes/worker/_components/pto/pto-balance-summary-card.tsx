import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router";
import { formatPtoDays, PTOLiabilityDialog } from "./pto-liability-dialog";
import { ptoBalanceSummaryQuery, ptoLiabilityReportQuery } from "./pto-queries";

export function PTOBalanceSummaryCard() {
  const t = useT();

  const { allowed: canManage } = usePermission(Resource.WorkerPTO, Operation.Manage);
  const [reportOpen, setReportOpen] = useState(false);
  const { data, isLoading, isError } = useQuery(ptoBalanceSummaryQuery());
  const liability = useQuery({
    ...ptoLiabilityReportQuery(),
    enabled: canManage,
  });

  if (isLoading) {
    return <Skeleton className="mb-3 h-16 w-full" />;
  }
  if (isError || !data) {
    return null;
  }

  return (
    <>
      <div
        className="mb-3 grid grid-cols-2 gap-2.5 sm:grid-cols-4 xl:grid-cols-5"
        data-testid="pto-balance-summary"
      >
        <SummaryTile label={t("Workers on a policy")} value={data.workersTracked.toLocaleString()} />
        <SummaryTile
          label={t("Not enrolled")}
          value={data.workersUnassigned.toLocaleString()}
          hint={
            data.workersUnassigned > 0 ? (
              <Link to="/hr/pto-policies" className="underline">
                {t("Manage policies")}
              </Link>
            ) : undefined
          }
        />
        <SummaryTile
          label={t("Banked days")}
          value={formatPtoDays(data.totalBalanceDays)}
          hint={t("across all tracked balances")}
        />
        <SummaryTile
          label={t("Pending days")}
          value={formatPtoDays(data.totalPendingDays)}
          hint={t("awaiting a decision")}
        />
        {canManage ? (
          <SummaryTile
            label={t("Owed on exit")}
            value={liability.data ? formatPtoDays(liability.data.liabilityDays) : "—"}
            hint={
              <button
                type="button"
                className="underline"
                onClick={() => setReportOpen(true)}
                data-testid="pto-liability-open"
              >
                {t("View liability report")}
              </button>
            }
          />
        ) : null}
      </div>
      {canManage ? <PTOLiabilityDialog open={reportOpen} onOpenChange={setReportOpen} /> : null}
    </>
  );
}

function SummaryTile({
  label,
  value,
  hint,
}: {
  label: string;
  value: string;
  hint?: React.ReactNode;
}) {
  return (
    <div className="bg-muted/30 rounded-lg border p-3">
      <p className="text-muted-foreground text-[11px] font-medium uppercase">{label}</p>
      <p className="mt-1 text-lg font-semibold tabular-nums">{value}</p>
      {hint ? <p className="text-muted-foreground text-[11px]">{hint}</p> : null}
    </div>
  );
}
