import { type LocationSchema } from "@/lib/schemas/location-schema";
import { type StopSchema } from "@/lib/schemas/stop-schema";

export enum StopStatus {
  New = "New",
  InTransit = "InTransit",
  Completed = "Completed",
  Canceled = "Canceled",
}

export enum StopType {
  Pickup = "Pickup",
  Delivery = "Delivery",
  SplitPickup = "SplitPickup",
  SplitDelivery = "SplitDelivery",
}

export type Stop = StopSchema & {
  location?: LocationSchema | null;
};
