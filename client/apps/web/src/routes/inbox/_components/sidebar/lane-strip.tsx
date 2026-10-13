import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@trenova/shared/components/ui/sheet";
import { Menu01Icon } from "@trenova/shared/components/icons";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { laneLabel } from "../folder-labels";
import { BRIEFING, isSameFolder, type TriageFolder } from "../folders";
import { laneCount, visibleLanes } from "../lanes";
import { useTriageCounts } from "../use-triage-data";
import { TriageSidebar } from "./triage-sidebar";

/**
 * The rail on a phone: the Briefing and the lanes as a strip that scrolls
 * sideways, and every other folder behind one button, in a sheet.
 */
export function LaneStrip({
  folder,
  onSelect,
  onShortcuts,
}: {
  folder: TriageFolder;
  onSelect: (folder: TriageFolder) => void;
  onShortcuts: () => void;
}) {
  const t = useT();
  const counts = useTriageCounts().data;
  const [open, setOpen] = useState(false);
  const chip = (active: boolean) =>
    cn(
      "ui-focus-ring flex h-7 shrink-0 items-center gap-1.5 rounded-full px-3 text-sm transition-colors",
      active
        ? "bg-nav-active text-foreground font-medium"
        : "text-foreground-muted ring-foreground/10 ring-1 hover:text-foreground",
    );

  return (
    <div className="border-border-subtle flex items-center gap-1.5 border-b px-3 py-2">
      <button
        type="button"
        aria-label={t("Folders")}
        onClick={() => setOpen(true)}
        className="ui-focus-ring text-foreground-muted hover:bg-surface-hover flex size-7 shrink-0 items-center justify-center rounded-md"
      >
        <Menu01Icon aria-hidden className="size-4" />
      </button>
      <div className="flex min-w-0 flex-1 gap-1.5 overflow-x-auto">
        <button
          type="button"
          aria-pressed={folder.kind === "briefing"}
          onClick={() => onSelect(BRIEFING)}
          className={chip(folder.kind === "briefing")}
        >
          <NovaMark className="size-3" />
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
                <span className="bg-warning text-warning-on-solid rounded-full px-1.5 font-mono text-2xs font-semibold tabular-nums">
                  {count}
                </span>
              )}
            </button>
          );
        })}
      </div>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="left" className="w-72 p-0">
          <SheetHeader className="px-4 pt-4 pb-0">
            <SheetTitle>{t("Triage")}</SheetTitle>
          </SheetHeader>
          <TriageSidebar
            folder={folder}
            onSelect={(next) => {
              setOpen(false);
              onSelect(next);
            }}
            onShortcuts={() => {
              setOpen(false);
              onShortcuts();
            }}
          />
        </SheetContent>
      </Sheet>
    </div>
  );
}
