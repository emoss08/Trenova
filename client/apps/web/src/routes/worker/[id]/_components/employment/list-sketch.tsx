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
