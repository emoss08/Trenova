import { cn } from "@/lib/utils";

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
        "pointer-events-none absolute bottom-3 left-3 z-10 flex items-center gap-2 rounded-md border border-border bg-card/85 px-2 py-1 text-[10px] font-medium text-muted-foreground shadow-sm backdrop-blur-sm",
        className,
      )}
    >
      {ENTRIES.map((e) => (
        <span key={e.label} className="inline-flex items-center gap-1">
          <span
            aria-hidden
            className="inline-block size-2 rounded-full"
            style={{ background: e.color }}
          />
          {e.label}
        </span>
      ))}
    </div>
  );
}
