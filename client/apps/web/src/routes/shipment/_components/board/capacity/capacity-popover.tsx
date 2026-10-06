import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { CapacityMatch, CapacityUnit } from "@/lib/graphql/shipment-board";
import type { CapacityProvider } from "@/lib/shipment-board/capacity-providers";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import type { ReactElement } from "react";

const MATCH_LIMIT = 2;

const clock = (unix: number | null | undefined) =>
  unix
    ? formatToUserTimezone(unix, { showTimeZone: false, showSeconds: false, showDate: false })
    : "—";

type CapacityPopoverProps = {
  unit: CapacityUnit;
  provider: CapacityProvider;
  hos: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCommit: (unit: CapacityUnit, match: CapacityMatch) => void;
  pending: boolean;
  trigger: ReactElement;
};

/**
 * The loads a driver or carrier fits best, with the action that commits one.
 * The first match is the recommendation; the second is the alternative.
 */
export function CapacityPopover({
  unit,
  provider,
  hos,
  open,
  onOpenChange,
  onCommit,
  pending,
  trigger,
}: CapacityPopoverProps) {
  const t = useT();
  const { data: matches = [], isLoading } = useQuery({
    ...queries.shipmentBoard.capacityMatches(unit.kind, unit.id, MATCH_LIMIT),
    enabled: open,
    staleTime: 15_000,
  });

  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger render={trigger} />
      <PopoverContent align="start" className="animate-pop-in w-[340px] gap-3 p-3">
        <div className="flex flex-col gap-0.5">
          <span className="text-sm font-semibold">{unit.name}</span>
          <span className="text-muted-foreground font-mono text-xs">{provider.popoverMeta(unit, t, hos)}</span>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className="text-muted-foreground text-xs font-medium">{provider.matchesTitle(unit, t)}</span>
          {isLoading ? (
            <>
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </>
          ) : matches.length === 0 ? (
            <span className="text-muted-foreground text-sm">{t("Nothing uncovered fits right now.")}</span>
          ) : (
            matches.map((match, index) => (
              <div key={match.shipmentId} className="flex items-center gap-2">
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-sm font-medium">
                    {match.originCity} → {match.destinationCity}
                  </span>
                  <span className="text-muted-foreground truncate font-mono text-xs">
                    {provider.matchDetail(match, t, clock(match.pickupAt))}
                  </span>
                </div>
                <Button
                  size="xs"
                  variant={index === 0 ? "default" : "outline"}
                  disabled={pending}
                  onClick={() => onCommit(unit, match)}
                >
                  {provider.commitLabel(t)}
                </Button>
              </div>
            ))
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
