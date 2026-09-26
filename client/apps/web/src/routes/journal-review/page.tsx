import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy } from "react";
import {
  JournalReviewSummarySkeleton,
  JournalReviewSummaryStrip,
} from "./_components/review-summary";

const Table = lazy(() => import("./_components/review-table"));

export function JournalReviewPage() {
  const t = useT();
  const summaryQuery = useQuery(queries.journalEntry.reviewSummary());
  const { allowed: canApprove } = usePermission(Resource.JournalEntry, Operation.Approve);
  const summary = summaryQuery.data;
  const automatic = summary?.postingMode === "Automatic";

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Journals to post"),
        description: summary?.requiresApproval
          ? t(
              "Journals from invoices, payments, adjustments and settlements wait here until someone approves and posts them.",
            )
          : t(
              "Journals from invoices, payments, adjustments and settlements wait here until someone posts them.",
            ),
      }}
    >
      {summaryQuery.isLoading ? <JournalReviewSummarySkeleton /> : null}
      {summaryQuery.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>{t("The journal review queue could not be loaded.")}</span>
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
      {summary ? <JournalReviewSummaryStrip summary={summary} /> : null}
      {summary && automatic && summary.awaitingApproval + summary.readyToPost > 0 ? (
        <Alert size="sm" variant="info">
          <AlertDescription>
            {t(
              "Journals now post on their own. These were written while posting was manual and still need a person.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      {summary && !canApprove ? (
        <Alert size="sm" variant="info">
          <AlertDescription>
            {t(
              "You can see these journals, but approving and posting them needs the Journal Entry approve permission.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </PageLayout>
  );
}
