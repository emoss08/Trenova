import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowRightIcon,
  BanknoteIcon,
  BarChart3Icon,
  BookOpenIcon,
  FileTextIcon,
  PlusIcon,
  ScaleIcon,
  UsersIcon,
} from "lucide-react";
import { Link } from "react-router";

export function AccountingDashboardPage() {
  const agingQuery = useQuery({
    ...queries.ar.aging(),
  });

  const reconciliationQuery = useQuery({
    ...queries.bankReceipt.summary(),
  });

  const aging = agingQuery.data;
  const recon = reconciliationQuery.data;
  const isLoading = agingQuery.isLoading || reconciliationQuery.isLoading;

  const matchRate =
    recon && recon.importedCount > 0
      ? Math.round((recon.matchedCount / recon.importedCount) * 100)
      : 0;

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Accounting",
        description: "Financial management, journals, AR, and bank reconciliation.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-6">
        {isLoading ? (
          <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full rounded-md" />
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
            <KPICard
              icon={BookOpenIcon}
              label="AR Outstanding"
              value={formatCurrency((aging?.totals.totalOpenMinor ?? 0) / 100)}
            />
            <KPICard
              icon={BanknoteIcon}
              label="Unmatched Receipts"
              value={String(recon?.exceptionCount ?? 0)}
              detail={recon ? formatCurrency(recon.exceptionAmount / 100) : undefined}
            />
            <KPICard
              icon={ScaleIcon}
              label="Match Rate"
              value={`${matchRate}%`}
              detail={recon ? `${recon.matchedCount} of ${recon.importedCount}` : undefined}
            />
            <KPICard
              icon={UsersIcon}
              label="Active Work Items"
              value={String(recon?.activeWorkItemCount ?? 0)}
              detail={
                recon
                  ? `${recon.assignedWorkItemCount} assigned, ${recon.inReviewWorkItemCount} in review`
                  : undefined
              }
            />
          </div>
        )}

        <div>
          <h2 className="mb-3 text-sm font-semibold">Quick Actions</h2>
          <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-3">
            <QuickActionCard
              icon={PlusIcon}
              label="Create Manual Journal"
              to="/accounting/manual-journals/new"
            />
            <QuickActionCard
              icon={BarChart3Icon}
              label="Trial Balance"
              to="/accounting/reports/trial-balance"
            />
            <QuickActionCard
              icon={UsersIcon}
              label="AR Aging Report"
              to="/accounting/ar/aging"
            />
            <QuickActionCard
              icon={BanknoteIcon}
              label="Bank Receipts"
              to="/accounting/reconciliation/bank-receipts"
            />
            <QuickActionCard
              icon={FileTextIcon}
              label="Income Statement"
              to="/accounting/reports/income-statement"
            />
            <QuickActionCard
              icon={ScaleIcon}
              label="Balance Sheet"
              to="/accounting/reports/balance-sheet"
            />
          </div>
        </div>

        {recon ? (
          <div className="grid gap-4 xl:grid-cols-2">
            <Card className="rounded-md">
              <CardHeader>
                <CardTitle className="text-sm font-semibold">Reconciliation Status</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <StatusRow
                    label="Imported"
                    count={recon.importedCount}
                    amount={recon.importedAmount}
                  />
                  <StatusRow
                    label="Matched"
                    count={recon.matchedCount}
                    amount={recon.matchedAmount}
                  />
                  <StatusRow
                    label="Exceptions"
                    count={recon.exceptionCount}
                    amount={recon.exceptionAmount}
                    variant="danger"
                  />
                </div>
                <div className="mt-4">
                  <Link to="/accounting/reconciliation/summary">
                    <Button variant="outline" size="sm">
                      View Full Summary
                      <ArrowRightIcon className="ml-1.5 size-3.5" />
                    </Button>
                  </Link>
                </div>
              </CardContent>
            </Card>

            <Card className="rounded-md">
              <CardHeader>
                <CardTitle className="text-sm font-semibold">Navigation</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-2">
                  <NavLink to="/accounting/manual-journals" label="Manual Journals" />
                  <NavLink to="/accounting/journal-reversals" label="Journal Reversals" />
                  <NavLink to="/accounting/reports/trial-balance" label="Trial Balance" />
                  <NavLink to="/accounting/reports/income-statement" label="Income Statement" />
                  <NavLink to="/accounting/reports/balance-sheet" label="Balance Sheet" />
                  <NavLink to="/accounting/ar/aging" label="AR Aging" />
                  <NavLink to="/accounting/ar/customer-ledger" label="Customer Ledger" />
                  <NavLink to="/accounting/reconciliation/work-queue" label="Work Queue" />
                </div>
              </CardContent>
            </Card>
          </div>
        ) : null}
      </div>
    </PageLayout>
  );
}

function KPICard({
  icon: Icon,
  label,
  value,
  detail,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string;
  detail?: string;
}) {
  return (
    <Card className="group relative gap-0 overflow-hidden rounded-md">
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
        <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
          {label}
        </CardTitle>
        <span className="inline-flex size-7 shrink-0 items-center justify-center rounded-md bg-muted">
          <Icon className="size-4" />
        </span>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold tabular-nums tracking-tight">{value}</p>
        {detail ? <p className="text-[11px] text-muted-foreground">{detail}</p> : null}
      </CardContent>
    </Card>
  );
}

function QuickActionCard({
  icon: Icon,
  label,
  to,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  to: string;
}) {
  return (
    <Link to={to}>
      <Card className="group cursor-pointer gap-0 rounded-md transition-colors hover:bg-muted/50">
        <CardContent className="flex items-center gap-3 py-3">
          <span className="inline-flex size-8 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
            <Icon className="size-4" />
          </span>
          <span className="text-sm font-medium">{label}</span>
        </CardContent>
      </Card>
    </Link>
  );
}

function StatusRow({
  label,
  count,
  amount,
  variant,
}: {
  label: string;
  count: number;
  amount: number;
  variant?: "danger";
}) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <span className="text-sm">{label}</span>
        <span className="ml-2 font-mono text-xs text-muted-foreground">{count}</span>
      </div>
      <span
        className={`text-sm font-medium tabular-nums ${
          variant === "danger" ? "text-red-600 dark:text-red-400" : ""
        }`}
      >
        {formatCurrency(amount / 100)}
      </span>
    </div>
  );
}

function NavLink({ to, label }: { to: string; label: string }) {
  return (
    <Link
      to={to}
      className="flex items-center gap-1 rounded-md px-2 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
    >
      {label}
      <ArrowRightIcon className="size-3" />
    </Link>
  );
}
