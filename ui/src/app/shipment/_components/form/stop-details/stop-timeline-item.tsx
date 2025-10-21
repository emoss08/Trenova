import { formatSplitDateTime } from "@/lib/date";
import type { MoveSchema } from "@/lib/schemas/move-schema";
import type { StopSchema } from "@/lib/schemas/stop-schema";
import { cn } from "@/lib/utils";
import React from "react";
import { StopTimelineInnerContent } from "./_components/stop-timeline-inner-content";
import { StopTimelineInnerPlaceholder } from "./_components/stop-timeline-inner-placeholder";
import { getLineStyles } from "./stop-utils";

interface StopTimelineItemProps extends React.ComponentProps<"div"> {
  stop: StopSchema;
  nextStopHasInfo: string | number;
  isLast: boolean;
  moveStatus: MoveSchema["status"];
  prevStopStatus?: StopSchema["status"];
  hasErrors: boolean;
  errorMessages?: string[];
}

export function StopTimelineItem({
  stop,
  hasErrors,
  errorMessages = [],
  nextStopHasInfo,
  isLast,
  moveStatus,
  prevStopStatus,
  ...props
}: StopTimelineItemProps) {
  const currentStop = stop;

  const hasStopInfo =
    currentStop.location?.addressLine1 || currentStop.plannedArrival;

  const shouldShowLine = !isLast && hasStopInfo && nextStopHasInfo;

  const lineStyles = getLineStyles(currentStop.status, prevStopStatus);
  const plannedArrival = formatSplitDateTime(currentStop.plannedArrival);

  return (
    <StopTimelineOuter hasErrors={hasErrors} {...props}>
      {hasStopInfo ? (
        <>
          {shouldShowLine && Boolean(lineStyles) ? (
            <div
              className={cn(
                "absolute left-[121px] ml-[2px] top-[20px] bottom-0 w-[2px] z-10",
                lineStyles,
              )}
              style={{ height: "80px" }}
            />
          ) : null}
          <StopTimelineInnerContent
            plannedArrival={plannedArrival}
            currentStop={currentStop}
            isLast={isLast}
            moveStatus={moveStatus}
            hasErrors={hasErrors}
            errorMessages={errorMessages}
            prevStopStatus={prevStopStatus}
          />
        </>
      ) : (
        <StopTimelineInnerPlaceholder
          hasErrors={hasErrors}
          errorMessages={errorMessages}
          currentStop={currentStop}
        />
      )}
    </StopTimelineOuter>
  );
}

interface StopTimelineOuterProps extends React.ComponentProps<"div"> {
  hasErrors?: boolean;
}

function StopTimelineOuter({
  children,
  hasErrors,
  ...props
}: StopTimelineOuterProps) {
  return (
    <div
      {...props}
      className={cn(
        "relative h-[60px] rounded-lg select-none bg-muted pt-2 border border-border group",
        hasErrors && "border-destructive bg-destructive/10",
      )}
    >
      {children}
    </div>
  );
}
