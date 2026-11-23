import { useWorkPane } from "@/components/work-pane";
import { workerOverviewQuery } from "@/lib/queries/worker";
import type { WorkerTabProps, WorkerTarget } from "@/routes/worker/use-worker-state";
import { useWorkerAreas } from "@/routes/worker/worker-areas";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { AttentionList } from "./attention-list";
import { ClearanceBand } from "./clearance-band";
import { FileGrid } from "./file-grid";
import { OverviewSkeleton } from "./overview-skeleton";
import { AssignmentPanel, RightNowPanel } from "./right-now";

/**
 * The worker at a glance: whether they are cleared to drive and until when, with every dated
 * requirement on the runway; what needs someone; where they are in their day and how dispatch
 * sees them; and every part of their record with where it stands. It reads the same overview
 * as the page's header, so it costs no second request.
 */
export default function OverviewTab({ workerId, onOpen }: WorkerTabProps) {
  const t = useT();
  const areas = useWorkerAreas();
  const pane = useWorkPane();
  const { data, error, isLoading, refetch } = useQuery(workerOverviewQuery(workerId));

  if (isLoading) return <OverviewSkeleton />;
  if (error || !data) {
    return (
      <ErrorState
        layout="sheet"
        error={error}
        onRetry={() => void refetch()}
        title={t("The overview could not be loaded")}
      />
    );
  }

  const open = (target: WorkerTarget) => pane.closeThen(() => onOpen(target));

  return (
    <div className="@container/overview flex min-w-0 animate-reveal flex-col gap-4">
      <ClearanceBand
        overview={data}
        canOpenCredentials={areas.isDrawable({ tab: "qualification", view: "credentials" })}
        onOpen={open}
      />
      <div className="grid items-start gap-4 @[900px]/overview:grid-cols-[minmax(0,1fr)_24rem]">
        <AttentionList concerns={data.concerns} concernCount={data.concernCount} onOpen={open} />
        <div className="flex min-w-0 flex-col gap-4">
          <RightNowPanel
            overview={data}
            canOpenHours={areas.isDrawable({ tab: "time", view: "hours-of-service" })}
            canOpenTimeOff={areas.isDrawable({ tab: "time", view: "time-off" })}
            onOpen={open}
          />
          <AssignmentPanel overview={data} />
        </div>
      </div>
      <FileGrid overview={data} areas={areas} onOpen={open} />
    </div>
  );
}
