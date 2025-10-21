import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { StopSchema } from "@/lib/schemas/stop-schema";
import { getStopTypeLabel } from "../stop-utils";

export function StopTimelineError({
  currentStop,
  errorMessages,
}: {
  currentStop: StopSchema;
  errorMessages: string[];
}) {
  return (
    <StopTimelineErrorOuter>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <StopTimelineErrorTrigger currentStop={currentStop} />
          </TooltipTrigger>
          {errorMessages.length > 0 && (
            <TooltipContent className="max-w-xs">
              <div className="space-y-1">
                <p className="font-semibold text-sm">Validation Errors:</p>
                {errorMessages.map((msg, idx) => (
                  <p key={idx} className="text-xs">
                    • {msg}
                  </p>
                ))}
              </div>
            </TooltipContent>
          )}
        </Tooltip>
      </TooltipProvider>
      <p className="text-red-500 text-xs">Click to edit and fix the errors</p>
    </StopTimelineErrorOuter>
  );
}

interface StopTimelineErrorTriggerProps extends React.ComponentProps<"div"> {
  currentStop: StopSchema;
}

function StopTimelineErrorTrigger({
  currentStop,
  ...props
}: StopTimelineErrorTriggerProps) {
  return (
    <div className="flex items-center gap-2 cursor-help" {...props}>
      <div className="size-5 rounded-full bg-destructive flex items-center justify-center">
        <span className="text-xs font-bold text-red-200">!</span>
      </div>
      <span className="text-sm text-red-500">
        Error in {getStopTypeLabel(currentStop.type)} stop
      </span>
    </div>
  );
}

function StopTimelineErrorOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2">
      {children}
    </div>
  );
}
