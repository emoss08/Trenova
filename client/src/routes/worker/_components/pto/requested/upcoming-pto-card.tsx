import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { initials } from "@/lib/utils";
import type { WorkerPTO } from "@/types/worker";
import { UpcomingPTOContent } from "./upcoming-pto-content";

function UpcomingPTOCardOuter({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="group relative mb-1 overflow-hidden rounded-xl border border-border p-3 transition-colors"
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
        <Avatar className="size-9 bg-muted ring-1 ring-border">
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
