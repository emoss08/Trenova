import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";

const GHOST_WEEKS: readonly { name: string; share: number }[] = [
  { name: "w-24", share: 70 },
  { name: "w-32", share: 55 },
  { name: "w-20", share: 85 },
];

type EmptyProps = {
  title: string;
  description: string;
  className?: string;
};

/**
 * The queue as it will look with weeks in it: a person, their week, the
 * hours as one track, the total, and the decision waiting at the end.
 */
export function TimesheetsEmpty({ title, description, className }: EmptyProps) {
  return (
    <EmptySheet
      className={className}
      title={title}
      description={description}
      sketch={
        <div className="border-border/70 bg-card divide-border/60 divide-y divide-dashed rounded-md border">
          {GHOST_WEEKS.map((week, index) => (
            <div
              key={index}
              className="grid grid-cols-[auto_minmax(0,1fr)_auto_auto] items-center gap-4 px-3 py-2.5"
            >
              <span className="bg-muted size-7 rounded-md" />
              <div className="flex min-w-0 flex-col gap-2">
                <span className="flex items-center gap-2">
                  <GhostLine className={`h-2 ${week.name}`} />
                  <GhostLine className="w-16" />
                </span>
                <GhostBar share={week.share} className="w-40" />
              </div>
              <GhostLine className="h-2 w-10" />
              <span className="border-border/70 block h-5 w-16 rounded-md border" />
            </div>
          ))}
        </div>
      }
    />
  );
}

const GHOST_RUNS: readonly { period: string; share: number }[] = [
  { period: "w-28", share: 100 },
  { period: "w-28", share: 100 },
];

/** The run history as it will look: a file per period, with what it carried. */
export function PayrollRunsEmpty({ title, description, className }: EmptyProps) {
  return (
    <EmptySheet
      className={className}
      title={title}
      description={description}
      sketch={
        <div className="flex flex-col gap-2">
          {GHOST_RUNS.map((run, index) => (
            <div
              key={index}
              className="border-border/70 bg-card grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-md border px-3 py-2.5"
            >
              <span className="bg-muted size-7 rounded-md" />
              <div className="flex min-w-0 flex-col gap-2">
                <span className="flex items-center gap-2">
                  <GhostLine className={`h-2 ${run.period}`} />
                  <span className="border-border/70 block h-4 w-10 rounded-md border" />
                </span>
                <GhostBar share={run.share} className="w-48" />
              </div>
              <span className="flex gap-1.5">
                <span className="border-border/70 block h-5 w-12 rounded-md border" />
                <span className="border-border/70 block h-5 w-10 rounded-md border" />
              </span>
            </div>
          ))}
        </div>
      }
    />
  );
}
