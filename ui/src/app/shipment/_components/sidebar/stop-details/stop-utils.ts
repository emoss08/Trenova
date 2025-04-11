import { cn } from "@/lib/utils";
import { MoveStatus } from "@/types/move";
import { StopStatus, StopType } from "@/types/stop";
import {
  faArrowDown,
  faCheck,
  faPlus,
} from "@fortawesome/pro-regular-svg-icons";
import { faCircle, faTruck, faXmark } from "@fortawesome/pro-solid-svg-icons";

export const getStatusIcon = (
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

export const getStopTypeLabel = (type: StopType) => {
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

export const getStopStatusBgColor = (status: StopStatus) => {
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

export const getLineStyles = (status: StopStatus) => {
  if (status === StopStatus.InTransit) {
    return cn(
      "bg-[length:2px_8px]",
      "bg-gradient-to-b from-blue-500 from-50% to-transparent to-50%",
      "motion-safe:animate-flow-down",
    );
  }
  return getStopStatusBgColor(status);
};
