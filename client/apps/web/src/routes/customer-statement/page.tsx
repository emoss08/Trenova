import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { AROpenItem, ARStatementTransaction } from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
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
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

const STATEMENT_COLUMNS = [
  { label: "Date" },
  { label: "Document" },
  { label: "Description" },
  { label: "Charges", numeric: true },
  { label: "Payments", numeric: true },
  { label: "Balance", numeric: true },
] as const;

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix);
}

function AgingBadge({ daysPastDue }: { daysPastDue: number }) {
  const t = useT();

  if (daysPastDue <= 0) return <Badge variant="active">{t("Current")}</Badge>;
  if (daysPastDue <= 30) return <Badge variant="orange">{t("{0}d", daysPastDue)}</Badge>;
  return <Badge variant="inactive">{t("{0}d", daysPastDue)}</Badge>;
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
    <div className="bg-card rounded-lg border px-4 py-3">
      <div className="flex items-center gap-2">
        <Icon className={cn("text-muted-foreground size-4", colorClass)} />
        <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
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
      <div className="bg-muted h-1.5 w-full overflow-hidden rounded-full">
        <div
          className={cn("h-full rounded-full transition-all", colorClass)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

export function CustomerStatementPage() {
  const t = useT();

  const { customerId } = useParams<{ customerId: string }>();
  const navigate = useNavigate();
  const [statementDate, setStatementDate] = useState("");
  const [startDate, setStartDate] = useState("");
  const hasDateFilters = Boolean(statementDate || startDate);
  const clearDateFilters = () => {
    setStatementDate("");
    setStartDate("");
  };

  const queryOptions = useMemo(() => {
    const options: { startDate?: number; asOfDate?: number } = {};
    if (statementDate) {
      const [y, m, d] = statementDate.split("-").map(Number);
      options.asOfDate = Math.floor(new Date(y, m - 1, d, 23, 59, 59).getTime() / 1000);
    }
    if (startDate) {
      const [y, m, d] = startDate.split("-").map(Number);
      options.startDate = Math.floor(new Date(y, m - 1, d).getTime() / 1000);
    }
    return options;
  }, [statementDate, startDate]);

  const {
    data: statement,
    isLoading,
    isError,
  } = useQuery({
    ...queries.ar.customerStatement(customerId!, queryOptions),
    enabled: Boolean(customerId),
  });

  if (!customerId) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Customer Statement"),
          description: t("No customer specified."),
        }}
      >
        <div className="mx-4 mt-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void navigate("/accounting/ar/open-items")}
          >
            <ArrowLeftIcon className="mr-1.5 size-3.5" />
            {t("Back to Open Items")}
          </Button>
        </div>
      </PageLayout>
    );
  }

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{ title: t("Customer Statement"), description: t("Loading...") }}
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
        pageHeaderProps={{ title: t("Customer Statement"), description: t("Failed to load.") }}
      >
        <div className="mx-4 mt-3 space-y-3">
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-300">
            {t(
              "Could not load the statement. The customer may not exist or you may not have permission.",
            )}
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void navigate("/accounting/ar/open-items")}
          >
            <ArrowLeftIcon className="mr-1.5 size-3.5" />
            {t("Back to Open Items")}
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
            {t("Back to Open Items")}
          </Button>
          <div className="flex items-end gap-3">
            <div>
              <label className="text-2xs text-muted-foreground mb-1 block font-medium">
                {t("Statement Date")}
              </label>
              <Input
                type="date"
                aria-label={t("Statement Date")}
                value={statementDate}
                onChange={(e) => setStatementDate(e.target.value)}
                className="h-8 w-[160px] text-xs"
              />
            </div>
            <div>
              <label className="text-2xs text-muted-foreground mb-1 block font-medium">
                {t("Start Date")}
              </label>
              <Input
                type="date"
                aria-label={t("Start Date")}
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="h-8 w-[160px] text-xs"
              />
            </div>
          </div>
        </div>

        <div className="grid gap-2.5 md:grid-cols-4">
          <MetricCard
            label={t("Opening Balance")}
            value={statement.openingBalanceMinor}
            icon={CalendarIcon}
          />
          <MetricCard
            label={t("Charges")}
            value={statement.totalChargesMinor}
            icon={FileTextIcon}
          />
          <MetricCard
            label={t("Payments")}
            value={statement.totalPaymentsMinor}
            icon={WalletIcon}
            colorClass="text-green-600 dark:text-green-400"
          />
          <MetricCard
            label={t("Ending Balance")}
            value={statement.endingBalanceMinor}
            icon={ReceiptTextIcon}
            colorClass={
              statement.endingBalanceMinor > 0
                ? "text-red-600 dark:text-red-400"
                : "text-green-600 dark:text-green-400"
            }
          />
        </div>

        <div className="bg-card rounded-lg border p-4">
          <h3 className="mb-3 text-sm font-semibold">{t("Aging Summary")}</h3>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            <AgingBar
              label={t("Current")}
              amount={aging.currentMinor}
              total={agingTotal}
              colorClass="bg-green-500"
            />
            <AgingBar
              label={t("1-30 Days")}
              amount={aging.days1To30Minor}
              total={agingTotal}
              colorClass="bg-yellow-500"
            />
            <AgingBar
              label={t("31-60 Days")}
              amount={aging.days31To60Minor}
              total={agingTotal}
              colorClass="bg-orange-500"
            />
            <AgingBar
              label={t("61-90 Days")}
              amount={aging.days61To90Minor}
              total={agingTotal}
              colorClass="bg-red-400"
            />
            <AgingBar
              label={t("90+ Days")}
              amount={aging.daysOver90Minor}
              total={agingTotal}
              colorClass="bg-red-600"
            />
          </div>
        </div>

        <div className="bg-card rounded-lg border">
          <div className="border-b px-4 py-3">
            <h3 className="text-sm font-semibold">
              {t("Transaction History")}
              <span className="text-muted-foreground ml-1.5 text-xs font-normal">
                ({statement.transactions.length})
              </span>
            </h3>
          </div>
          {statement.transactions.length === 0 ? (
            <EmptyTable
              title={t("Nothing in this period")}
              description={
                hasDateFilters
                  ? "No invoice, payment or credit touched this account between those dates. Widen the range, or clear it to see everything on record."
                  : "No invoice, payment or credit has touched this account yet. The first one starts the statement."
              }
              columns={STATEMENT_COLUMNS}
              onClearFilters={hasDateFilters ? clearDateFilters : undefined}
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-muted-foreground text-left">
                  <tr>
                    <th className="px-4 py-2.5 text-xs font-medium">{t("Date")}</th>
                    <th className="px-4 py-2.5 text-xs font-medium">{t("Document")}</th>
                    <th className="px-4 py-2.5 text-xs font-medium">{t("Description")}</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">{t("Charges")}</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">{t("Payments")}</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">{t("Balance")}</th>
                  </tr>
                </thead>
                <tbody>
                  {statement.transactions.map((txn: ARStatementTransaction, idx: number) => (
                    <tr
                      key={`${txn.documentNumber}-${idx}`}
                      className="hover:bg-muted/40 border-t transition-colors"
                    >
                      <td className="px-4 py-2.5 text-xs">{formatDate(txn.transactionDate)}</td>
                      <td className="px-4 py-2.5 font-mono text-xs font-medium">
                        {txn.documentNumber}
                      </td>
                      <td className="text-muted-foreground px-4 py-2.5 text-xs">{txn.eventType}</td>
                      <td className="px-4 py-2.5 text-right">
                        {txn.chargeMinor > 0 ? (
                          <AmountDisplay value={txn.chargeMinor} className="text-xs" />
                        ) : (
                          <span className="text-muted-foreground text-xs">{"\u2014"}</span>
                        )}
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        {txn.paymentMinor > 0 ? (
                          <AmountDisplay
                            value={txn.paymentMinor}
                            className="text-xs text-green-600 dark:text-green-400"
                          />
                        ) : (
                          <span className="text-muted-foreground text-xs">{"\u2014"}</span>
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
          <div className="bg-card rounded-lg border">
            <div className="border-b px-4 py-3">
              <h3 className="text-sm font-semibold">
                {t("Open Items")}
                <span className="text-muted-foreground ml-1.5 text-xs font-normal">
                  ({statement.openItems.length})
                </span>
              </h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-muted-foreground text-left">
                  <tr>
                    <th className="px-4 py-2.5 text-xs font-medium">{t("Invoice")}</th>
                    <th className="px-4 py-2.5 text-xs font-medium">{t("Invoice Date")}</th>
                    <th className="px-4 py-2.5 text-xs font-medium">{t("Due Date")}</th>
                    <th className="px-4 py-2.5 text-xs font-medium">{t("Aging")}</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">{t("Total")}</th>
                    <th className="px-4 py-2.5 text-right text-xs font-medium">{t("Open")}</th>
                  </tr>
                </thead>
                <tbody>
                  {statement.openItems.map((item: AROpenItem) => (
                    <tr
                      key={item.invoiceNumber}
                      className="hover:bg-muted/40 border-t transition-colors"
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
                <tfoot className="bg-muted/30 border-t">
                  <tr>
                    <td colSpan={4} className="px-4 py-2.5 text-right text-xs font-medium">
                      {t("Total Open")}
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
