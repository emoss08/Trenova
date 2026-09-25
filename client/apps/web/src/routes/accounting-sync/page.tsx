import { PageLayout } from "@/components/navigation/sidebar-layout";
import { ACCOUNTING_MAPPINGS_PATH, accountingSetupPath } from "@/lib/accounting-sync";
import { queries } from "@/lib/queries";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { Link } from "react-router";
import { LedgerHeaderActions } from "./_components/ledger-header-actions";
import { LedgerNotices, LedgerSummary, LedgerSummarySkeleton } from "./_components/ledger-summary";

const SYSTEM: AccountingSystem = "QuickBooksOnline";

const Table = lazy(() => import("./_components/ledger-table"));

export function AccountingSyncLedgerPage() {
  const t = useT();
  const summaryQuery = useQuery(queries.accountingSync.syncSummary(SYSTEM));
  const summary = summaryQuery.data;
  const providerName = summary?.providerName ?? "QuickBooks Online";
  const connection = summary?.connection ?? null;
  const syncing = connection?.syncEnabledAt != null;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Sync ledger"),
        description: t(
          "Every document Trenova sends to {0}, what happened to it, and what to do when it did not go through.",
          providerName,
        ),
        actions: summary ? <LedgerHeaderActions summary={summary} /> : null,
      }}
    >
      {summaryQuery.isLoading ? <LedgerSummarySkeleton /> : null}
      {summaryQuery.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>{t("The sync ledger could not be loaded.")}</span>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => void summaryQuery.refetch()}
            >
              {t("Retry")}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}
      {summary && !syncing ? (
        <Alert size="sm" variant="info">
          <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
            <span>
              {connection
                ? t(
                    "Nothing is sent to {0} until setup is finished: confirm the mappings, then choose a start date.",
                    providerName,
                  )
                : t("{0} is not connected yet.", providerName)}
            </span>
            <span className="flex gap-2">
              {connection?.setupStep === "Mappings" ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  render={<Link to={ACCOUNTING_MAPPINGS_PATH} />}
                >
                  {t("Open mappings")}
                </Button>
              ) : null}
              <Button type="button" size="sm" render={<Link to={accountingSetupPath(SYSTEM)} />}>
                {connection ? t("Finish setup") : t("Connect {0}", providerName)}
              </Button>
            </span>
          </AlertDescription>
        </Alert>
      ) : null}
      {summary && syncing ? (
        <>
          <LedgerSummary summary={summary} />
          <LedgerNotices summary={summary} />
        </>
      ) : null}
      <DataTableLazyComponent>
        <Table system={SYSTEM} providerName={providerName} />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
