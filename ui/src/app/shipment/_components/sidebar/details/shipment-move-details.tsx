import { StopStatusBadge } from "@/components/status-badge";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useShipment } from "@/hooks/use-shipment";
import { formatSplitDateTime } from "@/lib/date";
import { cn } from "@/lib/utils";
import { type ShipmentMove } from "@/types/move";
import { Stop, StopStatus } from "@/types/stop";
import {
  faArrowDown,
  faCheck,
  faPlus,
} from "@fortawesome/pro-regular-svg-icons";
import { faCircle, faTruck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { memo } from "react";

const getStatusIcon = (
  status: StopStatus,
  isLastStop: boolean,
  moveStatus: StopStatus,
) => {
  if (isLastStop && moveStatus === StopStatus.Completed) {
    return faCheck;
  }

  switch (status) {
    case StopStatus.New:
      return faPlus;
    case StopStatus.InTransit:
      return faTruck;
    case StopStatus.Completed:
      return faArrowDown;
    case StopStatus.Canceled:
      return faXmark;
    default:
      return faCircle;
  }
};

const getBgColor = (status: StopStatus) => {
  switch (status) {
    case StopStatus.New:
      return "bg-purple-500";
    case StopStatus.InTransit:
      return "bg-blue-500";
    case StopStatus.Completed:
      return "bg-green-500";
    case StopStatus.Canceled:
      return "bg-red-500";
    default:
      return "bg-gray-500";
  }
};

const getLineStyles = (status: StopStatus) => {
  if (status === StopStatus.InTransit) {
    return cn(
      "bg-[length:2px_8px]",
      "bg-gradient-to-b from-blue-500 from-50% to-transparent to-50%",
      "motion-safe:animate-flow-down",
    );
  }
  return getBgColor(status);
};

export function ShipmentMovesDetails() {
  const { shipment } = useShipment();

  if (!shipment) {
    return null;
  }

  const { moves } = shipment;

  return (
    <TooltipProvider delayDuration={0}>
      <div className="flex flex-col gap-1 py-4">
        <div className="flex items-center gap-1">
          <h3 className="text-sm font-medium">Moves</h3>
          <span className="text-2xs text-muted-foreground">
            ({moves?.length ?? 0})
          </span>
        </div>
        {moves.map((move) => (
          <MoveInformation key={move.id} move={move} />
        ))}
      </div>
    </TooltipProvider>
  );
}

const MoveInformation = memo(function MoveInformation({
  move,
}: {
  move?: ShipmentMove;
}) {
  if (!move) {
    return <p>No move</p>;
  }

  return (
    <div
      className="bg-card rounded-lg border border-bg-sidebar-border p-4"
      key={move.id}
    >
      <MoveStatusBadge move={move} />
      <div className="relative">
        <div className="space-y-6">
          {move.stops.map((stop, index) => {
            const isLastStop = index === move.stops.length - 1;

            return (
              <StopTimeline
                key={stop.id}
                stop={stop}
                isLast={isLastStop}
                moveStatus={move.status}
              />
            );
          })}
        </div>
      </div>
    </div>
  );
});

const MoveStatusBadge = memo(function MoveStatusBadge({
  move,
}: {
  move?: ShipmentMove;
}) {
  if (!move) {
    return <p>No move</p>;
  }

  return (
    <div className="flex justify-between items-center mb-4">
      <StopStatusBadge status={move.status} />
      <span className="text-sm text-muted-foreground">
        {move.stops.length} stops
      </span>
    </div>
  );
});

// New helper function to format the tooltip content
const formatStopTimingInfo = (stop: Stop) => {
  if (!stop.actualArrival || !stop.actualDeparture)
    return <p>Unable to show timing information</p>;

  const arrival = formatSplitDateTime(stop.actualArrival);
  const departure = formatSplitDateTime(stop.actualDeparture);

  return (
    <ul className="grid gap-1 text-xs">
      <li className="grid gap-0.5">
        <span className="text-muted-foreground">Actual Arrival Time:</span>
        <span className="font-medium">
          {arrival.date} {arrival.time}
        </span>
      </li>
      <li className="grid gap-0.5">
        <span className="text-muted-foreground">Actual Departure Time:</span>
        <span className="font-medium">
          {departure.date} {departure.time}
        </span>
      </li>
    </ul>
  );
};

const StopTimeline = memo(function StopTimeline({
  stop,
  isLast,
  moveStatus,
}: {
  stop: Stop;
  isLast: boolean;
  moveStatus: StopStatus;
}) {
  const stopIcon = getStatusIcon(stop.status, isLast, moveStatus);
  const bgColor = getBgColor(stop.status);
  const lineStyles = getLineStyles(stop.status);
  const plannedArrival = formatSplitDateTime(stop.plannedArrival);
  const tooltipContent =
    stop.status === StopStatus.Completed ? formatStopTimingInfo(stop) : null;

  const stopCircle = (
    <div
      className={cn(
        "rounded-full size-6 flex items-center justify-center",
        bgColor,
      )}
    >
      <Icon icon={stopIcon} className="size-3.5 text-white" />
    </div>
  );

  return (
    <div key={stop.id} className="relative">
      {!isLast && (
        <div
          className={cn(
            "absolute left-[121px] ml-[2px] top-[20px] bottom-0 w-[2px]",
            lineStyles,
          )}
          style={{ height: "48px" }}
        />
      )}

      <div className="flex items-start gap-4">
        <div className="w-24 text-right text-sm">
          <div className="text-primary">{plannedArrival.date}</div>
          <div className="text-muted-foreground">{plannedArrival.time}</div>
        </div>
        <div className="relative z-10">
          {tooltipContent ? (
            <Tooltip>
              <TooltipTrigger asChild>{stopCircle}</TooltipTrigger>
              <TooltipContent>{tooltipContent}</TooltipContent>
            </Tooltip>
          ) : (
            stopCircle
          )}
        </div>
        <div className="flex-1">
          <div className="text-sm text-primary">
            {stop.location?.addressLine1}
          </div>
          <div className="text-2xs text-muted-foreground">
            {stop.location?.city}, {stop.location?.state?.abbreviation}{" "}
            {stop.location?.postalCode}
          </div>
        </div>
      </div>
    </div>
  );
});
