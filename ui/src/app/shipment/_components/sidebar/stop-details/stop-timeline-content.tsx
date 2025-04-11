import { Icon } from "@/components/ui/icons";
import { formatSplitDateTime } from "@/lib/date";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { MoveStatus } from "@/types/move";
import { Stop, StopStatus, StopType } from "@/types/stop";
import { memo, useMemo, useState } from "react";
import { UseFieldArrayRemove, UseFieldArrayUpdate } from "react-hook-form";
import { StopDialog } from "./stop-dialog";
import {
  getLineStyles,
  getStatusIcon,
  getStopStatusBgColor,
  getStopTypeLabel,
} from "./stop-utils";

const LocationDisplay = memo(function LocationDisplay({
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
});

const StopCircle = memo(function StopCircle({
  status,
  isLast,
  moveStatus,
}: {
  status: StopStatus;
  isLast: boolean;
  moveStatus: MoveStatus;
}) {
  const stopIcon = getStatusIcon(status, isLast, moveStatus);
  const bgColor = getStopStatusBgColor(status);

  return (
    <div
      className={cn(
        "rounded-full size-6 flex items-center justify-center",
        bgColor,
      )}
    >
      <Icon icon={stopIcon} className="size-3.5 text-white" />
    </div>
  );
});

const StopTimeline = memo(function StopTimeline({
  stop,
  nextStop,
  isLast,
  moveStatus,
  moveIdx,
  stopIdx,
  update,
  remove,
}: {
  stop: Stop;
  nextStop: Stop | null;
  isLast: boolean;
  moveStatus: MoveStatus;
  moveIdx: number;
  stopIdx: number;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
}) {
  const [editModalOpen, setEditModalOpen] = useState<boolean>(false);
  const lineStyles = useMemo(() => getLineStyles(stop.status), [stop.status]);
  const plannedArrival = useMemo(
    () => formatSplitDateTime(stop.plannedArrival),
    [stop.plannedArrival],
  );

  const hasStopInfo = stop.location?.addressLine1 || stop.plannedArrival;
  const nextStopHasInfo =
    nextStop?.location?.addressLine1 || nextStop?.plannedArrival;
  const shouldShowLine = !isLast && hasStopInfo && nextStopHasInfo;

  return (
    <>
      <div
        key={stop.id}
        className="relative h-[60px] rounded-lg cursor-pointer select-none bg-muted/50 pt-2 border border-border"
        onClick={() => setEditModalOpen(!editModalOpen)}
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
                />
              </div>
              <div className="flex-1">
                <LocationDisplay location={stop.location} type={stop.type} />
              </div>
            </div>
          </>
        ) : (
          <div className="flex flex-col items-center justify-center text-center">
            <div className="text-foreground text-sm">
              Enter {getStopTypeLabel(stop.type)} Information
            </div>
            <p className="text-muted-foreground text-xs">
              {getStopTypeLabel(stop.type)} information is required to create a
              shipment.
            </p>
          </div>
        )}
      </div>

      <StopDialog
        open={editModalOpen}
        onOpenChange={setEditModalOpen}
        stopId={stop.id ?? ""}
        isEditing={true}
        moveIdx={moveIdx}
        stopIdx={stopIdx}
        update={update}
        remove={remove}
      />
    </>
  );
});

export default StopTimeline;
