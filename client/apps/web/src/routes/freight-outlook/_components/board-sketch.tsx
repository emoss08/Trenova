import { cn } from "@trenova/shared/lib/utils";

const SKETCH_ROWS = 6;
const SKETCH_WINDOWS = 12;
/** A few warm and cool cells, so the empty board reads as the heat grid it will become. */
const SKETCH_TONE: Record<string, string> = {
  "0:4": "bg-warning/40",
  "0:5": "bg-warning/70",
  "1:2": "bg-warning/30",
  "3:3": "bg-accent-sky/40",
  "3:4": "bg-accent-sky/60",
  "4:8": "bg-accent-sky/30",
};

/** The board before it has anything to show: area rows over the windows ahead. */
export function BoardSketch() {
  return (
    <div className="flex flex-col gap-1">
      {Array.from({ length: SKETCH_ROWS }, (_, row) => (
        <div key={row} className="grid grid-cols-[6rem_repeat(12,minmax(0,1fr))] gap-0.5">
          <span className="my-1.5 h-2 w-16 rounded-full bg-border/70" />
          {Array.from({ length: SKETCH_WINDOWS }, (_, col) => (
            <span
              key={col}
              className={cn("h-5 rounded-[3px]", SKETCH_TONE[`${row}:${col}`] ?? "bg-foreground/4")}
            />
          ))}
        </div>
      ))}
    </div>
  );
}
