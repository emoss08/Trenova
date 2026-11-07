import type { FreightOutlook } from "@/lib/graphql/freight-outlook";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { CollapseBriefButton } from "./brief-rail";
import { movesIn, moveKey } from "./outlook-model";
import { OutlookMoves } from "./outlook-moves";
import { readingSpan } from "./outlook-overview";
import { OutlookPace, OutlookRecord } from "./outlook-side";

/** An area this short with no move toward it is named under the moves as out of reach. */
const UNREACHED_SHORT = 1.5;

/**
 * The inspector's resting state, one scroll as the shipment board's side panel sets its brief:
 * how far to trust the forecast, the moves worth making and the customers off their pace.
 */
export function OutlookBriefPanel({
  outlook,
  indexes,
  pinned,
  hotKey,
  onHot,
  onOpenArea,
  onCollapse,
  dismissed,
  onDismissedChange,
}: {
  outlook: FreightOutlook;
  indexes: readonly number[];
  pinned: number | null;
  hotKey: string | null;
  onHot: (key: string | null) => void;
  onOpenArea: (cell: number, slotStart: number) => void;
  onCollapse: () => void;
  dismissed: ReadonlySet<string>;
  onDismissedChange: (next: ReadonlySet<string>) => void;
}) {
  const t = useT();
  const reading = movesIn(outlook.moves, outlook.slotStarts, indexes);
  const moves = reading.filter((move) => !dismissed.has(moveKey(move)));
  const hidden = reading.length - moves.length;
  const suggested = new Set(reading.map((move) => `${move.workerId}:${move.toCell}`));
  const sent = outlook.positionings.filter(
    (positioning) => !suggested.has(`${positioning.workerId}:${positioning.toCell}`),
  );
  const covered = new Set([
    ...reading.map((move) => move.toCell),
    ...outlook.positionings.map((positioning) => positioning.toCell),
  ]);
  const unreached = outlook.areas.filter(
    (area) =>
      !covered.has(area.cell) &&
      indexes.some((index) => (area.slots[index]?.short ?? 0) >= UNREACHED_SHORT),
  );

  return (
    <div className="flex min-h-0 flex-1 animate-in flex-col duration-200 ease-settle fade-in-0 slide-in-from-right-3 motion-reduce:animate-none">
      <div className="flex h-11 shrink-0 items-center gap-2 border-b border-border-subtle px-3.5">
        <h2 className="text-sm font-semibold">{t("Brief")}</h2>
        <span className="truncate text-xs text-foreground-subtle">
          {readingSpan(outlook, indexes, pinned, t)}
        </span>
        <span className="ml-auto">
          <CollapseBriefButton onCollapse={onCollapse} />
        </span>
      </div>
      <ScrollArea
        maskVariant="card"
        maskHeight={18}
        className="min-h-0 flex-1"
        viewportClassName="overscroll-contain"
      >
        <div className="flex flex-col gap-4 p-3.5">
          <OutlookRecord outlook={outlook} />
          <OutlookMoves
            outlook={outlook}
            moves={moves}
            sent={sent}
            unreached={unreached}
            dismissed={hidden}
            hotKey={hotKey}
            onHot={onHot}
            onDismiss={(key) => {
              onHot(null);
              onDismissedChange(new Set(dismissed).add(key));
            }}
            onRestore={() => onDismissedChange(new Set())}
            onOpenArea={onOpenArea}
          />
          <OutlookPace outlook={outlook} indexes={indexes} onOpenArea={onOpenArea} />
        </div>
      </ScrollArea>
    </div>
  );
}
