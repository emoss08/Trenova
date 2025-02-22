import { Icon } from "@/components/ui/icons";
import { formatSplitDateTime } from "@/lib/date";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { MoveStatus } from "@/types/move";
import { Stop, StopStatus, StopType } from "@/types/stop";
import {
  faArrowDown,
  faCheck,
  faPlus,
} from "@fortawesome/pro-regular-svg-icons";
import { faCircle, faTruck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { memo, useMemo, useState } from "react";
import { UseFieldArrayRemove, UseFieldArrayUpdate } from "react-hook-form";
import { StopDialog } from "./stop-dialog";

type TimingValue = {
  arrival: { date: string; time: string } | null;
  departure: { date: string; time: string } | null;
};

type TooltipItem = {
  label: string;
  value: string | number | TimingValue;
};

type TooltipSection = {
  title: string;
  items: TooltipItem[];
};

const getStatusIcon = (
  status: StopStatus,
  isLastStop: boolean,
  moveStatus: MoveStatus,
) => {
  if (isLastStop && moveStatus === MoveStatus.Completed) {
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

const getStopTypeBgColor = (type: StopType) => {
  switch (type) {
    case StopType.Pickup:
      return "bg-blue-500/10 border border-blue-500/30";
    case StopType.Delivery:
      return "bg-green-500/10 border border-green-500/30";
    case StopType.SplitPickup:
      return "bg-purple-500/10 border border-purple-500/30";
    case StopType.SplitDelivery:
      return "bg-amber-500/10 border border-amber-500/30";
    default:
      return "bg-gray-500/10 border border-gray-500/30";
  }
};

const getStopTypeLabel = (type: StopType) => {
  switch (type) {
    case StopType.Pickup:
      return "Pickup";
    case StopType.Delivery:
      return "Delivery";
    case StopType.SplitPickup:
      return "Split Pickup";
    case StopType.SplitDelivery:
      return "Split Delivery";
    default:
      return "Unknown";
  }
};

const getStopTypeTextColor = (type: StopType) => {
  switch (type) {
    case StopType.Pickup:
      return "text-blue-500";
    case StopType.Delivery:
      return "text-green-500";
    case StopType.SplitPickup:
      return "text-purple-500";
    case StopType.SplitDelivery:
      return "text-amber-500";
    default:
      return "text-gray-500";
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

export const StopTooltipContent = memo(function StopTooltipContent({
  stop,
}: {
  stop: Stop;
}) {
  const sections: TooltipSection[] = [
    {
      title: "Stop Details",
      items: [
        { label: "Status", value: stop.status },
        { label: "Type", value: getStopTypeLabel(stop.type) },
        {
          label: "Location",
          value: `${stop.location?.addressLine1}, ${stop.location?.city}, ${stop.location?.state?.abbreviation} ${stop.location?.postalCode}`,
        },
      ],
    },
    {
      title: "Cargo Information",
      items: [
        { label: "Pieces", value: stop.pieces?.toString() || "-" },
        { label: "Weight", value: stop.weight ? `${stop.weight} lbs` : "-" },
      ],
    },
    {
      title: "Timing",
      items: [
        {
          label: "Planned Arrival",
          value: `${formatSplitDateTime(stop.plannedArrival).date} ${formatSplitDateTime(stop.plannedArrival).time}`,
        },
        {
          label: "Planned Departure",
          value: `${formatSplitDateTime(stop.plannedDeparture).date} ${formatSplitDateTime(stop.plannedDeparture).time}`,
        },
        {
          label: "Actual Arrival",
          value: stop.actualArrival
            ? `${formatSplitDateTime(stop.actualArrival).date} ${formatSplitDateTime(stop.actualArrival).time}`
            : "-",
        },
        {
          label: "Actual Departure",
          value: stop.actualDeparture
            ? `${formatSplitDateTime(stop.actualDeparture).date} ${formatSplitDateTime(stop.actualDeparture).time}`
            : "-",
        },
      ],
    },
  ];

  const renderValue = (value: TooltipItem["value"]) => {
    return (
      <span className="text-xs font-medium text-right">{String(value)}</span>
    );
  };

  return (
    <div className="w-80 divide-y divide-border">
      {sections.map((section, idx) => (
        <div key={section.title} className={cn("py-2", idx === 0 && "pt-0")}>
          <div className="flex items-center gap-2 mb-2">
            <h3 className="text-xs font-semibold text-foreground">
              {section.title}
            </h3>
          </div>
          <div className="space-y-2">
            {section.items.map((item) => (
              <div
                key={item.label}
                className="flex justify-between items-start gap-4"
              >
                <span className="text-xs text-muted-foreground shrink-0">
                  {item.label}
                </span>
                {renderValue(item.value)}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
});

// Memoize the location display component since it's purely presentational
const LocationDisplay = memo(function LocationDisplay({
  location,
  type,
}: {
  location: Stop["location"];
  type: StopType;
}) {
  return (
    <>
      <div
        className={cn(
          "flex items-center gap-1 text-sm text-primary",
          getStopTypeTextColor(type),
        )}
      >
        <span className="text-xs">{location?.addressLine1}</span>
        <span className="text-2xs">({getStopTypeLabel(type)})</span>
      </div>
      <div className="text-2xs text-muted-foreground">
        {location?.city}, {location?.state?.abbreviation} {location?.postalCode}
      </div>
    </>
  );
});

// Memoize the StopCircle component since it's purely presentational
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
  const bgColor = getBgColor(status);

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

export const StopTimeline = memo(function StopTimeline({
  stop,
  isLast,
  moveStatus,
  moveIdx,
  stopIdx,
  update,
  remove,
}: {
  stop: Stop;
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

  console.log("Is stop the last stop?", {
    isLast,
    stopIdx,
    moveIdx,
  });

  return (
    <>
      <div
        key={stop.id}
        className={cn(
          "relative rounded-lg pt-2 cursor-pointer select-none",
          getStopTypeBgColor(stop.type),
        )}
        onClick={() => setEditModalOpen(!editModalOpen)}
      >
        {!isLast && (
          <div
            className={cn(
              "absolute left-[121px] ml-[2px] top-[20px] bottom-0 w-[2px] z-10",
              lineStyles,
            )}
            style={{ height: "68px" }}
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
              status={stop.status}
              isLast={isLast}
              moveStatus={moveStatus}
            />
          </div>
          <div className="flex-1">
            <LocationDisplay location={stop.location} type={stop.type} />
          </div>
        </div>
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
