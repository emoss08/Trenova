import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { initials } from "@trenova/shared/lib/utils";
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
        <Avatar className="bg-muted ring-border size-9 ring-1">
          <AvatarImage
            src={worker?.profilePicUrl ?? undefined}
            alt={`${worker?.firstName ?? ""} ${worker?.lastName ?? ""}`}
          />
          <AvatarFallback className="text-xs">
            {initials(worker?.firstName, worker?.lastName)}
          </AvatarFallback>
        </Avatar>
        <UpcomingPTOContent pto={workerPTO} />
      </UpcomingPTOCardInner>
    </UpcomingPTOCardOuter>
  );
}
