import type { InboundMessageCounts, TriageBriefing } from "@/lib/graphql/inbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { TriageFolder } from "../folders";
import type { DayFigures } from "./briefing-model";

type Cell = {
  key: string;
  label: string;
  value: number;
  detail: string;
  warn?: boolean;
  folder: TriageFolder;
};

/**
 * The day in figures, as the Desk's Today draws its own: one joined strip
 * under the headline. Each cell says what it counts, the number and what the
 * number means, and opens the folder that holds what was counted.
 */
export function BriefingStrip({
  briefing,
  counts,
  figures,
  onOpenFolder,
}: {
  briefing: Pick<TriageBriefing, "waiting" | "handledByNovaToday" | "corrections" | "comingBack">;
  counts: InboundMessageCounts | undefined;
  figures: DayFigures;
  onOpenFolder: (folder: TriageFolder) => void;
}) {
  const t = useT();
  const { corrections } = briefing;
  const change = Math.round(Math.abs(corrections.accuracyChange));
  const comingBackSnoozed = (counts?.snoozed ?? 0) > 0 || (counts?.waitingOnThem ?? 0) === 0;

  const cells: Cell[] = [
    {
      key: "waiting",
      label: t("Waiting on you"),
      value: briefing.waiting,
      detail:
        figures.overTarget > 0
          ? t("{0, plural, one {# over target} other {# over target}}", figures.overTarget)
          : t("None over target"),
      warn: figures.overTarget > 0,
      folder: { kind: "lane", lane: "waiting" },
    },
    {
      key: "cleared",
      label: t("Cleared by you"),
      value: figures.cleared,
      detail: t("Today"),
      folder: { kind: "lane", lane: "handled" },
    },
    {
      key: "nova",
      label: t("Handled by Nova"),
      value: briefing.handledByNovaToday,
      detail: t("Spot-check"),
      folder: { kind: "lane", lane: "handled" },
    },
    {
      key: "corrections",
      label: t("Corrections this week"),
      value: corrections.thisWeek,
      detail:
        corrections.accuracyChange > 0
          ? t("Accuracy up {0}%", change)
          : corrections.accuracyChange < 0
            ? t("Accuracy down {0}%", change)
            : t("Accuracy steady"),
      folder: { kind: "lane", lane: "all" },
    },
    {
      key: "back",
      label: t("Coming back later"),
      value: briefing.comingBack,
      detail: comingBackSnoozed ? t("Snoozed") : t("Waiting on them"),
      folder: { kind: "lane", lane: comingBackSnoozed ? "snoozed" : "them" },
    },
  ];

  return (
    <nav
      aria-label={t("The day in figures")}
      className="mt-9 grid grid-cols-5 gap-px overflow-hidden rounded-[6px] border border-dsk-b bg-dsk-b-sub @max-2xl:grid-cols-6"
    >
      {cells.map((cell, index) => (
        <button
          key={cell.key}
          type="button"
          onClick={() => onOpenFolder(cell.folder)}
          className={cn(
            "flex min-w-0 flex-col gap-0.75 ui-focus-ring bg-dsk-card px-3 pt-2.5 pb-2.75 text-left hover:bg-dsk-hover",
            // Three cells across, then two, so a narrow strip has no empty cell.
            index < 3 ? "@max-2xl:col-span-2" : "@max-2xl:col-span-3",
          )}
        >
          <span className="truncate text-xs text-dsk-subtle">{cell.label}</span>
          <span className="flex items-center gap-1.5 text-lg/tight font-semibold text-dsk-fg tabular-nums">
            {cell.warn && (
              <span aria-hidden className="size-1.5 flex-none rounded-full bg-dsk-warn" />
            )}
            {cell.value}
          </span>
          <span
            className={cn("truncate text-xs", cell.warn ? "text-dsk-warn-fg" : "text-dsk-subtle")}
          >
            {cell.detail}
          </span>
        </button>
      ))}
    </nav>
  );
}
