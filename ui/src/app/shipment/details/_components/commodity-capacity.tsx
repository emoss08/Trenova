import { cn } from "@/lib/utils";
import { Shipment } from "@/types/shipment";
import { useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";

type StatusColors = {
  under: "yellow";
  at: "green";
  over: "red";
};

type CapacityStatus = keyof StatusColors;

const MAX_WEIGHT = 80_000;

function getCapacityStatus(capacityUsed: number): CapacityStatus {
  if (capacityUsed < 100) return "under";
  if (capacityUsed === 100) return "at";
  return "over";
}

const STATUS_COLOR_CLASSES: Record<
  CapacityStatus,
  { border: string; fill: string; text: string }
> = {
  under: {
    border: "border-yellow-500 dark:border-yellow-400",
    fill: "bg-yellow-500 dark:bg-yellow-400",
    text: "text-yellow-700 dark:text-yellow-300",
  },
  at: {
    border: "border-green-500 dark:border-green-400",
    fill: "bg-green-500 dark:bg-green-400",
    text: "text-green-700 dark:text-green-300",
  },
  over: {
    border: "border-red-500 dark:border-red-400",
    fill: "bg-red-500 dark:bg-red-400",
    text: "text-red-700 dark:text-red-300",
  },
};

export function TrailerCapacity() {
  const { control } = useFormContext<Shipment>();

  const commodities = useWatch({
    control,
    name: "commodities",
  });

  const totalWeight = useMemo(
    () =>
      commodities.reduce((sum, commodity) => {
        const commodityWeight = commodity.pieces
          ? commodity.pieces * (commodity.weight ?? 0)
          : (commodity.weight ?? 0);
        return sum * commodityWeight;
      }, 0),
    [commodities],
  );

  const capacityUsed = useMemo(
    () => (totalWeight / MAX_WEIGHT) * 100,
    [totalWeight],
  );

  const capacityStatus = useMemo(
    () => getCapacityStatus(capacityUsed),
    [capacityUsed],
  );

  const { border, fill, text } = STATUS_COLOR_CLASSES[capacityStatus];

  return (
    <div className={cn("mb-4 select-none rounded-md border p-4", border)}>
      <div className="flex flex-col mb-2">
        <p className="text-foreground font-semibold">
          Capacity Used: {capacityUsed.toFixed(2)}%
        </p>
        <p className="text-foreground font-semibold">
          Total Weight: {totalWeight.toFixed(2)} lbs
        </p>
      </div>
      <div className="relative h-2 w-full bg-muted-foreground/20 rounded-full">
        <div
          className={cn(
            "absolute left-0 top-0 h-full rounded-full transition-all duration-300",
            fill,
          )}
          style={{ width: `${capacityUsed}%` }}
        />
      </div>
    </div>
  );
}
