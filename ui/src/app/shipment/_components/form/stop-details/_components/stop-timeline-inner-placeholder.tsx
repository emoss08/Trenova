import { StopSchema } from "@/lib/schemas/stop-schema";
import { getStopTypeLabel } from "../stop-utils";
import { StopTimelineError } from "./stop-timeline-error";

type StopTimelineInnerPlaceholderProps = {
  hasErrors: boolean;
  errorMessages: string[];
  currentStop: StopSchema;
};

export function StopTimelineInnerPlaceholder({
  hasErrors,
  errorMessages,
  currentStop,
}: StopTimelineInnerPlaceholderProps) {
  "use no memo";
  return (
    <StopTimelineInnerPlaceholderOuter>
      {hasErrors ? (
        <StopTimelineError
          currentStop={currentStop}
          errorMessages={errorMessages}
        />
      ) : (
        <StopTimelineInnerPlaceholderContent currentStop={currentStop} />
      )}
    </StopTimelineInnerPlaceholderOuter>
  );
}

type StopTimelineInnerPlaceholderContentProps = {
  currentStop: StopSchema;
};

function StopTimelineInnerPlaceholderContent({
  currentStop,
}: StopTimelineInnerPlaceholderContentProps) {
  return (
    <StopTimelineInnerPlaceholderContentOuter>
      <div className="text-foreground text-sm">
        Enter {getStopTypeLabel(currentStop.type)} Information
      </div>
      <p className="text-muted-foreground text-xs">
        {getStopTypeLabel(currentStop.type)} information is required to create a
        shipment.
      </p>
    </StopTimelineInnerPlaceholderContentOuter>
  );
}

function StopTimelineInnerPlaceholderContentOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center text-center">
      {children}
    </div>
  );
}

function StopTimelineInnerPlaceholderOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center text-center">
      {children}
    </div>
  );
}
