import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";

const GHOST_SECTIONS: readonly { heading: string; rows: readonly string[] }[] = [
  { heading: "w-20", rows: ["w-3/5", "w-2/5", "w-1/2"] },
  { heading: "w-28", rows: ["w-1/2", "w-3/5"] },
];

type FinancialReportEmptyProps = {
  title: string;
  description: string;
  className?: string;
};

/**
 * A financial statement drawn as what it is: sections with a heading, the
 * account lines under each with their amounts at the right, and the total
 * bar that closes the section, fading out where the figures would go.
 */
export function FinancialReportEmpty({ title, description, className }: FinancialReportEmptyProps) {
  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-xl"
      title={title}
      description={description}
      sketch={
        <div className="flex flex-col gap-4 text-left">
          {GHOST_SECTIONS.map((section, index) => (
            <div key={index} className="flex flex-col gap-2">
              <div className="border-border/70 bg-card rounded-md border">
                <div className="border-b px-3 py-2">
                  <GhostLine className={`h-2 ${section.heading}`} />
                </div>
                <div className="divide-border/60 flex flex-col divide-y divide-dashed px-3">
                  {section.rows.map((width, row) => (
                    <div key={row} className="flex items-center justify-between gap-6 py-2">
                      <span className="flex min-w-0 flex-1 items-center gap-3">
                        <GhostLine className="h-2 w-8" />
                        <GhostLine className={width} />
                      </span>
                      <GhostLine className="h-2 w-14 shrink-0" />
                    </div>
                  ))}
                </div>
              </div>
              <div className="bg-muted/40 border-border/70 flex items-center justify-between rounded-md border px-3 py-2">
                <GhostLine className="bg-foreground/30 h-2 w-16" />
                <GhostLine className="bg-foreground/30 h-2.5 w-16" />
              </div>
            </div>
          ))}
        </div>
      }
    />
  );
}
