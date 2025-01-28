import { type FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { type User } from "./user";

export type FleetCode = FleetCodeSchema & {
  manager?: User | null;
};
