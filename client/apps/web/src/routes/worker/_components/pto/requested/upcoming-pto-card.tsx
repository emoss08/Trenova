import { WorkerAvatar } from "@trenova/shared/components/worker-avatar";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { UpcomingPTOContent } from "./upcoming-pto-content";

function UpcomingPTOCardOuter({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="group border-border relative mb-1 overflow-hidden rounded-xl border p-3 transition-colors"
      role="article"
    >
      {children}
    </div>
  );
}

function UpcomingPTOCardInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-3">{children}</div>;
}

export function UpcomingPTOCard({ workerPTO }: { workerPTO: WorkerPTO }) {
  const { worker } = workerPTO;

  return (
    <UpcomingPTOCardOuter>
      <UpcomingPTOCardInner>
        <WorkerAvatar
          firstName={worker?.firstName}
          lastName={worker?.lastName}
          photo={worker?.photo}
          className="size-9"
          fallbackClassName="text-xs"
        />
        <UpcomingPTOContent pto={workerPTO} />
      </UpcomingPTOCardInner>
    </UpcomingPTOCardOuter>
  );
}
