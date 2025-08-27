import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ShipmentHoldSchema } from "@/lib/schemas/shipment-hold-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { Package, Receipt, Truck } from "lucide-react";
import * as React from "react";

export function HoldsCell({ holds }: { holds?: ShipmentSchema["holds"] }) {
  const active = (holds ?? []).filter((h) => !h?.releasedAt);
  if (active.length === 0) {
    return <span className="text-muted-foreground">—</span>;
  }

  // Overall top severity among all active holds
  const top = topSeverity(active);

  // Gate blocked flags (OR across active holds)
  const blocks = {
    dispatch: active.some((h) => !!h?.blocksDispatch),
    delivery: active.some((h) => !!h?.blocksDelivery),
    billing: active.some((h) => !!h?.blocksBilling),
  };

  const count = active.length;
  const tooltipItems = active.slice(0, 6).map((h) => {
    const s = h?.severity ?? "Advisory";
    const t = h?.type ?? "Hold";
    const c = h?.reasonCode ?? "";
    return `${t} — ${c} (${s})`.trim();
  });

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="inline-flex items-center gap-1.5 max-w-[150px]">
          <GateIcon
            blocked={blocks.dispatch}
            severity={top}
            label="Dispatch"
            Icon={Truck}
          />
          <GateIcon
            blocked={blocks.delivery}
            severity={top}
            label="Delivery"
            Icon={Package}
          />
          <GateIcon
            blocked={blocks.billing}
            severity={top}
            label="Billing"
            Icon={Receipt}
          />
        </div>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">
        <div className="text-xs">
          <div className="mb-1 font-medium">Active holds ({count})</div>
          <ul className="list-disc pl-4 space-y-0.5">
            {tooltipItems.map((t, i) => (
              <li key={i}>{t}</li>
            ))}
          </ul>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

// Highest severity across a set of holds
function topSeverity(
  holds: ShipmentSchema["holds"],
): ShipmentHoldSchema["severity"] {
  const rank: Record<ShipmentHoldSchema["severity"], number> = {
    Blocking: 3,
    Advisory: 2,
    Informational: 1,
  };
  let best: ShipmentHoldSchema["severity"] = "Informational";
  for (const h of holds ?? []) {
    const s = (h?.severity ?? "Advisory") as ShipmentHoldSchema["severity"];
    if (rank[s] > rank[best]) best = s;
  }
  return best;
}

function GateIcon({
  blocked,
  severity,
  label,
  Icon,
}: {
  blocked: boolean;
  label: string;
  severity: ShipmentHoldSchema["severity"];
  Icon: React.ComponentType<{ className?: string }>;
}) {
  // Color driven by severity; muted if not blocked
  const color = !blocked
    ? "text-muted-foreground"
    : severity === "Blocking"
      ? "text-red-500"
      : severity === "Advisory"
        ? "text-amber-500"
        : "text-sky-500";

  return (
    <span
      className={cn("inline-flex items-center", color)}
      aria-label={`${label} ${blocked ? `blocked (${severity})` : "allowed"}`}
      title={`${label} ${blocked ? `blocked (${severity})` : "allowed"}`}
    >
      <Icon className="h-4 w-4" />
    </span>
  );
}
