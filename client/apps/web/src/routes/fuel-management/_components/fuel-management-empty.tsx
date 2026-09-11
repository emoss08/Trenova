import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon } from "lucide-react";

const GHOST_TILES = 4;
const CHART_POINTS = "0,34 18,30 36,31 54,24 72,26 90,18 108,20 126,14 144,16 162,10 180,12";

type EmptyProps = {
  title: string;
  description: string;
  className?: string;
};

/**
 * The dashboard as it will look once an index carries prices: a tile per
 * index with this week's price and its move, and the trend line beneath.
 */
export function FuelDashboardEmpty({
  title,
  description,
  onOpenIndices,
  className,
}: EmptyProps & { onOpenIndices?: () => void }) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-xl"
      title={title}
      description={description}
      action={
        onOpenIndices ? (
          <Button variant="outline" size="sm" onClick={onOpenIndices}>
            {t("Open fuel indices")}
          </Button>
        ) : null
      }
      sketch={
        <div className="flex flex-col gap-3 text-left">
          <div className="grid grid-cols-4 gap-3">
            {Array.from({ length: GHOST_TILES }, (_, index) => (
              <div
                key={index}
                className="border-border/70 bg-card flex flex-col gap-2 rounded-md border p-3"
              >
                <GhostLine className="w-3/5" />
                <GhostLine className="h-3 w-12" />
                <span className="flex items-center gap-1.5">
                  <GhostPill className="h-3.5 w-8" />
                  <GhostLine className="w-10" />
                </span>
              </div>
            ))}
          </div>
          <div className="border-border/70 bg-card rounded-md border p-3">
            <div className="flex items-center justify-between">
              <GhostLine className="h-2 w-28" />
              <span className="flex gap-1">
                <GhostPill className="w-8 rounded-md" />
                <GhostPill className="w-8 rounded-md" />
                <GhostPill className="w-8 rounded-md" />
              </span>
            </div>
            <svg viewBox="0 0 180 40" className="mt-3 h-20 w-full" preserveAspectRatio="none">
              <polyline
                points={CHART_POINTS}
                fill="none"
                stroke="currentColor"
                strokeWidth="1.5"
                className="text-brand/40"
                vectorEffect="non-scaling-stroke"
              />
            </svg>
          </div>
        </div>
      }
    />
  );
}

/**
 * The programs tab as it will look with programs on it: a card per program
 * with its name and code, this week's rate large beneath, and the method
 * and price pills along the bottom, the way a real card reads.
 */
const GHOST_PROGRAMS: readonly { name: string; code: string; rate: string }[] = [
  { name: "w-28", code: "w-20", rate: "w-20" },
  { name: "w-24", code: "w-24", rate: "w-16" },
  { name: "w-32", code: "w-16", rate: "w-14" },
];

export function FuelProgramsEmpty({
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
            {t("Create a program")}
          </Button>
        ) : null
      }
      sketch={
        <div className="grid grid-cols-3 gap-3 text-left">
          {GHOST_PROGRAMS.map((program, index) => (
            <div
              key={index}
              className="border-border/70 bg-card flex flex-col gap-3 rounded-lg border p-4"
            >
              <div className="flex flex-col gap-1.5">
                <GhostLine className={`h-2 ${program.name}`} />
                <GhostLine className={program.code} />
              </div>
              <span className="flex items-baseline gap-1.5 pt-1">
                <GhostLine className={`h-4 ${program.rate}`} />
                <GhostLine className="w-8" />
              </span>
              <span className="flex gap-1.5">
                <GhostPill className="w-14 rounded-md" />
                <GhostPill className="w-16 rounded-md" />
              </span>
            </div>
          ))}
        </div>
      }
    />
  );
}

/**
 * The indices tab as it will look with indices listed: the table's own
 * columns, and a row per index with its code, name, source pill, latest
 * price and week, fading out where the rest would go.
 */
const INDEX_GRID =
  "grid grid-cols-[3rem_minmax(0,1.4fr)_minmax(0,1fr)_4rem_4.5rem_3.5rem_3.5rem] items-center gap-4";
const GHOST_INDEX_ROWS: readonly { name: string; region: string }[] = [
  { name: "w-4/5", region: "w-3/5" },
  { name: "w-3/5", region: "w-2/5" },
  { name: "w-full", region: "w-1/2" },
];

export function FuelIndicesEmpty({
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
            {t("Add a custom index")}
          </Button>
        ) : null
      }
      sketch={
        <div className="border-border/70 bg-card rounded-md border text-left">
          <div
            className={cn(
              INDEX_GRID,
              "text-muted-foreground text-2xs border-b px-3 py-1.5 leading-none",
            )}
          >
            <span>{t("Code")}</span>
            <span>{t("Name")}</span>
            <span>{t("Region")}</span>
            <span>{t("Source")}</span>
            <span>{t("Latest")}</span>
            <span>{t("Week")}</span>
            <span>{t("Status")}</span>
          </div>
          {GHOST_INDEX_ROWS.map((row, index) => (
            <div
              key={index}
              className={cn(
                INDEX_GRID,
                "border-border/60 border-b border-dashed px-3 py-2.5 last:border-0",
              )}
            >
              <GhostLine className="h-2 w-8" />
              <GhostLine className={row.name} />
              <GhostLine className={row.region} />
              <GhostPill className="w-12 rounded-md" />
              <GhostLine className="h-2 w-12" />
              <GhostLine className="w-10" />
              <GhostPill className="w-12 rounded-md" />
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
