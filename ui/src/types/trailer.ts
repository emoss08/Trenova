import { EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { type TrailerSchema } from "@/lib/schemas/trailer-schema";
import { UsStateSchema } from "@/lib/schemas/us-state-schema";

export type Trailer = TrailerSchema & {
  equipmentType?: EquipmentTypeSchema | null;
  equipmentManufacturer?: EquipmentManufacturerSchema | null;
  fleetCode?: FleetCodeSchema | null;
  registrationState?: UsStateSchema | null;
};
