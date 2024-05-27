import { createGlobalStore } from "@/lib/useGlobalStore";
import { Rate } from "@/types/dispatch";

export type ShipmentView = "list" | "calendar" | "map";

type RateStore = {
  currentRate?: Rate;
  currentView: ShipmentView;
  addRateDialogOpen: boolean;
};

export const useRateStore = createGlobalStore<RateStore>({
  currentRate: undefined,
  currentView: "list",
  addRateDialogOpen: false,
});
