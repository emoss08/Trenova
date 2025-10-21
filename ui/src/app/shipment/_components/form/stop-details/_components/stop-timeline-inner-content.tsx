import { MoveSchema } from "@/lib/schemas/move-schema";
import { StopSchema } from "@/lib/schemas/stop-schema";
import { LocationDisplay } from "./location-display";
import { StopCircle } from "./stop-circle";
import { StopPlannedTimes } from "./stop-planned-times";

type StopTimelineInnerContentProps = {
  plannedArrival: { date: string; time: string };
  currentStop: StopSchema;
  isLast: boolean;
  moveStatus: MoveSchema["status"];
  hasErrors: boolean;
  errorMessages: string[];
  prevStopStatus?: StopSchema["status"];
};

export function StopTimelineInnerContent({
  plannedArrival,
  currentStop,
  isLast,
  moveStatus,
  hasErrors,
  errorMessages,
  prevStopStatus,
}: StopTimelineInnerContentProps) {
  "use no memo";
  return (
    <StopTimelineInnerContentOuter>
      <StopPlannedTimes plannedArrival={plannedArrival} />
      <StopTimelineInnerContentInner>
        <StopCircle
          status={currentStop.status}
          isLast={isLast}
          moveStatus={moveStatus}
          hasErrors={hasErrors}
          errorMessages={errorMessages}
          prevStopStatus={prevStopStatus}
        />
      </StopTimelineInnerContentInner>
      <LocationDisplay
        location={currentStop.location}
        type={currentStop.type}
      />
    </StopTimelineInnerContentOuter>
  );
}

function StopTimelineInnerContentOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex items-start gap-4 py-1">{children}</div>;
}

function StopTimelineInnerContentInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="relative z-10">{children}</div>;
}
