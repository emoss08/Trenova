import { useT } from "@trenova/shared/i18n/use-t";
import { SectionPanel, SectionPanelLink } from "@/components/section-panel";
import { ar } from "@/lib/queries/ar";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { Link } from "react-router";
import { PANEL_LIST_HEIGHT, PanelQuiet, RowsBodySkeleton } from "./dashboard-panel";

const TOP_OVERDUE_LIMIT = 10;

export function TopOverdueCustomersCard({ className }: { className?: string }) {
  const t = useT();

  const { data: customers, isLoading } = useQuery(ar.topOverdueCustomers(TOP_OVERDUE_LIMIT));

  const rows = customers ?? [];
  const maxOverdue = rows.reduce((max, row) => Math.max(max, row.overdueMinor), 0);

  return (
    <SectionPanel
      title={t("Top overdue customers")}
      hint={t("by amount past due")}
      action={<SectionPanelLink to="/accounting/ar/aging">{t("View aging")}</SectionPanelLink>}
      className={className}
    >
      {isLoading ? (
        <RowsBodySkeleton label={t("Loading the top overdue customers")} lead="rank" />
      ) : rows.length === 0 ? (
        <PanelQuiet
          height={PANEL_LIST_HEIGHT}
          icon={<CheckCircleIcon className="text-success-foreground size-5" />}
        >
          {t("No overdue balances — nice work")}
        </PanelQuiet>
      ) : (
        <ol className={cn("divide-border-subtle divide-y overflow-y-auto", PANEL_LIST_HEIGHT)}>
          {rows.map((row, index) => (
            <li key={row.customerId}>
              <Link
                to={`/accounting/ar/customer-ledger?customerId=${encodeURIComponent(row.customerId)}`}
                className="hover:bg-surface-hover ui-focus-ring flex items-center gap-3 px-3 py-2 transition-colors"
              >
                <span className="text-muted-foreground w-4 shrink-0 text-right font-mono text-xs tabular-nums">
                  {index + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="truncate text-xs font-medium">{row.customerName}</span>
                    <span className="text-danger-foreground shrink-0 font-mono text-xs tabular-nums">
                      {formatCurrency(row.overdueMinor / 100)}
                    </span>
                  </div>
                  <div className="mt-1 flex items-center gap-2">
                    <div className="bg-sunken h-1 flex-1 overflow-hidden rounded-full">
                      <div
                        className="bg-danger h-full rounded-full"
                        style={{
                          width:
                            maxOverdue > 0 ? `${(row.overdueMinor / maxOverdue) * 100}%` : "0%",
                        }}
                      />
                    </div>
                    <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                      {t(
                        "{0, plural, one {# invoice} other {# invoices}} · oldest {1}d",
                        row.openInvoiceCount,
                        row.oldestDaysPastDue,
                      )}
                    </span>
                  </div>
                </div>
              </Link>
            </li>
          ))}
        </ol>
      )}
    </SectionPanel>
  );
}
