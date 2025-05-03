import { ShipmentStatus } from "@/types/shipment";
import { cn } from "./utils";

export function getShipmentStatusRowClassName(status: ShipmentStatus): string {
  switch (status) {
    case ShipmentStatus.Delayed:
      return cn(
        "bg-orange-700/10 hover:bg-orange-700/20 data-[state=selected]:bg-orange-700/30 focus-visible:bg-orange-700/20",
        "dark:bg-orange-700/20 dark:hover:bg-orange-700/30 dark:data-[state=selected]:bg-orange-700/30 dark:focus-visible:bg-orange-700/20",
      );
    default:
      return "";
  }
}
