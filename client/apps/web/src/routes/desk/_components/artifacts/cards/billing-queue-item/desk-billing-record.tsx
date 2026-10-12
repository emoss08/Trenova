import type { InPlaceRecordProps, RecordArtifactProps } from "../registry";
import { BillingItemSkeleton } from "./billing-item-skeleton";
import { DeskBillingItem } from "./desk-billing-item";
import { useDeskBillingStore } from "./desk-billing-store";

/**
 * A queue item opened from a row of the queue's table, in place of the table.
 * It steps through the queue in the queue's own order, as the server keeps it,
 * so the table's `order` is not needed.
 */
export function DeskBillingRecord({ recordId, onSelect, onBack, backLabel }: InPlaceRecordProps) {
  return (
    <DeskBillingItem
      itemId={recordId}
      fallback={<BillingItemSkeleton stepper />}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The billing item artifact: one card showing whichever queue item is picked,
 * the one it was opened on until a row or a step picks another.
 */
export function DeskBillingArtifact({ artifact, recordId, fallback }: RecordArtifactProps) {
  const lineageId = artifact.lineageId || artifact.id;
  const selected = useDeskBillingStore((state) => state.itemByLineage[lineageId]);
  const select = useDeskBillingStore((state) => state.select);

  return (
    <DeskBillingItem
      itemId={selected ?? recordId}
      fallback={fallback}
      onSelect={(itemId) => select(lineageId, itemId)}
    />
  );
}
