import { cn } from "@trenova/shared/lib/utils";

const ENTRIES: { label: string; color: string }[] = [
  { label: "In transit", color: "var(--brand)" },
  { label: "Delayed", color: "var(--destructive)" },
  { label: "Delivered", color: "var(--success)" },
  { label: "Unassigned", color: "var(--muted-foreground)" },
];

export function ShipmentMapLegend({ className }: { className?: string }) {
  return (
    <div
      className={cn(
        "pointer-events-none absolute bottom-3 left-3 z-10 flex items-center gap-1",
        className,
      )}
    >
      {ENTRIES.map((e) => (
        <span
          key={e.label}
          className="inline-flex items-center gap-1 rounded-md border border-border bg-background p-1 font-table text-2xs text-muted-foreground"
        >
          <span
            aria-hidden
            className="mb-0.5 inline-block size-2 rounded-full"
            style={{ background: e.color }}
          />
          {e.label}
        </span>
      ))}
    </div>
  );
}
