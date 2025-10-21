import { WorkerPTOSchema } from "@/lib/schemas/worker-schema";
import { useMemo } from "react";

export function usePTOTypeMeta(type: WorkerPTOSchema["type"]) {
  return useMemo(() => {
    switch (type) {
      case "Vacation":
        return {
          label: "Vacation",
          badgeVariant: "purple",
          accentClass: "from-purple-600 to-purple-600/5",
        };
      case "Sick":
        return {
          label: "Sick",
          badgeVariant: "red",
          accentClass: "from-red-600 to-red-600/5",
        };
      case "Holiday":
        return {
          label: "Holiday",
          badgeVariant: "info",
          accentClass: "from-blue-600 to-blue-600/5",
        };
      case "Bereavement":
        return {
          label: "Bereavement",
          badgeVariant: "active",
          accentClass: "from-green-600 to-green-600/5",
        };
      case "Maternity":
        return {
          label: "Maternity",
          badgeVariant: "pink",
          accentClass: "from-pink-600 to-pink-600/5",
        };
      case "Paternity":
        return {
          label: "Paternity",
          badgeVariant: "teal",
          accentClass: "from-teal-600 to-teal-600/5",
        };
      default:
        return {
          label: String(type),
          accentClass: "from-muted-foreground/30 to-transparent",
        };
    }
  }, [type]);
}
