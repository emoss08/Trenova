/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */
import { StopDialog } from "@/app/shipment/_components/sidebar/stop-details/stop-dialog";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { formatSplitDateTime } from "@/lib/date";
import type { MoveSchema } from "@/lib/schemas/move-schema";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { StopSchema } from "@/lib/schemas/stop-schema";
import { cn } from "@/lib/utils";
import { memo, useCallback, useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { useLocationData } from "./queries";
import {
  getLineStyles,
  getStatusIcon,
  getStopStatusBgColor,
  getStopStatusBorderColor,
  getStopTypeLabel,
} from "./stop-utils";

// Display component for location
function LocationDisplay({
  location,
  type,
  locationId,
}: {
  location?: StopSchema["location"] | null;
  type: StopSchema["type"];
  locationId?: string;
}) {
  // If we have a locationId but no location, fetch the location data directly
  const { data: fetchedLocation } = useLocationData(locationId || "");

  // Use fetchedLocation if available, otherwise fallback to the passed location
  const displayLocation = fetchedLocation || location;

  // If we don't have any location data, display the stop type only
  if (!displayLocation) {
    return (
      <div className="text-sm text-primary">
        <span>{getStopTypeLabel(type)}</span>
      </div>
    );
  }

  return (
    <>
      <div className="flex items-center gap-1 text-sm text-primary">
        <span className="text-xs">{displayLocation.addressLine1}</span>
        <span className="text-2xs">({getStopTypeLabel(type)})</span>
      </div>
      <div className="text-2xs text-muted-foreground">
        {displayLocation.city}, {displayLocation.state?.abbreviation}{" "}
        {displayLocation.postalCode}
      </div>
    </>
  );
}

// Status indicator circle
function StopCircle({
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

  const errorIndicator = hasErrors && (
    <div className="absolute -top-1 -right-1 size-3 rounded-full bg-destructive flex items-center justify-center">
      <span className="text-[8px] font-bold text-red-200">!</span>
    </div>
  );

  return (
    <div className="relative">
      <div
        className={cn(
          "rounded-full size-6 flex items-center justify-center",
          bgColor,
          prevStopStatus && "border-t-2",
          borderColor,
        )}
      >
        <Icon icon={stopIcon} className="size-3.5 text-white" />
      </div>
      {hasErrors && errorMessages.length > 0 ? (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>{errorIndicator}</TooltipTrigger>
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
      ) : (
        errorIndicator
      )}
    </div>
  );
}

function StopTimelineComponent({
  stop,
  nextStop,
  isLast,
  moveStatus,
  moveIdx,
  stopIdx,
  prevStopStatus,
}: {
  stop: StopSchema;
  nextStop: StopSchema | null;
  isLast: boolean;
  moveStatus: MoveSchema["status"];
  moveIdx: number;
  stopIdx: number;
  prevStopStatus?: StopSchema["status"];
}) {
  const {
    setValue,
    getValues,
    formState: { errors },
    watch,
  } = useFormContext<ShipmentSchema>();

  const [isDialogOpen, setIsDialogOpen] = useState(false);

  // Watch only the specific stop instead of entire moves array for better performance
  const watchedStop = watch(`moves.${moveIdx}.stops.${stopIdx}`);
  const watchedNextStop = watch(`moves.${moveIdx}.stops.${stopIdx + 1}`);

  // Use watched values if available, otherwise fallback to props
  const currentStop = watchedStop || stop;
  const currentNextStop = watchedNextStop || nextStop;

  // Check for errors - improved error detection
  const stopErrors = errors.moves?.[moveIdx]?.stops?.[stopIdx];
  const hasErrors = !!(stopErrors && Object.keys(stopErrors).length > 0);

  // Get specific error messages for display
  const errorMessages: string[] = hasErrors
    ? Object.entries(stopErrors)
        .map(([field, error]) => {
          if (error && typeof error === "object" && "message" in error) {
            return `${field}: ${error.message}`;
          }
          return null;
        })
        .filter((msg): msg is string => msg !== null)
    : [];

  // Check if we have stop info
  const hasStopInfo =
    currentStop.location?.addressLine1 || currentStop.plannedArrival;

  const hasActualDates =
    currentStop.actualArrival || currentStop.actualDeparture;

  const nextStopHasInfo =
    currentNextStop?.location?.addressLine1 || currentNextStop?.plannedArrival;

  // Handler to open dialog
  const openEditDialog = useCallback(() => {
    setIsDialogOpen(true);
  }, []);

  // Handle dialog state changes
  const handleDialogChange = useCallback((open: boolean) => {
    setIsDialogOpen(open);
  }, []);

  const handleRevert = useCallback(() => {
    // * Check if the stop has actual arrival and departure dates
    const actualArrival = getValues(
      `moves.${moveIdx}.stops.${stopIdx}.actualArrival`,
    );
    const actualDeparture = getValues(
      `moves.${moveIdx}.stops.${stopIdx}.actualDeparture`,
    );

    const hasActualDates = actualArrival || actualDeparture;

    if (!hasActualDates) {
      toast.error(
        "Cannot revert stop with no actual arrival or departure dates",
      );
      return;
    }

    setValue(`moves.${moveIdx}.stops.${stopIdx}.actualArrival`, undefined, {
      shouldDirty: true,
    });
    setValue(`moves.${moveIdx}.stops.${stopIdx}.actualDeparture`, undefined, {
      shouldDirty: true,
    });
  }, [getValues, moveIdx, setValue, stopIdx]);

  return (
    <>
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <div>
            <StopTimelineItem
              stop={currentStop}
              hasErrors={hasErrors}
              errorMessages={errorMessages}
              nextStopHasInfo={nextStopHasInfo}
              isLast={isLast}
              moveStatus={moveStatus}
              prevStopStatus={prevStopStatus}
            />
          </div>
        </ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem onClick={openEditDialog}>
            <StopContextMenuItem
              title="Edit"
              description="Edit the stop information"
            />
          </ContextMenuItem>
          {hasActualDates && (
            <ContextMenuItem onClick={handleRevert}>
              <StopContextMenuItem
                title="Revert"
                description="Revert and clear the stop arrival date and times"
              />
            </ContextMenuItem>
          )}
          {hasActualDates && hasStopInfo && (
            <ContextMenuItem>
              <StopContextMenuItem
                title="Cancel"
                description="Cancel the stop and clear the stop arrival date and times"
              />
            </ContextMenuItem>
          )}
          <ContextMenuItem variant="destructive">
            <StopContextMenuItem
              title="Remove"
              description="Remove the stop from the movement"
            />
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
      {isDialogOpen && (
        <StopDialog
          open={isDialogOpen}
          onOpenChange={handleDialogChange}
          moveIdx={moveIdx}
          stopIdx={stopIdx}
        />
      )}
    </>
  );
}

function StopTimelineItem({
  stop,
  hasErrors,
  errorMessages = [],
  nextStopHasInfo,
  isLast,
  moveStatus,
  prevStopStatus,
}: {
  stop: StopSchema;
  nextStopHasInfo: string | number;
  isLast: boolean;
  moveStatus: MoveSchema["status"];
  prevStopStatus?: StopSchema["status"];
  hasErrors?: boolean;
  errorMessages?: string[];
}) {
  const currentStop = stop;

  // Check if we have stop info
  const hasStopInfo =
    currentStop.location?.addressLine1 || currentStop.plannedArrival;

  const shouldShowLine = !isLast && hasStopInfo && nextStopHasInfo;

  const lineStyles = getLineStyles(currentStop.status, prevStopStatus);
  const plannedArrival = formatSplitDateTime(currentStop.plannedArrival);

  return (
    <div
      className={cn(
        "relative h-[60px] rounded-lg select-none bg-muted pt-2 border border-border group",
        hasErrors && "border-destructive bg-destructive/10",
      )}
    >
      {hasStopInfo ? (
        <>
          {shouldShowLine && (
            <div
              className={cn(
                "absolute left-[121px] ml-[2px] top-[20px] bottom-0 w-[2px] z-10",
                lineStyles,
              )}
              style={{ height: "80px" }}
            />
          )}
          <div className="flex items-start gap-4 py-1">
            <div className="w-24 text-right text-sm">
              <div className="text-primary text-xs">{plannedArrival.date}</div>
              <div className="text-muted-foreground text-2xs">
                {plannedArrival.time}
              </div>
            </div>
            <div className="relative z-10">
              <StopCircle
                status={currentStop.status}
                isLast={isLast}
                moveStatus={moveStatus}
                hasErrors={hasErrors}
                errorMessages={errorMessages}
                prevStopStatus={prevStopStatus}
              />
            </div>
            <div className="flex-1">
              <LocationDisplay
                location={currentStop.location}
                type={currentStop.type}
                locationId={currentStop.locationId}
              />
            </div>
          </div>
        </>
      ) : (
        <div className="flex flex-col items-center justify-center text-center">
          {hasErrors ? (
            <div className="flex flex-col items-center justify-center gap-2">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="flex items-center gap-2 cursor-help">
                      <div className="size-5 rounded-full bg-destructive flex items-center justify-center">
                        <span className="text-xs font-bold text-red-200">
                          !
                        </span>
                      </div>
                      <span className="text-sm text-red-500">
                        Error in {getStopTypeLabel(currentStop.type)} stop
                      </span>
                    </div>
                  </TooltipTrigger>
                  {errorMessages.length > 0 && (
                    <TooltipContent className="max-w-xs">
                      <div className="space-y-1">
                        <p className="font-semibold text-sm">
                          Validation Errors:
                        </p>
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
              <p className="text-red-500 text-xs">
                Click to edit and fix the errors
              </p>
            </div>
          ) : (
            <>
              <div className="text-foreground text-sm">
                Enter {getStopTypeLabel(currentStop.type)} Information
              </div>
              <p className="text-muted-foreground text-xs">
                {getStopTypeLabel(currentStop.type)} information is required to
                create a shipment.
              </p>
            </>
          )}
        </div>
      )}
    </div>
  );
}

function StopContextMenuItem({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-2">
        <span>{title}</span>
      </div>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  );
}

// Export memoized version to prevent unnecessary re-renders
export const StopTimeline = memo(StopTimelineComponent);
