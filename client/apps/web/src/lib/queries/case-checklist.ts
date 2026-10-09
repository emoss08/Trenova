import { apiService } from "@/services/api";
import type { ChecklistKind } from "@/types/case-checklist";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const caseChecklist = createQueryKeys("caseChecklist", {
  list: (kind: ChecklistKind) => ({
    queryKey: [kind],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      apiService.caseChecklistService.list(kind, { signal }),
  }),
});
