"use no memo";
import { EmptyState } from "@/components/ui/empty-state";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import {
  faBoxesStacked,
  faMoneyBill,
  faTruckContainer,
} from "@fortawesome/pro-solid-svg-icons";
import { useMemo } from "react";
import { AdditionalChargeRow } from "./additional-charge-row";

function AdditionalChargeRowHeader() {
  return (
    <div className="sticky top-0 z-10 grid grid-cols-10 gap-4 p-2 text-sm text-muted-foreground bg-card border-b border-border rounded-t-lg">
      <div className="col-span-4">Accessorial Charge</div>
      <div className="col-span-2 text-left">Unit</div>
      <div className="col-span-2 text-left">Amount</div>
      <div className="col-span-2" />
    </div>
  );
}

export function AdditionalChargeList({
  additionalCharges,
  handleEdit,
  handleDelete,
}: {
  additionalCharges: ShipmentSchema["additionalCharges"];
  handleEdit: (index: number) => void;
  handleDelete: (index: number) => void;
}) {
  const duplicateIndices = useMemo(() => {
    const chargeFrequency = new Map<string, number[]>();
    const duplicates = new Set<number>();

    additionalCharges?.forEach((charge, index) => {
      if (!charge.accessorialCharge) return;

      const key = `${charge.accessorialChargeId}-${charge.unit}-${charge.method}-${charge.amount}`;
      const indices = chargeFrequency.get(key) || [];

      if (indices.length > 0) {
        indices.forEach((idx) => duplicates.add(idx));
        duplicates.add(index);
      }

      indices.push(index);
      chargeFrequency.set(key, indices);
    });

    return duplicates;
  }, [additionalCharges]);

  if (!additionalCharges) return null;

  return !additionalCharges.length ? (
    <EmptyState
      className="max-h-[200px] p-4 border rounded-lg border-bg-sidebar-border"
      title="No Additional Charges"
      description="Shipment has no associated additional charges"
      icons={[faMoneyBill, faBoxesStacked, faTruckContainer]}
    />
  ) : (
    <AdditionalChargeListInner>
      <AdditionalChargeRowHeader />
      <AdditionalChargeListScrollArea>
        {additionalCharges.map((additionalCharge, index) => {
          const isDuplicate = duplicateIndices.has(index);
          return (
            <AdditionalChargeRow
              key={index}
              additionalCharge={additionalCharge}
              isDuplicate={isDuplicate}
              onEdit={handleEdit}
              onDelete={handleDelete}
              index={index}
            />
          );
        })}
      </AdditionalChargeListScrollArea>
    </AdditionalChargeListInner>
  );
}

export function AdditionalChargeListScrollArea({
  children,
}: {
  children: React.ReactNode;
}) {
  return <ScrollArea className="flex max-h-40 flex-col">{children}</ScrollArea>;
}

function AdditionalChargeListInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-bg-sidebar-border bg-card">
      {children}
    </div>
  );
}
