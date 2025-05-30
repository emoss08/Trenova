import { type FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import type { UserSchema } from "@/lib/schemas/user-schema";

export type FleetCode = FleetCodeSchema & {
  manager?: UserSchema | null;
};
