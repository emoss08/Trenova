import {
  createContext,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type FocusEvent,
  type PointerEvent,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { cn } from "@trenova/shared/lib/utils";

/**
 * How long the pointer rests on a mark before its card opens: quick to read,
 * yet long enough that sweeping across a chart opens nothing on the way.
 */
const OPEN_DELAY_MS = 80;
/** How long a card waits after the pointer leaves, so moving to the next mark switches it in place. */
const CLOSE_GRACE_MS = 90;
const EDGE = 8;
/** Below this much room above the mark, the card opens under it. */
const ROOM_ABOVE = 220;
/** Where a card that follows the pointer sits from it: right of and below, as a chart's tooltip does. */
const POINTER_OFFSET_X = 14;
const POINTER_OFFSET_Y = 18;

/**
 * Where a card sits. `mark` holds it above or below the mark and follows the pointer along it, for
 * a bar read in place (a timeline). `pointer` holds it by the pointer in both directions, for a
 * chart whose marks are tall hit areas.
 */
export type HoverCardFollow = "mark" | "pointer";

type Anchor<T> = { item: T; x: number; y: number; top: number; bottom: number };

/** What a mark does with its hover: it never holds state, so hovering redraws no mark. */
export type PointerHover<T> = {
  enter: (item: T, element: Element, pointerX?: number, pointerY?: number) => void;
  /** The pointer moved along the mark: the card follows it without a render. */
  move: (pointerX: number, pointerY?: number) => void;
  leave: () => void;
  dismiss: () => void;
  /** A mark going away closes its own card: no pointer leave will come from it. */
  release: (id: string) => void;
};

function createAnchorStore<T>() {
  let anchor: Anchor<T> | null = null;
  let follower: ((x: number, y: number) => void) | null = null;
  const listeners = new Set<() => void>();
  return {
    get: () => anchor,
    set: (next: Anchor<T> | null) => {
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
    /** Places the open card at the pointer straight on the DOM. */
    follow: (x: number, y: number) => follower?.(x, y),
    /** The open card registers how it follows the pointer; null when it closes. */
    setFollower: (next: ((x: number, y: number) => void) | null) => {
      follower = next;
    },
  };
}

type AnchorStore<T> = ReturnType<typeof createAnchorStore<T>>;

function createPointerHover<T>(store: AnchorStore<T>, itemId: (item: T) => string) {
  let openTimer: number | null = null;
  let closeTimer: number | null = null;
  /** The anchor a card waiting to open will take; the pointer keeps moving it. */
  let waiting: Anchor<T> | null = null;
  const clear = () => {
    if (openTimer !== null) window.clearTimeout(openTimer);
    if (closeTimer !== null) window.clearTimeout(closeTimer);
    openTimer = null;
    closeTimer = null;
    waiting = null;
  };
  const hover: PointerHover<T> = {
    enter: (item, element, pointerX, pointerY) => {
      const rect = element.getBoundingClientRect();
      const next = {
        item,
        x: pointerX ?? rect.left + rect.width / 2,
        y: pointerY ?? rect.top + rect.height / 2,
        top: rect.top,
        bottom: rect.bottom,
      };
      const open = store.get() !== null;
      clear();
      // A card already open follows the pointer to the next mark at once.
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
    move: (pointerX, pointerY) => {
      if (waiting) {
        waiting.x = pointerX;
        waiting.y = pointerY ?? waiting.y;
        return;
      }
      const anchor = store.get();
      if (!anchor) return;
      // The anchor keeps the latest position, so a render of the card lands where it is.
      anchor.x = pointerX;
      anchor.y = pointerY ?? anchor.y;
      store.follow(anchor.x, anchor.y);
    },
    leave: () => {
      // A pointer leave and a blur both arrive as a mark is left: one close is pending at most.
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
    release: (id) => {
      if (waiting && itemId(waiting.item) === id) clear();
      const anchor = store.get();
      if (anchor && itemId(anchor.item) === id) {
        clear();
        store.set(null);
      }
    },
  };
  return hover;
}

/** The card's left edge for a pointer x: centered on it, kept inside the window. */
function cardLeft(x: number, width: number) {
  return Math.min(Math.max(x - width / 2, EDGE), window.innerWidth - width - EDGE);
}

/**
 * Beside the pointer, right of and below it; on the other side of it where the window ends, so
 * the card never covers the pointer or leaves the screen.
 */
function besidePointer(x: number, y: number, width: number, height: number) {
  const right = x + POINTER_OFFSET_X;
  const below = y + POINTER_OFFSET_Y;
  const left = right + width > window.innerWidth - EDGE ? x - POINTER_OFFSET_X - width : right;
  const top = below + height > window.innerHeight - EDGE ? y - POINTER_OFFSET_X - height : below;

  return { left: Math.max(left, EDGE), top: Math.max(top, EDGE) };
}

const PointerHoverContext = createContext<PointerHover<unknown> | null>(null);

/** The hover of the nearest card, typed by what its marks carry. */
export function usePointerHover<T>(): PointerHover<T> | null {
  return useContext(PointerHoverContext) as PointerHover<T> | null;
}

/**
 * The hover and focus handlers for one mark that only shows its card: a chart column, a cell, a
 * row. A mark that also drags or selects calls the hover itself. Its card closes when the mark
 * goes away.
 */
export function usePointerHoverTarget<T>(item: T, id: string) {
  const hover = usePointerHover<T>();
  useEffect(() => () => hover?.release(id), [hover, id]);

  return {
    onPointerEnter: (event: PointerEvent<Element>) =>
      hover?.enter(item, event.currentTarget, event.clientX, event.clientY),
    onPointerMove: (event: PointerEvent<Element>) => hover?.move(event.clientX, event.clientY),
    onPointerLeave: () => hover?.leave(),
    onFocus: (event: FocusEvent<Element>) => hover?.enter(item, event.currentTarget),
    onBlur: () => hover?.leave(),
  };
}

/**
 * One hover card for a set of marks: timeline bars, chart columns, heatmap cells. Each mark
 * reports where the pointer rests and this card alone redraws, on the raised floating surface the
 * menus use. It opens after a short rest, follows the pointer along a mark and from mark to mark,
 * and closes when the pointer leaves.
 */
export function PointerHoverCard<T>({
  itemId,
  render,
  width,
  follow = "mark",
  children,
}: {
  /** Identifies a mark's item, so a mark going away closes only its own card. */
  itemId: (item: T) => string;
  render: (item: T) => ReactNode;
  /** A fixed width; without one the card fits its content, as a chart's tooltip does. */
  width?: number;
  follow?: HoverCardFollow;
  children: ReactNode;
}) {
  const [store] = useState(createAnchorStore<T>);
  const [hover] = useState(() => createPointerHover(store, itemId));
  // A card waiting to open never opens after its marks are gone.
  useEffect(() => () => hover.dismiss(), [hover]);

  return (
    <PointerHoverContext.Provider value={hover as PointerHover<unknown>}>
      {children}
      <HoverCardSurface store={store} render={render} width={width} follow={follow} />
    </PointerHoverContext.Provider>
  );
}

function HoverCardSurface<T>({
  store,
  render,
  width,
  follow,
}: {
  store: AnchorStore<T>;
  render: (item: T) => ReactNode;
  width: number | undefined;
  follow: HoverCardFollow;
}) {
  const anchor = useSyncExternalStore(store.subscribe, store.get, store.get);
  const cardRef = useRef<HTMLDivElement>(null);

  // The card is placed once it is drawn, so a card that fits its content is centered on its real
  // width. While open, the pointer moves it directly, at most once a frame.
  useLayoutEffect(() => {
    if (!anchor) return;
    const place = (x: number, y: number) => {
      const card = cardRef.current;
      if (!card) return;
      if (follow === "pointer") {
        const at = besidePointer(x, y, width ?? card.offsetWidth, card.offsetHeight);
        card.style.left = `${at.left}px`;
        card.style.top = `${at.top}px`;
        return;
      }
      card.style.left = `${cardLeft(x, width ?? card.offsetWidth)}px`;
    };
    place(anchor.x, anchor.y);
    let frame: number | null = null;
    let x = anchor.x;
    let y = anchor.y;
    store.setFollower((nextX, nextY) => {
      x = nextX;
      y = nextY;
      frame ??= window.requestAnimationFrame(() => {
        frame = null;
        place(x, y);
      });
    });
    return () => {
      store.setFollower(null);
      if (frame !== null) window.cancelAnimationFrame(frame);
    };
  }, [anchor, store, width, follow]);

  if (!anchor || typeof document === "undefined") return null;
  const above = follow === "mark" && anchor.top > ROOM_ABOVE;

  return createPortal(
    // Floating surfaces are drawn in the dark scope in both themes.
    <div className="dark contents">
      <div
        ref={cardRef}
        role="tooltip"
        className={cn(
          "bg-raised text-foreground ui-lift-float pointer-events-none fixed z-50 rounded-lg text-xs",
          width === undefined
            ? "flex max-w-72 min-w-44 flex-col gap-0.5 px-2.5 py-2"
            : "px-3 py-2.5",
        )}
        style={
          follow === "pointer"
            ? { width }
            : {
                width,
                top: above ? anchor.top - EDGE : anchor.bottom + EDGE,
                transform: above ? "translateY(-100%)" : undefined,
              }
        }
      >
        {render(anchor.item)}
      </div>
    </div>,
    document.body,
  );
}

/** A chart tooltip's first line: what the mark is. */
export function HoverCardHeading({ children }: { children: ReactNode }) {
  return <p className="font-semibold">{children}</p>;
}

/**
 * One line of a chart tooltip: a muted label, with the series swatch when it names one, and its
 * figure on the right.
 */
export function HoverCardRow({
  label,
  value,
  swatchClass,
  className,
}: {
  label: ReactNode;
  value?: ReactNode;
  /** The background class of the series the row names. */
  swatchClass?: string;
  className?: string;
}) {
  return (
    <p className={cn("flex items-center justify-between gap-3.5", className)}>
      <span className="text-muted-foreground inline-flex min-w-0 items-center gap-1.5">
        {swatchClass ? (
          <span aria-hidden className={cn("size-2 shrink-0 rounded-xs", swatchClass)} />
        ) : null}
        <span className="truncate">{label}</span>
      </span>
      {value !== undefined ? (
        <span className="shrink-0 font-mono font-medium tabular-nums">{value}</span>
      ) : null}
    </p>
  );
}
