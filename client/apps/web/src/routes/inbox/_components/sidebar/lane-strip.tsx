import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { laneLabel } from "../folder-labels";
import { BRIEFING, isSameFolder, type TriageFolder } from "../folders";
import { laneCount, visibleLanes } from "../lanes";
import { useTriageCounts } from "../use-triage-data";

/**
 * The lanes while the Desk's rail is out of sight (folded, floating, or a
 * sheet on a phone): the Briefing and the lanes as a strip that scrolls
 * sideways. Every other folder is in the rail, one press away.
 */
export function LaneStrip({
  folder,
  onSelect,
}: {
  folder: TriageFolder;
  onSelect: (folder: TriageFolder) => void;
}) {
  const t = useT();
  const counts = useTriageCounts().data;
  const chip = (active: boolean) =>
    cn(
      "flex h-7 shrink-0 items-center gap-1.5 rounded-badge ui-focus-ring px-3 text-sm transition-colors",
      active
        ? "bg-nav-active font-medium text-foreground"
        : "text-foreground-muted ring-1 ring-foreground/10 hover:text-foreground",
    );

  return (
    <div className="flex items-center gap-1.5 border-b border-border-subtle px-3 py-2">
      <div className="flex min-w-0 flex-1 gap-1.5 overflow-x-auto">
        <button
          type="button"
          aria-pressed={folder.kind === "briefing"}
          onClick={() => onSelect(BRIEFING)}
          className={chip(folder.kind === "briefing")}
        >
          <NovaBadge size="xs" />
          {t("Briefing")}
        </button>
        {visibleLanes(counts).map((lane) => {
          const target: TriageFolder = { kind: "lane", lane };
          const active = isSameFolder(folder, target);
          const count = laneCount(counts, lane);
          return (
            <button
              key={lane}
              type="button"
              aria-pressed={active}
              onClick={() => onSelect(target)}
              className={chip(active)}
            >
              {laneLabel(t, lane)}
              {lane === "waiting" && count > 0 && (
                <span className="rounded-badge bg-warning px-1.5 font-mono text-2xs font-semibold text-warning-on-solid tabular-nums">
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
