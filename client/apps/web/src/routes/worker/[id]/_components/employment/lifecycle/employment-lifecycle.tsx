import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { LifecycleRouteSkeleton } from "../employment-skeleton";
import { employmentCycleEventsQuery, workerChecklistsQuery } from "../employment-reads";
import { buildLifecycle } from "./lifecycle-model";
import { LifecycleRoute } from "./lifecycle-route";

type EmploymentLifecycleProps = {
  workerId: string;
  /** The hire date on the worker's profile, for a history that starts without a Hired event. */
  hireDate: number | null;
  /** Null when the reader may not record events. */
  onRecord: (() => void) | null;
  /** The record pane, read on intent when Record event is pointed at or focused. */
  recordPane: { preload: () => Promise<unknown> };
};

/**
 * The lifecycle route with its reads: the cycle's milestone events and the checklists (both
 * shared with the views below, so they cost nothing more there). The tenure on the Active stop
 * follows the clock, which ticks here, in the leaf that prints it.
 */
export function EmploymentLifecycle({
  workerId,
  hireDate,
  onRecord,
  recordPane,
}: EmploymentLifecycleProps) {
  const t = useT();
  const now = useNow("minute");
  const events = useQuery(employmentCycleEventsQuery(workerId));
  const checklists = useQuery(workerChecklistsQuery(workerId));

  if (events.isError || checklists.isError) {
    return (
      <ErrorState
        layout="compact"
        error={events.error ?? checklists.error}
        title={t("The employment lifecycle could not be loaded")}
        onRetry={() => {
          if (events.isError) void events.refetch();
          if (checklists.isError) void checklists.refetch();
        }}
      />
    );
  }
  if (!events.data || !checklists.data) return <LifecycleRouteSkeleton />;

  return (
    <LifecycleRoute
      stops={buildLifecycle({ events: events.data, checklists: checklists.data, hireDate, now }, t)}
      onRecord={onRecord}
      recordPane={recordPane}
    />
  );
}
