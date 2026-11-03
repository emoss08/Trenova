import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import {
  createContext,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { formatTimelineTime } from "./timeline-format";
import type { TimelineScale } from "./timeline-scale";

/**
 * How long the pointer rests on a bar before its card opens: quick to read,
 * yet long enough that sweeping across the sheet opens nothing on the way.
 */
const OPEN_DELAY_MS = 80;
/** How long a card waits after the pointer leaves, so moving to the next bar switches it in place. */
const CLOSE_GRACE_MS = 90;
const CARD_WIDTH = 288;
const EDGE = 8;
/** Below this much room above the bar, the card opens under it. */
const ROOM_ABOVE = 220;

type Anchor = { load: TimelineLoad; x: number; top: number; bottom: number };

/** What a bar does with its hover: it never holds state, so hovering redraws no bar. */
export type BarHover = {
  enter: (load: TimelineLoad, element: Element, pointerX?: number) => void;
  /** The pointer moved along the bar: the card follows it without a render. */
  move: (pointerX: number) => void;
  leave: () => void;
  dismiss: () => void;
  /** A bar going away closes its own card: no pointer leave will come from it. */
  release: (loadId: string) => void;
};

function createAnchorStore() {
  let anchor: Anchor | null = null;
  let follower: ((x: number) => void) | null = null;
  const listeners = new Set<() => void>();
  return {
    get: () => anchor,
    set: (next: Anchor | null) => {
      // Scrolling dismisses on every frame; a card already closed redraws nothing.
      if (next === anchor) return;
      anchor = next;
      for (const listener of listeners) listener();
    },
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    /** Places the open card at a pointer x straight on the DOM. */
    follow: (x: number) => follower?.(x),
    /** The open card registers how it follows the pointer; null when it closes. */
    setFollower: (next: ((x: number) => void) | null) => {
      follower = next;
    },
  };
}

type AnchorStore = ReturnType<typeof createAnchorStore>;

function createBarHover(store: AnchorStore): BarHover {
  let openTimer: number | null = null;
  let closeTimer: number | null = null;
  /** The anchor a card waiting to open will take; the pointer keeps moving it. */
  let waiting: Anchor | null = null;
  const clear = () => {
    if (openTimer !== null) window.clearTimeout(openTimer);
    if (closeTimer !== null) window.clearTimeout(closeTimer);
    openTimer = null;
    closeTimer = null;
    waiting = null;
  };
  return {
    enter: (load, element, pointerX) => {
      const rect = element.getBoundingClientRect();
      const next = {
        load,
        x: pointerX ?? rect.left + rect.width / 2,
        top: rect.top,
        bottom: rect.bottom,
      };
      const open = store.get() !== null;
      clear();
      // A card already open follows the pointer to the next bar at once.
      if (open) {
        store.set(next);
        return;
      }
      waiting = next;
      openTimer = window.setTimeout(() => {
        openTimer = null;
        store.set(waiting);
        waiting = null;
      }, OPEN_DELAY_MS);
    },
    move: (pointerX) => {
      if (waiting) {
        waiting.x = pointerX;
        return;
      }
      const anchor = store.get();
      if (!anchor) return;
      // The anchor keeps the latest x, so a render of the card lands where it is.
      anchor.x = pointerX;
      store.follow(pointerX);
    },
    leave: () => {
      // A pointer leave and a blur both arrive as a bar is left: one close is pending at most.
      clear();
      if (store.get() === null) return;
      closeTimer = window.setTimeout(() => {
        closeTimer = null;
        store.set(null);
      }, CLOSE_GRACE_MS);
    },
    dismiss: () => {
      clear();
      store.set(null);
    },
    release: (loadId) => {
      if (waiting?.load.id === loadId) clear();
      if (store.get()?.load.id === loadId) {
        clear();
        store.set(null);
      }
    },
  };
}

/** The card's left edge for a pointer x: centered on it, kept inside the window. */
function cardLeft(x: number) {
  return Math.min(Math.max(x - CARD_WIDTH / 2, EDGE), window.innerWidth - CARD_WIDTH - EDGE);
}

const BarHoverContext = createContext<BarHover | null>(null);

export function useBarHover(): BarHover | null {
  return useContext(BarHoverContext);
}

/**
 * The timeline's one hover card. Each bar reports where the pointer rests and
 * this card alone redraws, on the raised floating surface the menus use. It
 * opens after a short rest, follows the pointer along a bar and from bar to
 * bar, and closes on scroll or as a drag begins.
 */
export function TimelineHoverCard({
  scale,
  children,
}: {
  scale: TimelineScale;
  children: ReactNode;
}) {
  const [store] = useState(createAnchorStore);
  const [hover] = useState(() => createBarHover(store));
  // A card waiting to open never opens after the sheet is gone.
  useEffect(() => () => hover.dismiss(), [hover]);

  return (
    <BarHoverContext.Provider value={hover}>
      {children}
      <HoverCardSurface store={store} scale={scale} />
    </BarHoverContext.Provider>
  );
}

function HoverCardSurface({ store, scale }: { store: AnchorStore; scale: TimelineScale }) {
  const anchor = useSyncExternalStore(store.subscribe, store.get, store.get);
  const cardRef = useRef<HTMLDivElement>(null);

  // While open, the pointer moves the card directly, at most once a frame.
  useLayoutEffect(() => {
    if (!anchor) return;
    let frame: number | null = null;
    let x = anchor.x;
    store.setFollower((next) => {
      x = next;
      frame ??= window.requestAnimationFrame(() => {
        frame = null;
        if (cardRef.current) cardRef.current.style.left = `${cardLeft(x)}px`;
      });
    });
    return () => {
      store.setFollower(null);
      if (frame !== null) window.cancelAnimationFrame(frame);
    };
  }, [anchor, store]);

  if (!anchor || typeof document === "undefined") return null;
  const above = anchor.top > ROOM_ABOVE;

  return createPortal(
    // Floating surfaces are drawn in the dark scope in both themes.
    <div className="dark contents">
      <div
        ref={cardRef}
        role="tooltip"
        className="bg-raised text-foreground ui-lift-float pointer-events-none fixed z-50 rounded-lg px-3 py-2.5 text-xs"
        style={{
          width: CARD_WIDTH,
          left: cardLeft(anchor.x),
          top: above ? anchor.top - EDGE : anchor.bottom + EDGE,
          transform: above ? "translateY(-100%)" : undefined,
        }}
      >
        <BarDetail load={anchor.load} scale={scale} />
      </div>
    </div>,
    document.body,
  );
}

/** Every stop with its time, how late the load runs, and who has it. */
function BarDetail({ load, scale }: { load: TimelineLoad; scale: TimelineScale }) {
  const t = useT();
  const coverage = load.coverage;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex min-w-0 items-center gap-2">
        <span className="font-mono font-semibold tabular-nums">
          {load.shipment.proNumber ?? "—"}
        </span>
        <span className="text-muted-foreground truncate">{load.shipment.customer?.name}</span>
      </div>
      <ol className="flex flex-col gap-1">
        {load.stops.map((stop, index) => {
          let when: string;
          if (stop.done) {
            when = stop.departedAt
              ? t("Left {0}", formatTimelineTime(stop.departedAt, scale))
              : t("Arrived {0}", formatTimelineTime(stop.arrivedAt ?? stop.windowStart, scale));
          } else if (load.eta && load.eta.stopIndex === index) {
            when = t("ETA {0}", formatTimelineTime(load.eta.at, scale));
          } else {
            when = formatTimelineTime(stop.windowStart, scale);
          }
          return (
            <li key={stop.id} className="grid grid-cols-[1rem_minmax(0,1fr)_auto] gap-2">
              <span className="text-muted-foreground font-mono tabular-nums">{index + 1}</span>
              <span className="truncate">{stop.place || "—"}</span>
              <span className="text-muted-foreground font-mono tabular-nums">{when}</span>
            </li>
          );
        })}
      </ol>
      {load.eta && load.eta.lateMinutes > 0 ? (
        <span className="text-danger font-medium">
          {t("{0} past the appointment window", formatMinutesSpan(load.eta.lateMinutes))}
        </span>
      ) : null}
      <span className="text-muted-foreground border-subtle border-t pt-1.5">
        {coverage.kind === "driver" || coverage.kind === "carrier"
          ? [coverage.name, coverage.detail].filter(Boolean).join(" · ")
          : coverage.kind === "tendered"
            ? t("Tendered, waiting on the carrier")
            : t("Not covered")}
      </span>
    </div>
  );
}
