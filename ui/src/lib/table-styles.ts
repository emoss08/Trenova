import { ShipmentStatus } from "@/types/shipment";
import { cn } from "./utils";

export function getShipmentStatusRowClassName(status: ShipmentStatus): string {
  switch (status) {
    case ShipmentStatus.Delayed:
      return cn(
        "bg-orange-600/10 hover:bg-orange-600/20 data-[state=selected]:bg-orange-600/30 focus-visible:bg-orange-600/20",
        "dark:bg-orange-600/20 dark:hover:bg-orange-600/30 dark:data-[state=selected]:bg-orange-600/30 dark:focus-visible:bg-orange-600/20",
        "outline-orange-600 border-orange-600 [&_td]:md:border-orange-600",
      );
    case ShipmentStatus.Canceled:
      return cn(
        "bg-red-600/10 hover:bg-red-600/20 data-[state=selected]:bg-red-600/30 focus-visible:bg-red-600/20",
        "dark:bg-red-600/20 dark:hover:bg-red-600/30 dark:data-[state=selected]:bg-red-600/30 dark:focus-visible:bg-red-600/20",
        "outline-red-600 border-red-600 [&_td]:md:border-red-600",
      );
    default:
      return "";
  }
}
