import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { MoveSchema } from "@/lib/schemas/move-schema";
import type { StopSchema } from "@/lib/schemas/stop-schema";
import { cn } from "@/lib/utils";
import {
  getStatusIcon,
  getStopStatusBgColor,
  getStopStatusBorderColor,
} from "../stop-utils";

export function StopCircle({
  status,
  isLast,
  moveStatus,
  hasErrors,
  errorMessages = [],
  prevStopStatus,
}: {
  status: StopSchema["status"];
  isLast: boolean;
  moveStatus: MoveSchema["status"];
  hasErrors?: boolean;
  errorMessages?: string[];
  prevStopStatus?: StopSchema["status"];
}) {
  const stopIcon = getStatusIcon(status, isLast, moveStatus);
  const bgColor = getStopStatusBgColor(status);
  const borderColor = prevStopStatus
    ? getStopStatusBorderColor(prevStopStatus)
    : "";

  return (
    <StopCircleOuter>
      <StopCircleInner
        bgColor={bgColor}
        borderColor={borderColor}
        prevStopStatus={prevStopStatus}
      >
        <Icon icon={stopIcon} className="size-3.5 text-white" />
      </StopCircleInner>
      {hasErrors && errorMessages.length > 0 && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <ErrorIndicator />
            </TooltipTrigger>
            <TooltipContent side="right" className="max-w-xs">
              <div className="space-y-1">
                <p className="font-semibold text-sm">Validation Errors:</p>
                {errorMessages.map((msg, idx) => (
                  <p key={idx} className="text-xs">
                    • {msg}
                  </p>
                ))}
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </StopCircleOuter>
  );
}

function StopCircleOuter({ children }: { children: React.ReactNode }) {
  return <div className="relative">{children}</div>;
}

function StopCircleInner({
  children,
  bgColor,
  borderColor,
  prevStopStatus,
}: {
  children: React.ReactNode;
  bgColor: string;
  borderColor: string;
  prevStopStatus?: StopSchema["status"];
}) {
  return (
    <div
      className={cn(
        "rounded-full size-6 flex items-center justify-center",
        bgColor,
        prevStopStatus && "border-t-2",
        borderColor,
      )}
    >
      {children}
    </div>
  );
}

function ErrorIndicator() {
  return (
    <div className="absolute -top-1 -right-1 size-3 rounded-full bg-destructive flex items-center justify-center">
      <span className="text-[8px] font-bold text-red-200">!</span>
    </div>
  );
}
