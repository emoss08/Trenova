import { cn } from "@trenova/shared/lib/utils";
import { isActiveCarrierAssignment } from "@trenova/shared/types/shipment";
import type { Shipment } from "@trenova/shared/types/shipment";
import { Building2Icon, TriangleAlertIcon } from "lucide-react";

function formatDriverName(first: string | undefined | null, last: string | undefined | null) {
  const firstInitial = first?.[0] ? `${first[0].toUpperCase()}.` : "";
  const lastName = last ?? "";
  return [firstInitial, lastName].filter(Boolean).join(" ").trim();
}

export function DriverCell({ shipment }: { shipment: Shipment }) {
  const move = shipment.moves?.[0] ?? null;
  const assignment = move?.assignment ?? null;
  const driver = assignment?.primaryWorker ?? null;
  const tractor = assignment?.tractor ?? null;
  const trailer = assignment?.trailer ?? null;
  const carrierAssignment =
    move?.coverageType === "carrier" && isActiveCarrierAssignment(move?.carrierAssignment)
      ? move.carrierAssignment
      : null;

  if (carrierAssignment) {
    const carrierName = carrierAssignment.carrier?.name || "External carrier";
    const carrierLine = [carrierAssignment.carrier?.scac, carrierAssignment.externalDriverName]
      .filter(Boolean)
      .join(" · ");

    return (
      <div className="flex flex-col gap-0.5">
        <span className="inline-flex min-w-0 items-center gap-1 text-[11.5px] font-medium">
          <Building2Icon className="size-3 shrink-0 text-muted-foreground" aria-hidden />
          <span className="truncate">{carrierName}</span>
        </span>
        {carrierLine && (
          <span className="truncate font-table text-[9.5px] text-muted-foreground tabular-nums">
            {carrierLine}
          </span>
        )}
      </div>
    );
  }

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
