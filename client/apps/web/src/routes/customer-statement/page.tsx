import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel } from "@/components/section-panel";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { AROpenItem, ARStatementTransaction } from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon } from "lucide-react";
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

  if (daysPastDue <= 0) return <Badge variant="success">{t("Current")}</Badge>;
  if (daysPastDue <= 30) return <Badge variant="warning">{t("{0}d", daysPastDue)}</Badge>;
  return <Badge variant="danger">{t("{0}d", daysPastDue)}</Badge>;
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

  const backButton = (
    <Button variant="outline" size="sm" onClick={() => void navigate("/accounting/ar/open-items")}>
      <ArrowLeftIcon className="size-3.5" />
      {t("Back to open items")}
    </Button>
  );

  if (!customerId) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Customer statement"),
          description: t("No customer specified."),
          actions: backButton,
        }}
      >
        <Alert size="sm">
          <AlertDescription>{t("No customer specified.")}</AlertDescription>
        </Alert>
      </PageLayout>
    );
  }

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{ title: t("Customer statement"), description: t("Loading...") }}
      >
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-20 w-full rounded-lg" />
        <Skeleton className="h-48 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </PageLayout>
    );
  }

  if (isError || !statement) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Customer statement"),
          description: t("Failed to load."),
          actions: backButton,
        }}
      >
        <Alert variant="destructive" size="sm">
          <AlertDescription>
            {t(
              "Could not load the statement. The customer may not exist or you may not have permission.",
            )}
          </AlertDescription>
        </Alert>
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
        actions: backButton,
      }}
    >
      <div className="flex flex-wrap items-end gap-3">
        <div>
          <label className="text-2xs text-muted-foreground mb-1 block font-medium">
            {t("Statement date")}
          </label>
          <Input
            type="date"
            aria-label={t("Statement date")}
            value={statementDate}
            onChange={(e) => setStatementDate(e.target.value)}
            className="h-8 w-[160px] text-xs"
          />
        </div>
        <div>
          <label className="text-2xs text-muted-foreground mb-1 block font-medium">
            {t("Start date")}
          </label>
          <Input
            type="date"
            aria-label={t("Start date")}
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="h-8 w-[160px] text-xs"
          />
        </div>
      </div>

      <KpiStrip aria-label={t("Statement totals")}>
        <KpiStripItem
          label={t("Opening balance")}
          value={formatCurrency(statement.openingBalanceMinor / 100)}
        />
        <KpiStripItem
          label={t("Charges")}
          value={formatCurrency(statement.totalChargesMinor / 100)}
        />
        <KpiStripItem
          label={t("Payments")}
          value={formatCurrency(statement.totalPaymentsMinor / 100)}
          tone="success"
        />
        <KpiStripItem
          label={t("Ending balance")}
          value={formatCurrency(statement.endingBalanceMinor / 100)}
          tone={statement.endingBalanceMinor > 0 ? "danger" : "success"}
        />
      </KpiStrip>

      <SectionPanel title={t("Aging summary")}>
        <div className="grid gap-3 p-3 sm:grid-cols-2 lg:grid-cols-5">
          <AgingBar
            label={t("Current")}
            amount={aging.currentMinor}
            total={agingTotal}
            colorClass="bg-success"
          />
          <AgingBar
            label={t("1-30 Days")}
            amount={aging.days1To30Minor}
            total={agingTotal}
            colorClass="bg-warning"
          />
          <AgingBar
            label={t("31-60 Days")}
            amount={aging.days31To60Minor}
            total={agingTotal}
            colorClass="bg-warning"
          />
          <AgingBar
            label={t("61-90 Days")}
            amount={aging.days61To90Minor}
            total={agingTotal}
            colorClass="bg-danger"
          />
          <AgingBar
            label={t("90+ Days")}
            amount={aging.daysOver90Minor}
            total={agingTotal}
            colorClass="bg-danger"
          />
        </div>
      </SectionPanel>

      <SectionPanel title={t("Transaction history")} count={statement.transactions.length}>
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
                          className="text-xs text-success-foreground"
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
      </SectionPanel>

      {statement.openItems.length > 0 ? (
        <SectionPanel title={t("Open items")} count={statement.openItems.length}>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-muted-foreground text-left">
                <tr>
                  <th className="px-4 py-2.5 text-xs font-medium">{t("Invoice")}</th>
                  <th className="px-4 py-2.5 text-xs font-medium">{t("Invoice date")}</th>
                  <th className="px-4 py-2.5 text-xs font-medium">{t("Due date")}</th>
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
                    {t("Total open")}
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
                      className="text-xs font-semibold"
                    />
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
        </SectionPanel>
      ) : null}
    </PageLayout>
  );
}
