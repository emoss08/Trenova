import { EmptyState } from "@/components/ui/empty-state";
import { ScrollArea, VirtualizedScrollArea } from "@/components/ui/scroll-area";
import { AdditionalCharge } from "@/types/shipment";
import {
  faBoxesStacked,
  faMoneyBill,
  faTruckContainer,
} from "@fortawesome/pro-solid-svg-icons";
import { useVirtualizer } from "@tanstack/react-virtual";
import { memo, useCallback, useMemo, useRef } from "react";
import { MemoizedAdditionalChargeRow } from "./additional-charge-row";

const ROW_HEIGHT = 30;
const OVERSCAN = 5;

const ListHeader = memo(() => (
  <div className="sticky top-0 z-10 grid grid-cols-10 gap-4 p-2 text-sm text-muted-foreground bg-card border-b border-border rounded-t-lg">
    <div className="col-span-4">Accessorial Charge</div>
    <div className="col-span-2 text-left">Unit</div>
    <div className="col-span-2 text-left">Amount</div>
    <div className="col-span-2" />
  </div>
));

ListHeader.displayName = "ListHeader";

// Utility to create a unique key for comparing additional charges
const createChargeKey = (charge: AdditionalCharge): string => {
  return `${charge.accessorialChargeId}-${charge.unit}-${charge.method}-${charge.amount}`;
};

export function AdditionalChargeList({
  additionalCharges,
  handleEdit,
  handleDelete,
}: {
  additionalCharges: AdditionalCharge[];
  handleEdit: (index: number) => void;
  handleDelete: (index: number) => void;
}) {
  const parentRef = useRef<HTMLDivElement>(null);

  // Find duplicate charges by creating a frequency map
  const duplicateIndices = useMemo(() => {
    const chargeFrequency = new Map<string, number[]>();
    const duplicates = new Set<number>();

    additionalCharges.forEach((charge, index) => {
      if (!charge.accessorialCharge) return;

      const key = createChargeKey(charge);
      const indices = chargeFrequency.get(key) || [];

      // If this is the second or later occurrence, mark all occurrences as duplicates
      if (indices.length > 0) {
        indices.forEach((idx) => duplicates.add(idx));
        duplicates.add(index);
      }

      indices.push(index);
      chargeFrequency.set(key, indices);
    });

    return duplicates;
  }, [additionalCharges]);

  const virtualizer = useVirtualizer({
    count: additionalCharges?.length ?? 0,
    getScrollElement: () => parentRef.current,
    estimateSize: useCallback(() => ROW_HEIGHT, []),
    overscan: OVERSCAN,
    enabled: !!additionalCharges?.length,
  });

  return !additionalCharges.length ? (
    <EmptyState
      className="max-h-[200px] p-4 border rounded-lg border-bg-sidebar-border"
      title="No Additional Charges"
      description="Shipment has no associated additional charges"
      icons={[faMoneyBill, faBoxesStacked, faTruckContainer]}
    />
  ) : (
    <div className="rounded-lg border border-bg-sidebar-border bg-card">
      <ListHeader />
      {additionalCharges.length > 20 ? (
        <VirtualizedScrollArea
          ref={parentRef}
          className="flex max-h-40 flex-col"
        >
          <div style={{ height: `${virtualizer.getTotalSize()}px` }}>
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const additionalCharge = additionalCharges[virtualRow.index];
              const isDuplicate = duplicateIndices.has(virtualRow.index);

              return (
                <MemoizedAdditionalChargeRow
                  key={virtualRow.index}
                  additionalCharge={additionalCharge as AdditionalCharge}
                  isLast={virtualRow.index === additionalCharges.length - 1}
                  isDuplicate={isDuplicate}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                  index={virtualRow.index}
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                />
              );
            })}
          </div>
        </VirtualizedScrollArea>
      ) : (
        <ScrollArea className="flex max-h-40 flex-col">
          {additionalCharges.map((additionalCharge, index) => {
            const isDuplicate = duplicateIndices.has(index);

            return (
              <MemoizedAdditionalChargeRow
                key={index}
                additionalCharge={additionalCharge as AdditionalCharge}
                isLast={index === additionalCharges.length - 1}
                isDuplicate={isDuplicate}
                onEdit={handleEdit}
                onDelete={handleDelete}
                index={index}
                style={{ height: ROW_HEIGHT }}
              />
            );
          })}
        </ScrollArea>
      )}
    </div>
  );
}
