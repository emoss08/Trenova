import { type AssignmentSchema } from "@/lib/schemas/assignment-schema";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { type Tractor } from "./tractor";
import { type Trailer } from "./trailer";

export enum AssignmentStatus {
  New = "New",
  InProgress = "InProgress",
  Completed = "Completed",
  Canceled = "Canceled",
}

export type Assignment = AssignmentSchema & {
  tractor?: Tractor | null;
  trailer?: Trailer | null;
  primaryWorker?: WorkerSchema | null;
  secondaryWorker?: WorkerSchema | null;
};
