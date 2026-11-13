import { pageTour } from "@/lib/queries/page-tour";
import { recordMyPageTour, type PageTourOutcome, type PageTourView } from "@/lib/graphql/page-tour";
import { usePageTourStore } from "@/stores/page-tour-store";
import { Popover as PopoverPrimitive } from "@base-ui/react/popover";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
} from "@trenova/shared/components/ui/popover";
import { useElementSize } from "@trenova/shared/hooks/use-element-size";
import { useReducedMotion } from "@trenova/shared/hooks/use-reduced-motion";
import { useT } from "@trenova/shared/i18n/use-t";
import { reportFailure } from "@trenova/shared/lib/failure-reporter";
import { useEffect, useRef, useState } from "react";
import { outgrowsItsClip, useDrawnTargets, useStepTarget, useTourFrame } from "./tour-frame";
import type { Extent } from "./tour-geometry";
import { isTourSeen, nextStepIndex, type TourDefinition } from "./tour-model";
import { TourSpotlight } from "./tour-spotlight";

/** Idle until its page is ready, then open at a step, then closed for the rest of the visit. */
type TourState = "idle" | "closed" | { index: number };

/** Where a step with nothing to point at sits: a point a little above the window's middle. */
const CENTER_ANCHOR = {
  getBoundingClientRect: () =>
    DOMRect.fromRect({
      x: window.innerWidth / 2,
      y: window.innerHeight * 0.3,
      width: 0,
      height: 0,
    }),
};

/** The card's padding (`p-4`), which its measured content size leaves out. */
const CARD_PADDING = 16;
/** The card's size before it has been measured: its width and a body of five lines. */
const UNMEASURED_CARD: Extent = { width: 352, height: 240 };

/**
 * A page's tour: a short walk through what the page shows, one part at a time, each part lit
 * while the rest is dimmed. It opens by itself the first time a person reaches the page (once
 * `ready` says the parts it points at are drawn), is remembered on the server so it never opens
 * by itself again on any device, and can be replayed from the page header's About note.
 *
 * A page renders one, next to the content the tour points at, and marks each target with
 * `tourTarget(name)`. Finishing and skipping both count as seen; a tour whose steps change
 * raises its `version` to be shown again.
 */
