import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useAccountingDriftActions } from "@/hooks/use-accounting-drift-actions";
import { usePermission } from "@/hooks/use-permission";
import { ACCOUNTING_SYNC_PATH, accountingSetupPath } from "@/lib/accounting-sync";
import { queries } from "@/lib/queries";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy } from "react";
import { Link } from "react-router";
import { DriftNotices, DriftSummary, DriftSummarySkeleton } from "./_components/drift-summary";

const SYSTEM: AccountingSystem = "QuickBooksOnline";

const Table = lazy(() => import("./_components/drift-table"));

export function AccountingDriftPage() {
  const t = useT();
  const syncSummary = useQuery(queries.accountingSync.syncSummary(SYSTEM));
  const overviewQuery = useQuery(queries.accountingSync.driftOverview(SYSTEM));
  const { allowed: canUpdate } = usePermission(Resource.AccountingSync, Operation.Update);
  const actions = useAccountingDriftActions();
  const providerName = syncSummary.data?.providerName ?? "QuickBooks Online";
  const connection = syncSummary.data?.connection ?? null;
  const overview = overviewQuery.data;
  const syncing = connection?.syncEnabledAt != null;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Drift findings"),
        description: overview?.checkedAt
          ? t(
              "Documents Trenova sent that differ in {0} now. Last checked {1}.",
              providerName,
              formatUnixDateTimeShort(overview.checkedAt),
            )
          : t("Documents Trenova sent that differ in {0} now. Not checked yet.", providerName),
        actions: (
          <div className="flex gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              render={<Link to={ACCOUNTING_SYNC_PATH} />}
            >
              {t("Sync ledger")}
            </Button>
            {canUpdate && syncing ? (
              <Button
                type="button"
                size="sm"
                isLoading={actions.check.isPending}
                loadingText={t("Starting...")}
                onClick={() => actions.check.mutate(SYSTEM)}
              >
                {t("Check now")}
              </Button>
            ) : null}
          </div>
        ),
      }}
    >
      {overviewQuery.isLoading || syncSummary.isLoading ? <DriftSummarySkeleton /> : null}
      {overviewQuery.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>{t("Drift findings could not be loaded.")}</span>
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
                    "Trenova compares {0} with what it sent once setup is finished and syncing is on.",
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
          <DriftSummary overview={overview} />
          <DriftNotices overview={overview} providerName={providerName} />
        </>
      ) : null}
      <DataTableLazyComponent>
        <Table
          system={SYSTEM}
          providerName={providerName}
          toleranceMinor={overview?.toleranceMinor ?? 0}
        />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
