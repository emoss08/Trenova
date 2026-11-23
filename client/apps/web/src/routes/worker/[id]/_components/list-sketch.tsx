import { GhostLine } from "@trenova/shared/components/ui/empty-sheet";

/** The faint drawing an empty or failed list shows: rows of unwritten lines. */
export function ListSketch({ rows = 3 }: { rows?: number }) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border p-3">
      {Array.from({ length: rows }, (_, row) => (
        <div key={row} className="grid grid-cols-[1rem_minmax(0,1fr)_4rem] items-center gap-3">
          <span className="size-3.5 rounded-sm border" />
          <GhostLine className={row % 2 === 0 ? "w-3/5" : "w-4/5"} />
          <GhostLine />
        </div>
      ))}
    </div>
  );
}

/**
 * The faint drawing an empty sheet shows for a record that is written out rather than listed: a
 * card of unwritten lines, one per width (`["w-2/3", "w-1/2"]`).
 */
export function LineSketch({ widths }: { widths: readonly string[] }) {
  return (
    <div className="flex flex-col gap-2 rounded-md border bg-card p-4">
      {widths.map((width, index) => (
        <GhostLine key={index} className={width} />
      ))}
    </div>
  );
}
