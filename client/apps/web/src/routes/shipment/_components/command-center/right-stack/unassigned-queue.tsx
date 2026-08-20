import {
  UnassignedQueueList,
  type UnassignedQueueSummary,
} from "@/components/work-modules/unassigned-queue-list";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useCallback, useState } from "react";
import { useCommandCenterUrl } from "../url-state";
import { ModuleCard } from "./module-card";

const EMPTY_SUMMARY: UnassignedQueueSummary = { totalCount: undefined, pendingRevenue: 0 };

export function UnassignedQueue({ enabled = true }: { enabled?: boolean }) {
  const [, setUrl] = useCommandCenterUrl();
  const [summary, setSummary] = useState<UnassignedQueueSummary>(EMPTY_SUMMARY);

  const handleSelect = useCallback(
    (shipmentId: string) => setUrl({ expanded: shipmentId }),
    [setUrl],
  );

  return (
    <ModuleCard
      id="unassigned"
      title="Unassigned"
      count={summary.totalCount}
      countTone="warning"
      rightSlot={
        <span className="font-table text-muted-foreground hidden text-[9.5px] tabular-nums sm:inline">
          {formatCurrency(summary.pendingRevenue)} waiting
        </span>
      }
    >
      <UnassignedQueueList enabled={enabled} onSelect={handleSelect} onSummary={setSummary} />
    </ModuleCard>
  );
}
