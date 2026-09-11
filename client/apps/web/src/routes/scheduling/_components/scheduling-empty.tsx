import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { DAY_LABELS } from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon, XIcon } from "lucide-react";

type EmptyProps = {
  title: string;
  description: string;
  className?: string;
};

/**
 * The board as it will look with people on it: a row per worker with their
 * name and terminal, and a cell per day of the week with the shift drawn in
 * where they are expected, the way the real rota lays it out.
 */
const GHOST_WEEKS: readonly { name: string; fleet: string; on: readonly boolean[] }[] = [
  { name: "w-24", fleet: "w-12", on: [false, true, true, true, true, true, false] },
  { name: "w-20", fleet: "w-14", on: [true, true, false, false, true, true, true] },
  { name: "w-28", fleet: "w-10", on: [false, true, true, true, true, false, false] },
  { name: "w-16", fleet: "w-12", on: [false, false, true, true, true, true, true] },
];

export function RotaEmpty({
  title,
  description,
  onClearFilters,
  className,
}: EmptyProps & { onClearFilters?: () => void }) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onClearFilters ? (
          <Button variant="outline" size="sm" onClick={onClearFilters}>
            <XIcon className="size-3.5" />
            {t("Clear filters")}
          </Button>
        ) : null
      }
      sketch={
        <div className="border-border/70 bg-card overflow-hidden rounded-lg border text-left">
          <div className="text-muted-foreground text-2xs grid grid-cols-[minmax(0,1.6fr)_repeat(7,minmax(0,1fr))] items-center gap-2 border-b px-3 py-1.5 leading-none">
            <span />
            {DAY_LABELS.map((label, index) => (
              <span key={index} className="text-center">
                {label}
              </span>
            ))}
          </div>
          <div className="divide-border/60 flex flex-col divide-y divide-dashed">
            {GHOST_WEEKS.map((row, index) => (
              <div
                key={index}
                className="grid grid-cols-[minmax(0,1.6fr)_repeat(7,minmax(0,1fr))] items-center gap-2 px-3 py-2"
              >
                <span className="flex min-w-0 items-center gap-2">
                  <span className="bg-muted size-6 shrink-0 rounded-full" />
                  <span className="flex min-w-0 flex-col gap-1.5">
                    <GhostLine className={`h-2 ${row.name}`} />
                    <GhostLine className={row.fleet} />
                  </span>
                </span>
                {row.on.map((on, day) => (
                  <span
                    key={day}
                    className={cn(
                      "block h-5 rounded-md",
                      on ? "bg-brand/25" : "border-border/60 border border-dashed",
                    )}
                  />
                ))}
              </div>
            ))}
          </div>
        </div>
      }
    />
  );
}

/**
 * The shift patterns as they will look: a card per pattern with its name and
 * colour, the hours it runs, the days it covers and who is on it.
 */
const GHOST_SHIFTS: readonly { name: string; window: string; days: readonly boolean[] }[] = [
  { name: "w-16", window: "w-28", days: [false, true, true, true, true, true, false] },
  { name: "w-20", window: "w-24", days: [true, true, true, false, false, true, true] },
  { name: "w-14", window: "w-32", days: [false, false, true, true, true, true, false] },
];

export function ShiftsEmpty({
  title,
  description,
  onCreate,
  className,
}: EmptyProps & { onCreate?: () => void }) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onCreate ? (
          <Button variant="outline" size="sm" onClick={onCreate}>
            <PlusIcon className="size-3.5" />
            {t("Add a shift")}
          </Button>
        ) : null
      }
      sketch={
        <div className="grid grid-cols-3 gap-3 text-left">
          {GHOST_SHIFTS.map((shift, index) => (
            <div
              key={index}
              className="border-border/70 bg-card flex flex-col gap-3 rounded-lg border p-4"
            >
              <div className="flex items-start justify-between gap-2">
                <span className="flex min-w-0 flex-col gap-1.5">
                  <span className="flex items-center gap-1.5">
                    <span className="bg-brand/40 size-2 shrink-0 rounded-full" />
                    <GhostLine className={`h-2 ${shift.name}`} />
                  </span>
                  <GhostLine className={shift.window} />
                </span>
                <GhostPill className="w-12" />
              </div>
              <span className="flex gap-1">
                {shift.days.map((on, day) => (
                  <span
                    key={day}
                    className={cn(
                      "block h-4 flex-1 rounded-sm",
                      on ? "bg-brand/25" : "border-border/60 border",
                    )}
                  />
                ))}
              </span>
            </div>
          ))}
        </div>
      }
    />
  );
}

/**
 * The swap queue as it will look with requests in it: two drivers with an
 * arrow between them, the day being offered, the state it has reached and
 * the office's two buttons at the right.
 */
const GHOST_SWAPS: readonly { names: string; date: string }[] = [
  { names: "w-40", date: "w-24" },
  { names: "w-32", date: "w-28" },
  { names: "w-36", date: "w-20" },
];

export function SwapsEmpty({ title, description, className }: EmptyProps) {
  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-xl"
      title={title}
      description={description}
      sketch={
        <div className="flex flex-col gap-2 text-left">
          {GHOST_SWAPS.map((swap, index) => (
            <div
              key={index}
              className="border-border/70 bg-card flex items-center justify-between gap-3 rounded-lg border p-3"
            >
              <span className="flex min-w-0 items-center gap-3">
                <span className="flex items-center gap-1">
                  <span className="bg-muted size-6 rounded-full" />
                  <GhostLine className="h-px w-3" />
                  <span className="bg-muted size-6 rounded-full" />
                </span>
                <span className="flex min-w-0 flex-col gap-1.5">
                  <span className="flex items-center gap-2">
                    <GhostLine className={`h-2 ${swap.names}`} />
                    <GhostPill className="w-14" />
                  </span>
                  <GhostLine className={swap.date} />
                </span>
              </span>
              <span className="flex shrink-0 gap-1.5">
                <GhostPill className="h-5 w-14 rounded-md" />
                <GhostPill className="h-5 w-16 rounded-md" />
              </span>
            </div>
          ))}
        </div>
      }
    />
  );
}

function GhostPill({ className }: { className?: string }) {
  return <span className={cn("border-border/70 block h-4 rounded-full border", className)} />;
}
