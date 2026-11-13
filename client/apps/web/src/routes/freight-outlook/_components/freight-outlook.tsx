import { ForecastNotReady } from "@/components/dispatch/forecast-not-ready";
import { FloatingPanel } from "@/components/floating-panel";
import { PageTour } from "@/components/tour/page-tour";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { createRefitPoll, forecastStanding } from "@/lib/dispatch-forecast";
import { FREIGHT_OUTLOOK_REFETCH_MS, freightOutlook } from "@/lib/queries/freight-outlook";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { LayoutRightIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { AreaInspector, type OpenArea } from "./area-inspector";
import { BoardSketch } from "./board-sketch";
import { FreightOutlookSkeleton } from "./freight-outlook-skeleton";
import { OutlookBoard } from "./outlook-board";
import { OutlookBriefPanel } from "./outlook-brief-panel";
import { horizonSlots, moveKey, movesIn, viewIndexes, type Horizon } from "./outlook-model";
import { OutlookBrief, OutlookFigures } from "./outlook-overview";
import { OUTLOOK_TOUR } from "./outlook-tour";
import { usePageTourStore } from "@/stores/page-tour-store";

/** Whether the brief's panel is open: a per-browser layout choice, never a record. */
const BRIEF_OPEN_KEY = "trenova-freight-outlook-brief-open";
/** The panel's width: what the track record, a move's row and an area's map are drawn for. */
const PANEL_WIDTH = 400;
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
 * axis) and a panel floating over its right side holding the brief, or an area once one is
 * picked. A window pinned on
 * the board re-reads everything at that hour; a move pointed at in the brief is drawn on it.
 */
export function FreightOutlook({
  horizon,
  briefSlot,
}: {
  horizon: Horizon;
  /**
   * The place in the page header for the button that opens the brief. The outlook draws the
   * button there once it has an outlook to open, with the count of moves waiting in it.
   */
  briefSlot: HTMLElement | null;
}) {
  const t = useT();
  const [refitPoll] = useState(() => createRefitPoll(FREIGHT_OUTLOOK_REFETCH_MS));
  const query = useQuery({
    ...freightOutlook.outlook(),
    refetchInterval: (current) => refitPoll(current.state.data?.quality.refitting),
  });
  const [pinned, setPinned] = useState<number | null>(null);
  const [chosenArea, setOpenArea] = useState<OpenArea | null>(null);
  const [hotKey, setHotKey] = useState<string | null>(null);
  const [dismissed, setDismissed] = useState<ReadonlySet<string>>(() => new Set());
  const [briefLeftOpen, setBriefOpen] = useLocalStorage(BRIEF_OPEN_KEY, true);
  // The tour points into the brief, so while it runs the brief is drawn, whether its panel
  // was left closed or showing an area; both come back as they were when the tour closes.
  const touring = usePageTourStore((state) => state.showing === OUTLOOK_TOUR.key);
  const briefOpen = briefLeftOpen || touring;
  const openArea = touring ? null : chosenArea;
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
  const waiting = movesIn(outlook.moves, outlook.slotStarts, indexes).filter(
    (move) => !dismissed.has(moveKey(move)),
  ).length;
  const closePanel = () => {
    setHotKey(null);
    setOpenArea(null);
    setBriefOpen(false);
  };

  return (
    <div
      className={cn(
        "flex min-h-135 flex-1 flex-col transition-opacity",
        query.isFetching && !query.isPending && "opacity-90",
      )}
    >
      <PageTour tour={OUTLOOK_TOUR} />
      {briefSlot
        ? createPortal(
            <Button
              type="button"
              size="sm"
              variant="outline"
              aria-pressed={briefOpen}
              className="aria-pressed:border-brand/30 aria-pressed:bg-brand-subtle aria-pressed:text-brand-subtle-foreground"
              onClick={() => (briefOpen ? closePanel() : setBriefOpen(true))}
            >
              <LayoutRightIcon className="size-3.5" />
              {t("Brief")}
              {waiting > 0 ? (
                <Badge variant="success" className="h-4 px-1.5 text-2xs tabular-nums">
                  {formatNumber(waiting)}
                </Badge>
              ) : null}
            </Button>,
            briefSlot,
          )
        : null}
      <OutlookFigures outlook={outlook} indexes={indexes} pinned={effectivePinned} onJump={jump} />
      {/* The panel floats over the board instead of sitting beside it, so opening and closing
          it never resizes the board: the chart and every area row keep their width and are
          not laid out or drawn again while the panel arrives. */}
      <div className="@container relative flex min-h-0 flex-1">
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
        {briefOpen ? (
          <FloatingPanel label={openArea ? t("Area") : t("Brief")} width={PANEL_WIDTH}>
            {openArea ? (
              <AreaInspector
                open={openArea}
                onBack={() => setOpenArea(null)}
                onClose={closePanel}
              />
            ) : (
              <OutlookBriefPanel
                outlook={outlook}
                indexes={indexes}
                pinned={effectivePinned}
                hotKey={hotKey}
                onHot={setHotKey}
                onOpenArea={open}
                onClose={closePanel}
                dismissed={dismissed}
                onDismissedChange={setDismissed}
              />
            )}
          </FloatingPanel>
        ) : null}
      </div>
    </div>
  );
}
