import { SectionPanel } from "@/components/section-panel";
import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { describeTenure, monthsBetween } from "../lifecycle/lifecycle-model";

function latest(events: readonly WorkerEmploymentEventRow[], kind: string): number | null {
  let at: number | null = null;
  for (const event of events) {
    if (event.kind === kind && (at === null || event.effectiveAt > at)) at = event.effectiveAt;
  }
  return at;
}

/** The dates the employment turns on, read from the worker's profile and the history. */
export function EmploymentDates({
  worker,
  events,
  now,
}: {
  worker: WorkerRecord;
  events: readonly WorkerEmploymentEventRow[];
  now: number;
}) {
  const t = useT();
  const hired = worker.profile?.hireDate ?? latest(events, "Hired");
  const terminated = worker.profile?.terminationDate ?? null;
  const probation = latest(events, "ProbationEnded");
  const rate = latest(events, "RateChanged");
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
            {date(rate)}
          </DescriptionItem>
          <DescriptionItem label={t("Tenure")} numeric>
            {hired ? (
              describeTenure(monthsBetween(hired, terminated ?? now), t)
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Terminated")} numeric>
            {date(terminated)}
          </DescriptionItem>
        </DescriptionList>
      </div>
    </SectionPanel>
  );
}
