import { type EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { type EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { type FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { type TractorSchema } from "@/lib/schemas/tractor-schema";
import { type UsStateSchema } from "@/lib/schemas/us-state-schema";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";

export enum EquipmentStatus {
  Available = "Available",
  OutOfService = "Out of Service",
  AtMaintenance = "At Maintenance",
  Sold = "Sold",
}

export type Tractor = {
  fleetCode?: FleetCodeSchema | null;
  usState?: UsStateSchema | null;
  equipmentType?: EquipmentTypeSchema | null;
  equipmentManufacturer?: EquipmentManufacturerSchema | null;
  primaryWorker?: WorkerSchema | null;
  secondaryWorker?: WorkerSchema | null;
} & TractorSchema;
