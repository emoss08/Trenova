import { createGlobalStore } from "@/hooks/use-global-store";
import type { WorkerSyncLogLine, WorkerSyncSummary } from "@/types/samsara";

interface SamsaraSyncState {
  lastSuccessfulSync: WorkerSyncSummary | null;
  logLines: WorkerSyncLogLine[];
}

export const useSamsaraSyncStore = createGlobalStore<SamsaraSyncState>({
  lastSuccessfulSync: null,
  logLines: [],
});
