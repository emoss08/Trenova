import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { NotificationPanelLens } from "./notification-lenses";

const TITLES: Record<NotificationPanelLens, string> = defineLabels({
  forYou: "Nothing is waiting on you",
  team: "Your team is clear",
  later: "Nothing set aside",
  done: "Nothing done yet",
});

const DESCRIPTIONS: Record<NotificationPanelLens, string> = defineLabels({
  forYou: "What is assigned to you, mentions you or needs your decision shows up here.",
  team: "Broadcasts to everyone who can act on something show up here, one row per subject.",
  later: "Send a notification to Later and it comes back when you said.",
  done: "What you finish and what clears itself is kept here.",
});

const SKETCH_LINES = ["w-3/5", "w-4/5", "w-2/5"];

/** A part of the panel with nothing in it, drawn as the rows it will hold. */
export function NotificationEmpty({ lens }: { lens: NotificationPanelLens }) {
  const t = useT();

  return (
    <EmptySheet
      title={t(TITLES[lens])}
      description={t(DESCRIPTIONS[lens])}
      className="py-11"
      sketchClassName="max-w-55"
      sketch={
        <div className="flex flex-col gap-1.5">
          {SKETCH_LINES.map((width) => (
            <div
              key={width}
              className="flex items-center gap-2 rounded-md border border-border p-2"
            >
              <span className="size-4.5 shrink-0 rounded-md bg-surface-hover" />
              <GhostLine className={cn("bg-surface-hover", width)} />
            </div>
          ))}
        </div>
      }
    />
  );
}
