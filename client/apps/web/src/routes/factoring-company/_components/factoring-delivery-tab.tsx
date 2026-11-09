import { FactoringSubmissionStatusBadge } from "@/components/factoring/factoring-badges";
import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelLink, SectionPanelQuiet } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import { usePermission } from "@/hooks/use-permission";
import type { FactoringCompanySchedule } from "@/lib/graphql/factoring-company-page";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { factoringWorkspacePath } from "@/routes/factoring/use-factoring-state";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDate, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Link } from "react-router";
import { FactoringDeliverySection } from "./factoring-delivery-section";

function outcome(t: TranslateFn, schedule: FactoringCompanySchedule): string {
  switch (schedule.status) {
    case "Failed":
      return schedule.lastError ?? t("It didn't send");
    case "Funded":
    case "PartiallyFunded":
      return schedule.fundedAt
        ? t("Funded {0}", formatUnixDate(schedule.fundedAt))
        : t("Funding recorded");
    case "Sent":
      return schedule.transportTarget
        ? t("Sent to {0}, waiting on funding", schedule.transportTarget)
        : t("Sent, waiting on funding");
    case "Sending":
      return t("On its way to the factor");
    case "Rejected":
      return t("The factor rejected it");
    case "Cancelled":
      return t("Cancelled before it was funded");
  }
}

/**
 * How schedules reach the factor and whether that works: the delivery method with its last
 * test, then the last schedules it carried and what became of each.
 */
export default function FactoringDeliveryTab({
  factoringCompanyId,
}: {
  factoringCompanyId: string;
}) {
  const t = useT();
  const { allowed: canReadSchedules } = usePermission(Resource.FactoringSubmission, Operation.Read);
  const schedules = useQuery({
    ...factoringQueries.schedules(factoringCompanyId),
    enabled: canReadSchedules,
  });

  return (
    <div className="flex flex-col gap-4">
      <FactoringDeliverySection factoringCompanyId={factoringCompanyId} />
      {canReadSchedules ? (
        <SectionPanel
          title={t("Schedules sent")}
          action={
            <SectionPanelLink to={factoringWorkspacePath("schedules", factoringCompanyId)}>
              {t("Every schedule")}
            </SectionPanelLink>
          }
        >
          {schedules.isLoading ? (
            <div className="space-y-2 p-3">
              <Skeleton className="h-4 w-72" />
              <Skeleton className="h-4 w-64" />
              <Skeleton className="h-4 w-56" />
            </div>
          ) : schedules.error || !schedules.data ? (
            <div className="p-3">
              <Alert size="sm" variant="destructive">
                <AlertDescription className="flex items-center justify-between gap-3">
                  <span>{t("The schedules could not be loaded.")}</span>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => void schedules.refetch()}
                  >
                    {t("Retry")}
                  </Button>
                </AlertDescription>
              </Alert>
            </div>
          ) : schedules.data.length === 0 ? (
            <SectionPanelQuiet>
              {t(
                "No schedule has gone to this factor yet. The first one shows here once it's sent.",
              )}
            </SectionPanelQuiet>
          ) : (
            <HoverGlideGroup className="bg-card p-1.5">
              <ul className="flex flex-col gap-px">
                {schedules.data.map((schedule) => (
                  <li
                    key={schedule.id}
                    data-glide-target
                    className="relative grid grid-cols-[6rem_minmax(0,1fr)_auto] items-center gap-3 rounded-md px-2.5 py-2 md:grid-cols-[6rem_10rem_6rem_8rem_minmax(0,1fr)]"
                  >
                    <Link
                      to={recordPath("factoring_submission", schedule.id)}
                      className="ui-focus-ring rounded-sm font-mono text-sm hover:underline"
                    >
                      {schedule.number}
                    </Link>
                    <span className="text-muted-foreground hidden text-sm md:block">
                      {formatUnixDateTimeShort(schedule.sentAt ?? schedule.createdAt)}
                    </span>
                    <span className="text-muted-foreground hidden text-right text-sm tabular-nums md:block">
                      {t("{0, plural, one {# invoice} other {# invoices}}", schedule.invoiceCount)}
                    </span>
                    <span className="hidden text-right text-sm tabular-nums md:block">
                      {formatMinor(schedule.faceMinor, schedule.currencyCode)}
                    </span>
                    <span className="flex min-w-0 items-center gap-2">
                      <FactoringSubmissionStatusBadge status={schedule.status} />
                      <span className="text-muted-foreground truncate text-xs">
                        {outcome(t, schedule)}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            </HoverGlideGroup>
          )}
        </SectionPanel>
      ) : null}
    </div>
  );
}
