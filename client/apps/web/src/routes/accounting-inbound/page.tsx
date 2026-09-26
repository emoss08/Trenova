import { PageLayout } from "@/components/navigation/sidebar-layout";
import { ACCOUNTING_SYNC_PATH, accountingSetupPath } from "@/lib/accounting-sync";
import { queries } from "@/lib/queries";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { Link } from "react-router";
import {
  InboundNotices,
  InboundSummary,
  InboundSummarySkeleton,
} from "./_components/inbound-summary";

const SYSTEM: AccountingSystem = "QuickBooksOnline";

const Table = lazy(() => import("./_components/inbound-table"));

export function AccountingInboundPage() {
  const t = useT();
  const syncSummary = useQuery(queries.accountingSync.syncSummary(SYSTEM));
  const overviewQuery = useQuery(queries.accountingSync.inboundOverview(SYSTEM));
  const providerName = syncSummary.data?.providerName ?? "QuickBooks Online";
  const connection = syncSummary.data?.connection ?? null;
  const overview = overviewQuery.data;
  const syncing = connection?.syncEnabledAt != null;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Payments from the books"),
        description: t(
          "Payments recorded in {0} against invoices and settlements Trenova sent, and whether each was brought into Trenova.",
          providerName,
        ),
        actions: (
          <Button
            type="button"
            size="sm"
            variant="outline"
            render={<Link to={ACCOUNTING_SYNC_PATH} />}
          >
            {t("Sync ledger")}
          </Button>
        ),
      }}
    >
      {overviewQuery.isLoading || syncSummary.isLoading ? <InboundSummarySkeleton /> : null}
      {overviewQuery.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>{t("Payments from the books could not be loaded.")}</span>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => void overviewQuery.refetch()}
            >
              {t("Retry")}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}
      {syncSummary.data && !syncing ? (
        <Alert size="sm" variant="info">
          <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
            <span>
              {connection
                ? t(
                    "Trenova reads payments from {0} once setup is finished and syncing is on.",
                    providerName,
                  )
                : t("{0} is not connected yet.", providerName)}
            </span>
            <Button type="button" size="sm" render={<Link to={accountingSetupPath(SYSTEM)} />}>
              {connection ? t("Finish setup") : t("Connect {0}", providerName)}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}
      {overview && syncing ? (
        <>
          <InboundSummary overview={overview} />
          <InboundNotices overview={overview} system={SYSTEM} providerName={providerName} />
        </>
      ) : null}
      <DataTableLazyComponent>
        <Table system={SYSTEM} providerName={providerName} />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
