import { cn } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { TriangleAlertIcon } from "lucide-react";

function formatDriverName(first: string | undefined | null, last: string | undefined | null) {
  const firstInitial = first?.[0] ? `${first[0].toUpperCase()}.` : "";
  const lastName = last ?? "";
  return [firstInitial, lastName].filter(Boolean).join(" ").trim();
}

export function DriverCell({ shipment }: { shipment: Shipment }) {
  const assignment = shipment.moves?.[0]?.assignment ?? null;
  const driver = assignment?.primaryWorker ?? null;
  const tractor = assignment?.tractor ?? null;
  const trailer = assignment?.trailer ?? null;

  if (!driver) {
    return (
      <div className={cn("inline-flex items-center gap-1 text-[11px] font-medium text-warning")}>
        <TriangleAlertIcon className="size-3" />
        <span>Needs driver</span>
      </div>
    );
  }

  const name = formatDriverName(driver.firstName, driver.lastName) || "—";
  const equipmentLine = [tractor?.code, trailer?.code].filter(Boolean).join(" · ");

  return (
    <div className="flex flex-col gap-0.5">
      <span className="truncate text-[11.5px] font-medium">{name}</span>
      {equipmentLine && (
        <span className="truncate font-table text-[9.5px] text-muted-foreground tabular-nums">
          {equipmentLine}
        </span>
      )}
    </div>
  );
}
