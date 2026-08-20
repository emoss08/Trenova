import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { RateAgreementRule } from "@trenova/shared/types/rate";
import { HistoryIcon } from "lucide-react";
import { useState } from "react";

type LaneHistoryPopoverProps = {
  readonly rateAgreementId?: string;
  readonly laneKey: string;
};

/**
 * What this lane has cost over time.
 *
 * Every amendment — a hand edit, an imported sheet, a GRI — closes the old
 * rule out and inserts a successor, so the lineage is already in the table.
 * This is the answer to "what was the rate in March", read straight from it.
 */
export function LaneHistoryPopover({ rateAgreementId, laneKey }: LaneHistoryPopoverProps) {
  const [open, setOpen] = useState(false);

  const { data: history, isLoading } = useQuery({
    queryKey: ["rate-agreement-rule-history", rateAgreementId, laneKey],
    queryFn: () =>
      apiService.rateAgreementService.listRuleHistory(rateAgreementId as string, laneKey),
    enabled: open && Boolean(rateAgreementId) && Boolean(laneKey),
  });

  if (!rateAgreementId || !laneKey) {
    return null;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button type="button" variant="ghost" size="sm" className="h-6 gap-1 px-1.5">
            <HistoryIcon className="size-3" />
            <span className="text-2xs">History</span>
          </Button>
        }
      />
      <PopoverContent align="end" className="w-96 p-0">
        <div className="border-b p-3">
          <p className="text-xs font-medium">Rate History</p>
          <p className="mt-0.5 font-mono text-2xs text-muted-foreground">{laneKey}</p>
        </div>
        <div className="max-h-72 overflow-y-auto p-3">
          {isLoading && <p className="text-2xs text-muted-foreground">Reading the lineage…</p>}
          {!isLoading && (history?.length ?? 0) === 0 && (
            <p className="text-2xs text-muted-foreground">
              Nothing recorded yet — history begins the first time this lane is saved.
            </p>
          )}
          <div className="flex flex-col gap-2">
            {(history ?? []).map((entry) => (
              <HistoryEntry key={entry.id ?? entry.effectiveFrom} entry={entry} />
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function HistoryEntry({ entry }: { readonly entry: RateAgreementRule }) {
  const isCurrent = !entry.effectiveTo;

  return (
    <div className="flex items-center justify-between gap-3 rounded-md border px-2.5 py-1.5">
      <div>
        <p className="font-mono text-xs tabular-nums">
          {entry.rate != null ? formatCurrency(Number(entry.rate)) : "matrix-priced"}
        </p>
        <p className="text-2xs text-muted-foreground">
          {formatUnixDateMedium(entry.effectiveFrom)}
          {" — "}
          {entry.effectiveTo ? formatUnixDateMedium(entry.effectiveTo) : "current"}
        </p>
      </div>
      {isCurrent && (
        <Badge variant="secondary" className="text-[10px]">
          current
        </Badge>
      )}
    </div>
  );
}
