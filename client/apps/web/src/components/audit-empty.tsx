import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";

/**
 * The history as it will look once the record has changed: a card per change
 * with who made it, the operation pill beside the name, what it touched under
 * that, and how long ago at the end.
 */
const GHOST_ENTRIES: readonly { name: string; summary: string; pill: string }[] = [
  { name: "w-24", summary: "w-28", pill: "w-14" },
  { name: "w-20", summary: "w-20", pill: "w-12" },
  { name: "w-28", summary: "w-24", pill: "w-14" },
];

type AuditEmptyProps = {
  title: string;
  description: string;
  className?: string;
};

export function AuditEmpty({ title, description, className }: AuditEmptyProps) {
  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-sm"
      title={title}
      description={description}
      sketch={
        <div className="flex flex-col gap-2 text-left">
          {GHOST_ENTRIES.map((entry, index) => (
            <div
              key={index}
              className="border-border/70 bg-card flex items-center gap-3 rounded-lg border p-3"
            >
              <span className="bg-muted size-7 shrink-0 rounded-full" />
              <div className="flex min-w-0 flex-1 flex-col gap-2">
                <span className="flex items-center gap-2">
                  <GhostLine className={`h-2 ${entry.name}`} />
                  <span
                    className={`border-border/70 block h-4 rounded-full border ${entry.pill}`}
                  />
                </span>
                <GhostLine className={entry.summary} />
              </div>
              <GhostLine className="w-12 shrink-0" />
            </div>
          ))}
        </div>
      }
    />
  );
}
