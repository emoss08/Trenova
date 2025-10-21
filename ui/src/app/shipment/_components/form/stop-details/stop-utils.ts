import { MoveStatus, type MoveSchema } from "@/lib/schemas/move-schema";
import {
  StopStatus,
  StopType,
  type StopSchema,
} from "@/lib/schemas/stop-schema";
import { cn } from "@/lib/utils";
import {
  faArrowDown,
  faCheck,
  faPlus,
} from "@fortawesome/pro-regular-svg-icons";
import { faCircle, faTruck, faXmark } from "@fortawesome/pro-solid-svg-icons";

export const getStatusIcon = (
  status: StopSchema["status"],
  isLastStop: boolean,
  moveStatus: MoveSchema["status"],
) => {
  if (isLastStop && moveStatus === MoveStatus.enum.Completed) {
    return faCheck;
  }

  switch (status) {
    case StopStatus.enum.New:
      return faPlus;
    case StopStatus.enum.InTransit:
      return faTruck;
    case StopStatus.enum.Completed:
      return faArrowDown;
    case StopStatus.enum.Canceled:
      return faXmark;
    default:
      return faCircle;
  }
};

export const getStopTypeLabel = (type: StopSchema["type"]) => {
  switch (type) {
    case StopType.enum.Pickup:
      return "Pickup";
    case StopType.enum.Delivery:
      return "Delivery";
    case StopType.enum.SplitPickup:
      return "Split Pickup";
    case StopType.enum.SplitDelivery:
      return "Split Delivery";
    default:
      return "Unknown";
  }
};

export const getStopStatusBgColor = (status: StopSchema["status"]) => {
  switch (status) {
    case StopStatus.enum.New:
      return "bg-purple-500";
    case StopStatus.enum.InTransit:
      return "bg-blue-500";
    case StopStatus.enum.Completed:
      return "bg-green-500";
    case StopStatus.enum.Canceled:
      return "bg-red-500";
    default:
      return "bg-gray-500";
  }
};

export const getStopStatusBorderColor = (status: StopSchema["status"]) => {
  switch (status) {
    case StopStatus.enum.New:
      return "border-purple-500";
    case StopStatus.enum.InTransit:
      return "border-blue-500";
    case StopStatus.enum.Completed:
      return "border-green-500";
    case StopStatus.enum.Canceled:
      return "border-red-500";
    default:
      return "border-gray-500";
  }
};

export const getLineStyles = (
  status: StopSchema["status"],
  prevStatus?: StopSchema["status"],
) => {
  if (status === StopStatus.enum.InTransit) {
    return cn(
      "bg-[length:2px_8px]",
      "bg-gradient-to-b from-blue-500 from-50% to-transparent to-50%",
      "motion-safe:animate-flow-down",
    );
  }

  if (prevStatus && prevStatus !== status) {
    const fromColor = getStopStatusBgColor(prevStatus).replace("bg-", "from-");
    const toColor = getStopStatusBgColor(status).replace("bg-", "to-");

    return cn(
      "bg-gradient-to-b",
      fromColor,
      "from-30%",
      toColor,
      "to-70%",
      "opacity-80",
      "rounded-[1px]",
      "shadow-sm",
      "w-[3px] ml-[-.5px]",
    );
  }

  return getStopStatusBgColor(status);
};
