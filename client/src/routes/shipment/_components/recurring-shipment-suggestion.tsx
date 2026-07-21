import { Button } from "@/components/ui/button";
import { usePermissions } from "@/hooks/use-permission";
import { describeCron } from "@/lib/cron";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { Resource } from "@/types/permission";
import type { RecurringShipment } from "@/types/recurring-shipment";
import type { Shipment } from "@/types/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CalendarSyncIcon, SparklesIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";

const MATCH_DEBOUNCE_MS = 600;

type LaneKey = {
  customerId: string;
  originLocationId: string;
  destinationLocationId: string;
};

function laneKeyString(key: LaneKey | null): string {
  if (!key) return "";
  return `${key.customerId}|${key.originLocationId}|${key.destinationLocationId}`;
}

function useDebouncedLaneKey(): LaneKey | null {
  const { control } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" });
  const customerId = useWatch({ control, name: "customerId" });
  const moves = useWatch({ control, name: "moves" });

  const laneKey = useMemo<LaneKey | null>(() => {
    if (shipmentId || !customerId) return null;

    let originLocationId = "";
    let destinationLocationId = "";
    for (const move of moves ?? []) {
      for (const stop of move?.stops ?? []) {
        if (!stop?.locationId) continue;
        if (!originLocationId && (stop.type === "Pickup" || stop.type === "SplitPickup")) {
          originLocationId = stop.locationId;
        }
        if (stop.type === "Delivery" || stop.type === "SplitDelivery") {
          destinationLocationId = stop.locationId;
        }
      }
    }

    if (!originLocationId || !destinationLocationId) return null;

    return { customerId, originLocationId, destinationLocationId };
  }, [shipmentId, customerId, moves]);

  const [debouncedKey, setDebouncedKey] = useState<LaneKey | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKey(laneKey), MATCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [laneKey]);

  return debouncedKey;
}

function MatchBanner({ series, onDismiss }: { series: RecurringShipment; onDismiss: () => void }) {
  const queryClient = useQueryClient();
  const [generating, setGenerating] = useState(false);

  const handleGenerate = async () => {
    setGenerating(true);
    try {
      const result = await apiService.recurringShipmentService.generate(series.id as string);
      toast.success(
        result.shipment?.proNumber
          ? `Shipment ${result.shipment.proNumber} generated from "${series.name}"`
          : `Occurrence processed for "${series.name}"`,
        {
          description:
            "The recurring series created this shipment for you — you can discard this manual entry.",
        },
      );
      await queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
      onDismiss();
    } catch {
      toast.error("Failed to generate from the recurring shipment");
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div className="flex items-start gap-3 rounded-lg border border-blue-600/30 bg-blue-600/5 p-3">
      <CalendarSyncIcon className="mt-0.5 size-4 shrink-0 text-blue-600 dark:text-blue-400" />
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <p className="text-sm font-medium">A recurring shipment already covers this lane</p>
        <p className="text-xs text-muted-foreground">
          {`"${series.name}" runs ${describeCron(series.cronExpression).toLowerCase()}`}
          {series.nextOccurrenceAt
            ? ` — next pickup ${formatToUserTimezone(series.nextOccurrenceAt)}`
            : ""}
          . You can generate the next occurrence from it instead of entering this shipment manually.
        </p>
        <div className="mt-1 flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            onClick={handleGenerate}
            disabled={generating || series.status !== "Active"}
          >
            {generating ? "Generating..." : "Generate from series"}
          </Button>
          <Button type="button" size="sm" variant="ghost" onClick={onDismiss}>
            Continue manual entry
          </Button>
        </div>
      </div>
      <button
        type="button"
        aria-label="Dismiss suggestion"
        onClick={onDismiss}
        className="text-muted-foreground hover:text-foreground"
      >
        <XIcon className="size-4" />
      </button>
    </div>
  );
}

function PatternHint({
  shipmentCount,
  onDismiss,
}: {
  shipmentCount: number;
  onDismiss: () => void;
}) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-border bg-muted/40 p-3">
      <SparklesIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <p className="text-sm font-medium">This looks like a repeating lane</p>
        <p className="text-xs text-muted-foreground">
          {`This customer has shipped this lane ${shipmentCount} times in the last 90 days. Set it up as a recurring shipment and it will generate itself on schedule.`}
        </p>
        <div className="mt-1">
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={
              <a
                href="/shipment-management/recurring-shipments"
                target="_blank"
                rel="noopener noreferrer"
              />
            }
          >
            Set up recurring shipment
          </Button>
        </div>
      </div>
      <button
        type="button"
        aria-label="Dismiss suggestion"
        onClick={onDismiss}
        className="text-muted-foreground hover:text-foreground"
      >
        <XIcon className="size-4" />
      </button>
    </div>
  );
}

export function RecurringShipmentSuggestion() {
  const { canRead } = usePermissions(Resource.RecurringShipment);
  const laneKey = useDebouncedLaneKey();
  const [dismissedKey, setDismissedKey] = useState<string>("");

  const enabled = canRead && !!laneKey;
  const { data } = useQuery({
    ...queries.recurringShipment.match(
      laneKey ?? { customerId: "", originLocationId: "", destinationLocationId: "" },
    ),
    enabled,
    staleTime: 30_000,
  });

  if (!enabled || !data) return null;

  const currentKey = laneKeyString(laneKey);
  if (dismissedKey === currentKey) return null;

  const dismiss = () => setDismissedKey(currentKey);

  const activeMatch = data.matches.find((series) => series.status === "Active");
  if (activeMatch) {
    return <MatchBanner series={activeMatch} onDismiss={dismiss} />;
  }

  if (data.pattern && data.pattern.shipmentCount >= 3) {
    return <PatternHint shipmentCount={data.pattern.shipmentCount} onDismiss={dismiss} />;
  }

  return null;
}
