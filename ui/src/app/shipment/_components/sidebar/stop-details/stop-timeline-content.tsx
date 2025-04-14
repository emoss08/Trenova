import { StopDialog } from "@/app/shipment/_components/sidebar/stop-details/stop-dialog";
import { Icon } from "@/components/ui/icons";
import { formatSplitDateTime } from "@/lib/date";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { MoveStatus } from "@/types/move";
import { Stop, StopStatus, StopType } from "@/types/stop";
import { useCallback, useState } from "react";
import {
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  useFormContext,
} from "react-hook-form";
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
}: {
  location: Stop["location"];
  type: StopType;
}) {
  return (
    <>
      <div className="flex items-center gap-1 text-sm text-primary">
        <span className="text-xs">{location?.addressLine1}</span>
        <span className="text-2xs">({getStopTypeLabel(type)})</span>
      </div>
      <div className="text-2xs text-muted-foreground">
        {location?.city}, {location?.state?.abbreviation} {location?.postalCode}
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
  prevStopStatus,
}: {
  status: StopStatus;
  isLast: boolean;
  moveStatus: MoveStatus;
  hasErrors?: boolean;
  prevStopStatus?: StopStatus;
}) {
  const stopIcon = getStatusIcon(status, isLast, moveStatus);
  const bgColor = getStopStatusBgColor(status);
  const borderColor = prevStopStatus
    ? getStopStatusBorderColor(prevStopStatus)
    : "";

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
      {hasErrors && (
        <div className="absolute -top-1 -right-1 size-3 rounded-full bg-destructive flex items-center justify-center">
          <span className="text-[8px] font-bold text-red-200">!</span>
        </div>
      )}
    </div>
  );
}

export default function StopTimeline({
  stop,
  nextStop,
  isLast,
  moveStatus,
  moveIdx,
  stopIdx,
  update,
  remove,
  prevStopStatus,
}: {
  stop: Stop;
  nextStop: Stop | null;
  isLast: boolean;
  moveStatus: MoveStatus;
  moveIdx: number;
  stopIdx: number;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
  prevStopStatus?: StopStatus;
}) {
  // Dialog open/close state
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  // Form context for errors
  const {
    formState: { errors },
  } = useFormContext<ShipmentSchema>();

  // Get stop details
  const lineStyles = getLineStyles(stop.status, prevStopStatus);
  const plannedArrival = formatSplitDateTime(stop.plannedArrival);

  // Check for errors
  const stopErrors = errors.moves?.[moveIdx]?.stops?.[stopIdx];
  const hasErrors = stopErrors && Object.keys(stopErrors).length > 0;

  // Check if we have stop info
  const hasStopInfo = stop.location?.addressLine1 || stop.plannedArrival;
  const nextStopHasInfo =
    nextStop?.location?.addressLine1 || nextStop?.plannedArrival;
  const shouldShowLine = !isLast && hasStopInfo && nextStopHasInfo;

  // Handler to open dialog
  const openDialog = useCallback(() => {
    setIsDialogOpen(true);
  }, []);

  // Handle dialog state changes
  const handleDialogChange = useCallback((open: boolean) => {
    setIsDialogOpen(open);
  }, []);

  return (
    <div>
      {/* Clickable stop display */}
      <div
        className={cn(
          "relative h-[60px] rounded-lg cursor-pointer select-none bg-muted/50 pt-2 border border-border",
          hasErrors && "border-destructive bg-destructive/10",
        )}
        onClick={openDialog}
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
                <div className="text-primary text-xs">
                  {plannedArrival.date}
                </div>
                <div className="text-muted-foreground text-2xs">
                  {plannedArrival.time}
                </div>
              </div>
              <div className="relative z-10">
                <StopCircle
                  status={stop.status}
                  isLast={isLast}
                  moveStatus={moveStatus}
                  hasErrors={hasErrors}
                  prevStopStatus={prevStopStatus}
                />
              </div>
              <div className="flex-1">
                <LocationDisplay location={stop.location} type={stop.type} />
              </div>
            </div>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center text-center">
            {hasErrors ? (
              <div className="flex flex-col items-center justify-center">
                <span className="mt-1 text-sm text-red-500">
                  Error in &apos;{getStopTypeLabel(stop.type)}&apos; stop
                </span>
                <p className="text-red-500 text-xs">
                  Please click to edit and fix the errors.
                </p>
              </div>
            ) : (
              <>
                <div className="text-foreground text-sm">
                  Enter {getStopTypeLabel(stop.type)} Information
                </div>
                <p className="text-muted-foreground text-xs">
                  {getStopTypeLabel(stop.type)} information is required to
                  create a shipment.
                </p>
              </>
            )}
          </div>
        )}
      </div>

      {/* The dialog */}
      <StopDialog
        open={isDialogOpen}
        onOpenChange={handleDialogChange}
        isEditing={true}
        moveIdx={moveIdx}
        stopIdx={stopIdx}
        update={update}
        remove={remove}
      />
    </div>
  );
}
