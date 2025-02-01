import { type MoveSchema } from "@/lib/schemas/move-schema";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { type Stop } from "./stop";
import { type Tractor } from "./tractor";
import { type Trailer } from "./trailer";

export type ShipmentMove = MoveSchema & {
  primaryWorker: WorkerSchema;
  secondaryWorker?: WorkerSchema | null;
  trailer?: Trailer | null;
  tractor?: Tractor | null;
  stops: Stop[];
};
