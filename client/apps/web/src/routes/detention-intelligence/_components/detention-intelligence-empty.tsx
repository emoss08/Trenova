import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";

const GHOST_COLUMNS = [90, 55, 46, 32, 26, 20, 14, 9];
const GHOST_STEPS: readonly { title: string; detail: string }[] = [
  { title: "w-1/2", detail: "w-4/5" },
  { title: "w-2/5", detail: "w-3/4" },
  { title: "w-3/5", detail: "w-2/3" },
];

type DetentionIntelligenceEmptyProps = {
  title: string;
  description: string;
  /** Offered while the window can still be widened. */
  onWiden?: () => void;
  widenLabel: string;
};

/**
 * The page as it will look once stops settle detention: the next steps, and the ranking of the
 * facilities that cause the hours past free time.
 */
export function DetentionIntelligenceEmpty({
  title,
  description,
  onWiden,
  widenLabel,
}: DetentionIntelligenceEmptyProps) {
  return (
    <EmptySheet
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onWiden ? (
          <Button variant="outline" size="sm" onClick={onWiden}>
            {widenLabel}
          </Button>
        ) : null
      }
      sketch={
        <div className="flex flex-col gap-3 text-left">
          <div className="border-border/70 bg-card divide-border/60 flex flex-col divide-y divide-dashed rounded-lg border">
            {GHOST_STEPS.map((step, index) => (
              <div key={index} className="flex items-center gap-3 px-3 py-2.5">
                <span className="bg-muted size-5 shrink-0 rounded-md" />
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <GhostLine className={`h-2 ${step.title}`} />
                  <GhostLine className={step.detail} />
                </div>
                <GhostLine className="h-2 w-12" />
              </div>
            ))}
          </div>
          <div className="border-border/70 bg-card flex h-32 items-end gap-3 rounded-lg border px-4 pb-3">
            {GHOST_COLUMNS.map((share, index) => (
              <span
                key={index}
                className="bg-muted block flex-1 rounded-t-sm"
                style={{ height: `${share}%` }}
              />
            ))}
          </div>
        </div>
      }
    />
  );
}
