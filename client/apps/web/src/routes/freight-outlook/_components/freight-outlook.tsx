import { ForecastNotReady } from "@/components/dispatch/forecast-not-ready";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { createRefitPoll, forecastStanding } from "@/lib/dispatch-forecast";
import { FREIGHT_OUTLOOK_REFETCH_MS, freightOutlook } from "@/lib/queries/freight-outlook";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type ReactNode } from "react";
import { AreaInspector, type OpenArea } from "./area-inspector";
import { BoardSketch } from "./board-sketch";
import { BriefRail } from "./brief-rail";
import { FreightOutlookSkeleton } from "./freight-outlook-skeleton";
import { OutlookBoard } from "./outlook-board";
import { OutlookBriefPanel } from "./outlook-brief-panel";
import { horizonSlots, moveKey, movesIn, viewIndexes, type Horizon } from "./outlook-model";
import { OutlookBrief, OutlookFigures } from "./outlook-overview";

/** Whether the inspector is unfolded: a per-browser layout choice, never a record. */
const BRIEF_OPEN_KEY = "trenova-freight-outlook-brief-open";
/** An outlook with no board to draw sits in the middle of the workspace, not under its header. */
function BoardStandIn({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center overflow-y-auto">
      {children}
    </div>
  );
}

/**
 * The freight outlook as a full-screen workspace, the way the dispatch console is laid out:
 * figures across the top, the board (the network's tide over every area's windows, on one time
 * axis) and an inspector holding the brief, or an area once one is picked. A window pinned on
 * the board re-reads everything at that hour; a move pointed at in the brief is drawn on it.
 */
export function FreightOutlook({ horizon }: { horizon: Horizon }) {
  const t = useT();
  const [refitPoll] = useState(() => createRefitPoll(FREIGHT_OUTLOOK_REFETCH_MS));
  const query = useQuery({
    ...freightOutlook.outlook(),
    refetchInterval: (current) => refitPoll(current.state.data?.quality.refitting),
  });
  const [pinned, setPinned] = useState<number | null>(null);
  const [openArea, setOpenArea] = useState<OpenArea | null>(null);
  const [hotKey, setHotKey] = useState<string | null>(null);
  const [dismissed, setDismissed] = useState<ReadonlySet<string>>(() => new Set());
  const [briefOpen, setBriefOpen] = useLocalStorage(BRIEF_OPEN_KEY, true);
  const outlook = query.data;

  useEffect(() => {
    if (!openArea) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== "Escape" || isTypingTarget(event.target) || isWithinDialog(event.target)) {
        return;
      }
      setOpenArea(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [openArea]);

  if (query.isPending) {
    return <FreightOutlookSkeleton />;
  }
  if (!outlook) {
    return (
      <ErrorState
        error={query.error}
        title={t("The freight outlook didn't load")}
        description={t(
          "This isn't an empty outlook: trucks may still be short somewhere. Try again in a moment.",
        )}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (forecastStanding(outlook.quality) !== "Ready") {
    return (
      <BoardStandIn>
        <ForecastNotReady quality={outlook.quality} sketch={<BoardSketch />} />
      </BoardStandIn>
    );
  }
  if (outlook.areas.length === 0) {
    return (
      <BoardStandIn>
        <EmptySheet
          sketch={<BoardSketch />}
          title={t("No forecast areas yet")}
          description={t(
            "The forecast learns areas from where your loads pick up. Once it has, they appear here.",
          )}
        />
      </BoardStandIn>
    );
  }

  const slots = horizonSlots(horizon);
  const effectivePinned = pinned !== null && pinned < outlook.slotStarts.length ? pinned : null;
  const indexes = viewIndexes(slots, effectivePinned);
  const open = (cell: number, slotStart: number) => {
    setHotKey(null);
    setBriefOpen(true);
    setOpenArea((current) =>
      current?.cell === cell && current.slotStart === slotStart ? null : { cell, slotStart },
    );
  };
  const jump = (section: string) => {
    setOpenArea(null);
    setBriefOpen(true);
    requestAnimationFrame(() =>
      document.getElementById(section)?.scrollIntoView({ behavior: "smooth", block: "start" }),
    );
  };
  const hotMove =
    hotKey === null
      ? null
      : (movesIn(outlook.moves, outlook.slotStarts, indexes).find(
          (move) => moveKey(move) === hotKey,
        ) ?? null);

  return (
    <div
      className={cn(
        "flex min-h-135 flex-1 flex-col transition-opacity",
        query.isFetching && !query.isPending && "opacity-90",
      )}
    >
      <OutlookFigures outlook={outlook} indexes={indexes} pinned={effectivePinned} onJump={jump} />
      <div className="flex min-h-0 flex-1 max-xl:flex-col">
        <section
          aria-label={t("Board")}
          className="flex min-w-0 flex-1 flex-col gap-3 px-4 pt-3.5 pb-4 xl:min-h-0 xl:overflow-hidden"
        >
          <OutlookBrief
            outlook={outlook}
            indexes={indexes}
            pinned={effectivePinned}
            onOpenArea={open}
            onJump={jump}
          />
          <OutlookBoard
            outlook={outlook}
            indexes={indexes}
            horizonSlots={slots}
            pinned={effectivePinned}
            onPin={setPinned}
            selectedCell={openArea?.cell ?? null}
            hotMove={hotMove}
            onOpenArea={open}
          />
        </section>
        {/* Folds by width, as the console's inspector does, so the board takes back the room
            rather than being covered; the content keeps its width and fades so nothing reflows
            mid-fold. */}
        <aside
          aria-label={openArea ? t("Area") : t("Brief")}
          className={cn(
            "relative flex shrink-0 flex-col overflow-hidden border-border-subtle bg-card max-xl:border-t xl:min-h-0 xl:border-l",
            "transition-[width] duration-320 ease-settle motion-reduce:transition-none",
            briefOpen ? "xl:w-100" : "max-xl:h-11 xl:w-11",
          )}
        >
          <div
            inert={!briefOpen}
            aria-hidden={!briefOpen}
            className={cn(
              "flex min-h-0 flex-1 flex-col transition-opacity duration-200 motion-reduce:transition-none xl:w-100",
              briefOpen ? "opacity-100" : "pointer-events-none opacity-0",
            )}
          >
            {openArea ? (
              <AreaInspector
                open={openArea}
                onBack={() => setOpenArea(null)}
                onCollapse={() => setBriefOpen(false)}
              />
            ) : (
              <OutlookBriefPanel
                outlook={outlook}
                indexes={indexes}
                pinned={effectivePinned}
                hotKey={hotKey}
                onHot={setHotKey}
                onOpenArea={open}
                onCollapse={() => {
                  setHotKey(null);
                  setBriefOpen(false);
                }}
                dismissed={dismissed}
                onDismissedChange={setDismissed}
              />
            )}
          </div>
          {briefOpen ? null : (
            <BriefRail
              moves={
                movesIn(outlook.moves, outlook.slotStarts, indexes).filter(
                  (move) => !dismissed.has(moveKey(move)),
                ).length
              }
              onExpand={() => setBriefOpen(true)}
            />
          )}
        </aside>
      </div>
    </div>
  );
}
