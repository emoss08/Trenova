import { AmountDisplay } from "@/components/accounting/amount-display";
import { EmptyState } from "@/components/empty-state";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@/lib/utils";
import type { StatementOpenItem, StatementTransaction } from "@/types/customer-statement";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeftIcon,
  CalendarIcon,
  FileTextIcon,
  ReceiptTextIcon,
  WalletIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router";

function formatDate(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function AgingBadge({ daysPastDue }: { daysPastDue: number }) {
  if (daysPastDue <= 0) return <Badge variant="active">Current</Badge>;
  if (daysPastDue <= 30) return <Badge variant="orange">{daysPastDue}d</Badge>;
  return <Badge variant="inactive">{daysPastDue}d</Badge>;
}

function MetricCard({
  label,
  value,
  icon: Icon,
  colorClass,
}: {
  label: string;
  value: number;
  icon: React.ComponentType<{ className?: string }>;
  colorClass?: string;
}) {
  return (
    <div className="rounded-lg border bg-card px-4 py-3">
      <div className="flex items-center gap-2">
        <Icon className={cn("size-4 text-muted-foreground", colorClass)} />
        <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
          {label}
        </p>
      </div>
      <p className={cn("mt-1.5 text-2xl font-semibold tracking-tight tabular-nums", colorClass)}>
        {formatCurrency(value / 100)}
      </p>
    </div>
  );
}

function AgingBar({
  label,
  amount,
  total,
  colorClass,
}: {
  label: string;
  amount: number;
  total: number;
  colorClass: string;
}) {
  const pct = total > 0 ? Math.max((amount / total) * 100, 1) : 0;

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium tabular-nums">{formatCurrency(amount / 100)}</span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full rounded-full transition-all", colorClass)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

export function CustomerStatementPage() {
  const { customerId } = useParams<{ customerId: string }>();
  const navigate = useNavigate();
  const [statementDate, setStatementDate] = useState("");
  const [startDate, setStartDate] = useState("");

  const queryParams = useMemo(() => {
    const params: Record<string, string> = {};
    if (statementDate) {
      const [y, m, d] = statementDate.split("-").map(Number);
      params.statementDate = String(Math.floor(new Date(y, m - 1, d).getTime() / 1000));
    }
    if (startDate) {
      const [y, m, d] = startDate.split("-").map(Number);
      params.startDate = String(Math.floor(new Date(y, m - 1, d).getTime() / 1000));
    }
    return Object.keys(params).length > 0 ? params : undefined;
  }, [statementDate, startDate]);

  const { data: statement, isLoading, isError } = useQuery({
    ...queries.ar.customerStatement(customerId!, queryParams),
    enabled: Boolean(customerId),
  });

  if (!customerId) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: "Customer Statement",
          description: "No customer specified.",
        }}
      >
        <div className="mx-4 mt-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void navigate("/accounting/ar/open-items")}
          >
            <ArrowLeftIcon className="mr-1.5 size-3.5" />
            Back to Open Items
          </Button>
        </div>
      </PageLayout>
    );
  }

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{ title: "Customer Statement", description: "Loading..." }}
      >
        <div className="mx-4 mt-3 space-y-4">
          <Skeleton className="h-8 w-48" />
          <div className="grid gap-2.5 md:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24 rounded-lg" />
            ))}
          </div>
          <Skeleton className="h-48 w-full rounded-lg" />
          <Skeleton className="h-64 w-full rounded-lg" />
        </div>
      </PageLayout>
    );
  }

  if (isError || !statement) {
    return (
      <PageLayout
        pageHeaderProps={{ title: "Customer Statement", description: "Failed to load." }}
      >
        <div className="mx-4 mt-3 space-y-3">
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
            Could not load the statement. The customer may not exist or you may not have permission.
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void navigate("/accounting/ar/open-items")}
          >
            <ArrowLeftIcon className="mr-1.5 size-3.5" />
            Back to Open Items
          </Button>
        </div>
      </PageLayout>
    );
  }

  const aging = statement.aging;
  const agingTotal = aging.totalOpenMinor || 1;

  return (
    <PageLayout
      pageHeaderProps={{
        title: statement.customerName,
        description: `Statement as of ${formatDate(statement.statementDate)}`,
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void navigate("/accounting/ar/open-items")}
          >
            <ArrowLeftIcon className="mr-1.5 size-3.5" />
            Back to Open Items
          </Button>
          <div className="flex items-end gap-3">
            <div>
              <label className="mb-1 block text-2xs font-medium text-muted-foreground">
                Statement Date
              </label>
              <Input
                type="date"
                value={statementDate}
                onChange={(e) => setStatementDate(e.target.value)}
                className="h-8 w-[160px] text-xs"
              />
            </div>
            <div>
              <label className="mb-1 block text-2xs font-medium text-muted-foreground">
                Start Date
              </label>
              <Input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="h-8 w-[160px] text-xs"
              />
            </div>
          </div>
        </div>

        <div className="grid gap-2.5 md:grid-cols-4">
          <MetricCard
            label="Opening Balance"
            value={statement.openingBalanceMinor}
            icon={CalendarIcon}
          />
          <MetricCard
            label="Charges"
            value={statement.totalChargesMinor}
            icon={FileTextIcon}
          />
          <MetricCard
            label="Payments"
            value={statement.totalPaymentsMinor}
            icon={WalletIcon}
            colorClass="text-green-600 dark:text-green-400"
          />
          <MetricCard
            label="Ending Balance"
            value={statement.endingBalanceMinor}
            icon={ReceiptTextIcon}
            colorClass={
              statement.endingBalanceMinor > 0
                ? "text-red-600 dark:text-red-400"
                : "text-green-600 dark:text-green-400"
            }
          />
        </div>

        <div className="rounded-lg border bg-card p-4">
          <h3 className="mb-3 text-sm font-semibold">Aging Summary</h3>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            <AgingBar
              label="Current"
              amount={aging.currentMinor}
              total={agingTotal}
              colorClass="bg-green-500"
            />
            <AgingBar
              label="1-30 Days"
              amount={aging.days1To30Minor}
              total={agingTotal}
              colorClass="bg-yellow-500"
            />
            <AgingBar
              label="31-60 Days"
              amount={aging.days31To60Minor}
              total={agingTotal}
              colorClass="bg-orange-500"
            />
            <AgingBar
              label="61-90 Days"
              amount={aging.days61To90Minor}
              total={agingTotal}
              colorClass="bg-red-400"
            />
            <AgingBar
              label="90+ Days"
              amount={aging.daysOver90Minor}
              total={agingTotal}
              colorClass="bg-red-600"
            />
          </div>
        </div>

        <div className="rounded-lg border bg-card">
          <div className="border-b px-4 py-3">
            <h3 className="text-sm font-semibold">
              Transaction History
              <span className="ml-1.5 text-xs font-normal text-muted-foreground">
                ({statement.transactions.length})
              </span>
            </h3>
          </div>
          {statement.transactions.length === 0 ? (
            <div className="flex justify-center py-10">
              <EmptyState
                title="No transactions"
                description="No transactions found for this period."
                icons={[FileTextIcon]}
                className="max-w-none border-none shadow-none"
              />
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-left text-muted-foreground">
                  <tr>
                    <th className="px-4 py-2.5 text-xs font-medium">Date</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Document</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Description</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">Charges</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">Payments</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">Balance</th>
                  </tr>
                </thead>
                <tbody>
                  {statement.transactions.map((txn: StatementTransaction, idx: number) => (
                    <tr
                      key={`${txn.documentNumber}-${idx}`}
                      className="border-t transition-colors hover:bg-muted/40"
                    >
                      <td className="px-4 py-2.5 text-xs">{formatDate(txn.date)}</td>
                      <td className="px-4 py-2.5 font-mono text-xs font-medium">
                        {txn.documentNumber}
                      </td>
                      <td className="px-4 py-2.5 text-xs text-muted-foreground">
                        {txn.description}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        {txn.chargeMinor > 0 ? (
                          <AmountDisplay value={txn.chargeMinor} className="text-xs" />
                        ) : (
                          <span className="text-xs text-muted-foreground">{"\u2014"}</span>
                        )}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        {txn.paymentMinor > 0 ? (
                          <AmountDisplay
                            value={txn.paymentMinor}
                            className="text-xs text-green-600 dark:text-green-400"
                          />
                        ) : (
                          <span className="text-xs text-muted-foreground">{"\u2014"}</span>
                        )}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        <AmountDisplay
                          value={txn.runningBalanceMinor}
                          className="text-xs font-medium"
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {statement.openItems.length > 0 ? (
          <div className="rounded-lg border bg-card">
            <div className="border-b px-4 py-3">
              <h3 className="text-sm font-semibold">
                Open Items
                <span className="ml-1.5 text-xs font-normal text-muted-foreground">
                  ({statement.openItems.length})
                </span>
              </h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-left text-muted-foreground">
                  <tr>
                    <th className="px-4 py-2.5 text-xs font-medium">Invoice</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Invoice Date</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Due Date</th>
                    <th className="px-4 py-2.5 text-xs font-medium">Aging</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">Total</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">Open</th>
                  </tr>
                </thead>
                <tbody>
                  {statement.openItems.map((item: StatementOpenItem) => (
                    <tr
                      key={item.invoiceNumber}
                      className="border-t transition-colors hover:bg-muted/40"
                    >
                      <td className="px-4 py-2.5 font-mono text-xs font-medium">
                        {item.invoiceNumber}
                      </td>
                      <td className="px-4 py-2.5 text-xs">{formatDate(item.invoiceDate)}</td>
                      <td className="px-4 py-2.5 text-xs">{formatDate(item.dueDate)}</td>
                      <td className="px-4 py-2.5">
                        <AgingBadge daysPastDue={item.daysPastDue} />
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        <AmountDisplay value={item.totalAmountMinor} className="text-xs" />
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        <AmountDisplay
                          value={item.openAmountMinor}
                          className="text-xs font-semibold"
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot className="border-t bg-muted/30">
                  <tr>
                    <td colSpan={4} className="px-4 py-2.5 text-right text-xs font-medium">
                      Total Open
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <AmountDisplay
                        value={statement.openItems.reduce((s, i) => s + i.totalAmountMinor, 0)}
                        className="text-xs font-semibold"
                      />
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <AmountDisplay
                        value={statement.openItems.reduce((s, i) => s + i.openAmountMinor, 0)}
                        className="text-xs font-bold"
                      />
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>
        ) : null}
      </div>
    </PageLayout>
  );
}