export function PageTour({ tour, ready = true }: { tour: TourDefinition; ready?: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const options = pageTour.views();
  const views = useQuery({ ...options, staleTime: Number.POSITIVE_INFINITY, enabled: ready });
  const requested = usePageTourStore((state) => state.requested);
  const announce = usePageTourStore((state) => state.announce);
  const withdraw = usePageTourStore((state) => state.withdraw);
  const clearRequest = usePageTourStore((state) => state.clearRequest);
  const always = usePageTourStore((state) => state.always);
  const show = usePageTourStore((state) => state.show);
  const hide = usePageTourStore((state) => state.hide);
  const still = useReducedMotion();
  const [cardRef, cardSize] = useElementSize<HTMLDivElement>();
  const [state, setState] = useState<TourState>("idle");
  const recording = useRef(false);

  useEffect(() => {
    if (!ready) return;
    announce(tour.key);
    return () => withdraw(tour.key);
  }, [ready, tour.key, announce, withdraw]);

  const steps = tour.steps(t);
  const replaying = ready && requested === tour.key;
  const unseen = ready && views.isSuccess && (always || !isTourSeen(views.data, tour));
  const index =
    typeof state === "object" ? state.index : replaying || (state === "idle" && unseen) ? 0 : null;
  const step = index === null ? undefined : steps[index];
  const showing = step !== undefined;
  const target = useStepTarget(step?.target);
  const drawn = useDrawnTargets(
    steps.flatMap((each) => (each.target === undefined ? [] : [each.target])),
    showing,
  );
  const onPage = (name: string) => drawn.has(name);
  const card =
    cardSize.height > 0
      ? {
          width: cardSize.width + CARD_PADDING * 2,
          height: cardSize.height + CARD_PADDING * 2,
        }
      : UNMEASURED_CARD;
  const frame = useTourFrame(target, step?.side ?? "bottom", card);

  // The page is told while its tour is open, so it can draw itself whole: unfold a panel a
  // step points into, and put it back as it was afterwards.
  useEffect(() => {
    if (!showing) return;
    show(tour.key);
    return () => hide(tour.key);
  }, [showing, tour.key, show, hide]);

  // A part that fits is brought to the middle; one longer than the box it scrolls in moves
  // only as far as it must, so a list stays at its top instead of jumping to its middle.
  useEffect(() => {
    if (!target) return;
    target.scrollIntoView?.({
      block: outgrowsItsClip(target) ? "nearest" : "center",
      inline: "nearest",
      behavior: still ? "auto" : "smooth",
    });
  }, [target, still]);

  if (index === null || !step) return null;

  const remember = (outcome: PageTourOutcome) => {
    if (recording.current) return;
    recording.current = true;
    recordMyPageTour({ tour: tour.key, tourVersion: tour.version, outcome })
      .then((view) => {
        queryClient.setQueryData<PageTourView[]>(options.queryKey, (current = []) => [
          ...current.filter((seen) => seen.tour !== view.tour),
          view,
        ]);
      })
      .catch((error: unknown) => {
        reportFailure(error, { source: "background", operation: "recordPageTour" });
      })
      .finally(() => {
        recording.current = false;
      });
  };
  const close = (outcome: PageTourOutcome) => {
    setState("closed");
    if (replaying) clearRequest();
    remember(outcome);
  };
  const go = (direction: 1 | -1) => {
    const next = nextStepIndex(steps, index, direction, onPage);
    if (next === null) {
      if (direction === 1) close("Finished");
      return;
    }
    setState({ index: next });
  };
  const last = nextStepIndex(steps, index, 1, onPage) === null;
  const first = nextStepIndex(steps, index, -1, onPage) === null;

  return (
    <>
      <TourSpotlight lit={frame?.lit ?? null} />
      <Popover
        open
        modal="trap-focus"
        onOpenChange={(open, details) => {
          if (!open && details.reason !== "outside-press") close("Skipped");
        }}
      >
        <PopoverContent
          ref={cardRef}
          anchor={frame?.anchor ?? CENTER_ANCHOR}
          side={frame?.side ?? "bottom"}
          align="center"
          sideOffset={frame?.gap ?? 0}
          data-slot="page-tour"
          className="w-88 max-w-[calc(100vw-2rem)] gap-3 p-4"
          onKeyDown={(event) => {
            if (event.key === "ArrowRight") go(1);
            if (event.key === "ArrowLeft") go(-1);
          }}
        >
          <PopoverHeader className="flex-row items-start justify-between gap-3">
            <PopoverTitle className="text-sm leading-snug font-semibold">{step.title}</PopoverTitle>
            <PopoverPrimitive.Close
              render={
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label={t("Skip tour")}
                  className="-mt-1 -mr-1.5 shrink-0"
                />
              }
            >
              <XCloseIcon className="size-3.5" />
            </PopoverPrimitive.Close>
          </PopoverHeader>
          <PopoverDescription className="text-sm leading-relaxed text-muted-foreground">
            {step.body}
          </PopoverDescription>
          <div className="flex items-center justify-between gap-3 pt-1">
            <span aria-live="polite" className="text-xs text-foreground-subtle tabular-nums">
              {t("Step {0} of {1}", index + 1, steps.length)}
            </span>
            <div className="flex items-center gap-1.5">
              {first ? (
                <Button variant="ghost" size="sm" onClick={() => close("Skipped")}>
                  {t("Skip tour")}
                </Button>
              ) : (
                <Button variant="ghost" size="sm" onClick={() => go(-1)}>
                  {t("Back")}
                </Button>
              )}
              <Button size="sm" autoFocus onClick={() => go(1)}>
                {last ? t("Done") : t("Next")}
              </Button>
            </div>
          </div>
        </PopoverContent>
      </Popover>
    </>
  );
}
