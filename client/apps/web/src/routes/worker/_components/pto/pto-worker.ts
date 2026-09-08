import { initials } from "@trenova/shared/lib/utils";
import type { WorkerPTO } from "@trenova/shared/types/worker";

type PTOWorkerSource = Pick<WorkerPTO, "worker">;

export function ptoWorkerName(pto: PTOWorkerSource): string {
  const first = pto.worker?.firstName ?? "";
  const last = pto.worker?.lastName ?? "";
  return `${first} ${last}`.trim() || "Unknown worker";
}

export function ptoWorkerInitials(pto: PTOWorkerSource): string {
  return initials(pto.worker?.firstName, pto.worker?.lastName);
}
