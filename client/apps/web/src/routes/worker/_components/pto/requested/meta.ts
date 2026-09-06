import { ptoTypeMeta } from "@trenova/shared/lib/pto";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useMemo } from "react";

export function usePTOTypeMeta(type: WorkerPTO["type"]) {
  return useMemo(() => ptoTypeMeta(type), [type]);
}
