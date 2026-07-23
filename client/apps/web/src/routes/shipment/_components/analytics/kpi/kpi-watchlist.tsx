import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import type React from "react";
import type { Tone } from "../mock-data";
import { KpiCard, KpiHeader } from "./kpi-card";
import { toneVar } from "./tone";

export type WatchlistItem = {
  id: string;
  who: string;
  meta: string;
  tone: Tone;
};

type KpiWatchlistProps = {
  label: string;
  items: WatchlistItem[];
  icon?: React.ReactNode;
  info?: React.ReactNode;
  span?: 2 | 3;
};

export function KpiWatchlist({ label, items, icon, info, span = 3 }: KpiWatchlistProps) {
  return (
    <KpiCard span={span} density="compact" className="gap-1.5 p-2.5">
      <KpiHeader
        icon={icon}
        label={label}
        info={info}
        right={
          <span className="font-mono text-[10px] text-muted-foreground/80 tabular-nums">
            {items.length}
          </span>
        }
      />
      <ScrollArea
        className="-mr-1 min-h-0 flex-1"
        viewportClassName="pr-1"
        type="hover"
        maskVariant="card"
        maskHeight={16}
      >
        <div className="flex flex-col gap-0.75">
          {items.map((item, index) => (
            <div
              key={item.id}
              className={cn(
                "flex items-center justify-between gap-2 rounded-sm px-1.5 py-0.75",
                index === 0 && "bg-foreground/4",
              )}
            >
              <span className="flex min-w-0 items-center gap-1.5 overflow-hidden">
                <span
                  aria-hidden
                  className="size-1.25 shrink-0 rounded-full"
                  style={{ background: toneVar(item.tone) }}
                />
                <span className="truncate font-mono text-[11px] text-foreground">{item.who}</span>
              </span>
              <span
                className="shrink-0 font-mono text-[10.5px] tabular-nums"
                style={{ color: toneVar(item.tone) }}
              >
                {item.meta}
              </span>
            </div>
          ))}
        </div>
      </ScrollArea>
    </KpiCard>
  );
}
