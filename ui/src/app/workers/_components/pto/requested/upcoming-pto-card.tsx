import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import { usePTOTypeMeta } from ".";
import { UpcomingPTOContent } from "./upcoming-pto-content";

const initials = (first?: string, last?: string) =>
  `${(first?.[0] ?? "").toUpperCase()}${(last?.[0] ?? "").toUpperCase()}`.trim() ||
  "•";

function UpcomingPTOCardOutter({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="group relative overflow-hidden rounded-xl border border-border p-3 transition-colors mb-1"
      role="article"
    >
      {children}
    </div>
  );
}

function UpcomingPTOCardInner({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-3">{children}</div>;
}

function AccentBar({ accentClass }: { accentClass: string }) {
  return (
    <div
      className={cn(
        "pointer-events-none absolute inset-y-0 left-0 w-[3px] bg-gradient-to-b",
        accentClass,
      )}
      aria-hidden
    />
  );
}

export function UpcomingPTOCard({ workerPTO }: { workerPTO: WorkerPTOSchema }) {
  const { worker, type } = workerPTO;
  const { accentClass } = usePTOTypeMeta(type);

  return (
    <UpcomingPTOCardOutter>
      <AccentBar accentClass={accentClass} />
      <UpcomingPTOCardInner>
        <Avatar className="h-9 w-9 ring-1 ring-border">
          <AvatarImage
            src={worker?.profilePictureUrl ?? undefined}
            alt={`${worker?.firstName ?? ""} ${worker?.lastName ?? ""}`}
          />
          <AvatarFallback className="text-xs">
            {initials(worker?.firstName, worker?.lastName)}
          </AvatarFallback>
        </Avatar>
        <UpcomingPTOContent pto={workerPTO} />
      </UpcomingPTOCardInner>
    </UpcomingPTOCardOutter>
  );
}
