import { useT } from "@trenova/shared/i18n/use-t";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { humanizeTariffName } from "@trenova/shared/lib/tariff-inputs";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type { RateAgreementRule } from "@trenova/shared/types/rate";
import type { TariffWireValue } from "@trenova/shared/types/tariff";
import { ClockRewindIcon } from "@trenova/shared/components/icons";
import { useState } from "react";

type LaneHistoryPopoverProps = {
  readonly rateAgreementId?: string;
  /** The stored key, ids and all — what the history endpoint matches on. */
  readonly laneKey: string;
  /** The key as a person reads it, names in place of ids. */
  readonly displayLaneKey?: string;
};

/**
 * What this lane has cost over time.
 *
 * Every amendment — a hand edit, an imported sheet, a GRI — closes the old
 * rule out and inserts a successor, so the lineage is already in the table.
 * This is the answer to "what was the rate in March", read straight from it.
 */
export function LaneHistoryPopover({
  rateAgreementId,
  laneKey,
  displayLaneKey,
}: LaneHistoryPopoverProps) {
  const t = useT();

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
            <ClockRewindIcon className="size-3" />
            <span className="text-2xs">{t("History")}</span>
          </Button>
        }
      />
      <PopoverContent align="end" className="w-96 p-0">
        <div className="border-b p-3">
          <p className="text-xs font-medium">{t("Lane history")}</p>
          <p className="text-2xs text-muted-foreground mt-0.5 font-mono">
            {displayLaneKey ?? laneKey}
          </p>
        </div>
        <ScrollArea className="h-70 px-3 pb-2">
          {isLoading && (
            <p className="text-2xs text-muted-foreground">{t("Reading the lineage…")}</p>
          )}
          {!isLoading && (history?.length ?? 0) === 0 && (
            <p className="text-2xs text-muted-foreground">
              {t("Nothing recorded yet — history begins the first time this lane is saved.")}
            </p>
          )}
          <div className="flex flex-col gap-2">
            {(history ?? []).map((entry, index, all) => (
              <HistoryEntry
                key={entry.id ?? entry.effectiveFrom}
                entry={entry}
                previous={all.find(
                  (other, at) => at !== index && other.id && other.id === entry.supersedesRuleId,
                )}
              />
            ))}
          </div>
        </ScrollArea>
      </PopoverContent>
    </Popover>
  );
}

/** The names of the inputs whose value differs between two versions of a lane. */
function changedInputs(
  entry: RateAgreementRule,
  previous: RateAgreementRule | undefined,
): Set<string> {
  if (!previous) return new Set();
  const names = new Set([
    ...Object.keys(entry.inputs ?? {}),
    ...Object.keys(previous.inputs ?? {}),
  ]);
  return new Set(
    [...names].filter(
      (name) => JSON.stringify(entry.inputs?.[name]) !== JSON.stringify(previous.inputs?.[name]),
    ),
  );
}

function HistoryEntry({
  entry,
  previous,
}: {
  readonly entry: RateAgreementRule;
  /** The version this one replaced, to mark which terms it changed. */
  readonly previous?: RateAgreementRule;
}) {
  const t = useT();
  const isCurrent = !entry.effectiveTo;
  const inputs = Object.entries(entry.inputs ?? {});
  const changed = changedInputs(entry, previous);

  return (
    <div className="flex items-start justify-between gap-3 rounded-md border px-2.5 py-1.5">
      <div className="flex min-w-0 flex-col gap-0.5">
        {inputs.length === 0 ? (
          <p className="text-muted-foreground text-xs">{t("The tariff's defaults")}</p>
        ) : (
          <ul className="flex flex-col gap-0.5">
            {inputs.map(([name, value]) => (
              <li key={name} className="flex items-baseline gap-1.5 text-xs">
                <span className="text-muted-foreground">{humanizeTariffName(name)}</span>
                <span className={cn("font-mono tabular-nums", changed.has(name) && "font-medium")}>
                  {formatTariffValue(value as TariffWireValue, t, { currency: entry.currency })}
                </span>
                {changed.has(name) && (
                  <Badge variant="info" className="text-2xs">
                    {t("Changed")}
                  </Badge>
                )}
              </li>
            ))}
          </ul>
        )}
        <p className="text-2xs text-muted-foreground">
          {entry.effectiveTo
            ? t(
                "{0} — {1}",
                formatUnixDateMedium(entry.effectiveFrom),
                formatUnixDateMedium(entry.effectiveTo),
              )
            : t("{0} — current", formatUnixDateMedium(entry.effectiveFrom))}
        </p>
      </div>
      {isCurrent && (
        <Badge variant="neutral" className="text-2xs">
          {t("Current")}
        </Badge>
      )}
    </div>
  );
}
