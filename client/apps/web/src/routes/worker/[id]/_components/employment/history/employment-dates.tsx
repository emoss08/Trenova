import { SectionPanel } from "@/components/section-panel";
import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { employmentCycleEventsQuery, latestEmploymentEventQuery } from "../employment-reads";
import { describeTenure, monthsBetween } from "../lifecycle/lifecycle-model";
import { ShimmerPanel } from "../../tab-skeleton-parts";

function latest(events: readonly WorkerEmploymentEventRow[], kind: string): number | null {
  let at: number | null = null;
  for (const event of events) {
    if (event.kind === kind && (at === null || event.effectiveAt > at)) at = event.effectiveAt;
  }
  return at;
}

/** Length of service up to now, or to the termination; the clock lives here, in the leaf. */
function Tenure({ hired, terminated }: { hired: number; terminated: number | null }) {
  const t = useT();
  const now = useNow("minute");
  return <>{describeTenure(monthsBetween(hired, terminated ?? now), t)}</>;
}

/**
 * The dates the employment turns on, from the worker's profile, the cycle's milestones (the
 * read the lifecycle route already holds) and the newest rate change, so none of them depends
 * on how much of the history has been paged in.
 */
export function EmploymentDates({ workerId, worker }: { workerId: string; worker: WorkerRecord }) {
  const t = useT();
  const cycle = useQuery(employmentCycleEventsQuery(workerId));
  const rate = useQuery(latestEmploymentEventQuery(workerId, "RateChanged"));

  if (cycle.isPending || rate.isPending) return <ShimmerPanel rows={5} />;
  if (cycle.isError || rate.isError) {
    return (
      <SectionPanel title={t("Dates")}>
        <div className="p-3">
          <ErrorState
            layout="compact"
            error={cycle.error ?? rate.error}
            title={t("The employment dates could not be loaded")}
            onRetry={() => {
              if (cycle.isError) void cycle.refetch();
              if (rate.isError) void rate.refetch();
            }}
          />
        </div>
      </SectionPanel>
    );
  }

  const hired = worker.profile?.hireDate ?? latest(cycle.data, "Hired");
  const terminated = worker.profile?.terminationDate ?? null;
  const probation = latest(cycle.data, "ProbationEnded");
  const rateChanged = rate.data?.effectiveAt ?? null;
  const date = (value: number | null) =>
    value ? formatUnixDateMedium(value) : <DescriptionEmpty />;

  return (
    <SectionPanel title={t("Dates")}>
      <div className="px-3 py-1.5">
        <DescriptionList layout="split">
          <DescriptionItem label={t("Hired")} numeric>
            {date(hired)}
          </DescriptionItem>
          <DescriptionItem label={t("Probation ended")} numeric>
            {date(probation)}
          </DescriptionItem>
          <DescriptionItem label={t("Last rate change")} numeric>
            {date(rateChanged)}
          </DescriptionItem>
          <DescriptionItem label={t("Tenure")} numeric>
            {hired ? <Tenure hired={hired} terminated={terminated} /> : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Terminated")} numeric>
            {date(terminated)}
          </DescriptionItem>
        </DescriptionList>
      </div>
    </SectionPanel>
  );
}
