import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useConnectedAccountingSystem } from "@/hooks/use-connected-accounting-system";
import {
  ACCOUNTING_MAPPINGS_PATH,
  accountingSetupPath,
  DEFAULT_ACCOUNTING_SYSTEM,
} from "@/lib/accounting-sync";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { Link } from "react-router";
import { InboundPaymentsLink } from "@/components/accounting-sync/inbound-payments-link";
import { LedgerHeaderActions } from "./_components/ledger-header-actions";
import { LedgerNotices, LedgerSummary, LedgerSummarySkeleton } from "./_components/ledger-summary";

const Table = lazy(() => import("./_components/ledger-table"));

export function AccountingSyncLedgerPage() {
  const t = useT();
  const resolved = useConnectedAccountingSystem("syncSummary");
  const system = resolved.system ?? DEFAULT_ACCOUNTING_SYSTEM;
  const summaryQuery = useQuery({
    ...queries.accountingSync.syncSummary(system),
    enabled: resolved.system !== null,
  });
  const summary = summaryQuery.data;
  const providerName = summary?.providerName ?? t("the accounting system");
  const loading = resolved.isLoading || summaryQuery.isLoading;
  const failed = resolved.isError || summaryQuery.isError;
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
        actions: summary ? (
          <div className="flex flex-wrap items-center gap-2">
            {syncing ? <InboundPaymentsLink system={system} /> : null}
            <LedgerHeaderActions summary={summary} />
          </div>
        ) : null,
      }}
    >
      {loading ? <LedgerSummarySkeleton /> : null}
      {failed ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>{t("The sync ledger could not be loaded.")}</span>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => (resolved.isError ? resolved.retry() : void summaryQuery.refetch())}
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
              <Button type="button" size="sm" render={<Link to={accountingSetupPath(system)} />}>
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
        {resolved.system ? <Table system={resolved.system} providerName={providerName} /> : null}
      </DataTableLazyComponent>
    </PageLayout>
  );
}
