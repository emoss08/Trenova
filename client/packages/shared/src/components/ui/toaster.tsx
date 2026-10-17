import { ToastCard, type SwipeDirection } from "@trenova/shared/components/ui/toast-card";
import { layoutToastStack } from "@trenova/shared/components/ui/toast-stack";
import {
  belongsToToaster,
  toastStore,
  type ToastId,
} from "@trenova/shared/components/ui/toast-store";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { AnimatePresence, m } from "motion/react";
import { useEffect, useRef, useState, useSyncExternalStore, type CSSProperties } from "react";

export type ToastPosition =
  | "top-left"
  | "top-center"
  | "top-right"
  | "bottom-left"
  | "bottom-center"
  | "bottom-right";

export type ToasterProps = {
  /** Names this Toaster, so `toast(…, { toasterId })` lands here instead of in the app's. */
  id?: string;
  position?: ToastPosition;
  /** Distance in pixels from the screen edge the stack grows from. */
  offset?: number;
  /** How many toasts a collapsed stack draws. */
  visibleToasts?: number;
  /** Alt+T moves focus to the newest toast. On by default for the app's unnamed Toaster. */
  hotkey?: boolean;
  className?: string;
};

const GAP = 8;
const PEEK = 10;
const EXPANDED_VISIBLE = 6;
const CLEAR_SPACE = 34;
const DEFAULT_OFFSET = 16;
const HOTKEY_CODE = "KeyT";

const PLACEMENT: Record<ToastPosition, string> = {
  "top-left": "left-4",
  "top-center": "left-1/2 -translate-x-1/2",
  "top-right": "right-4",
  "bottom-left": "left-4",
  "bottom-center": "left-1/2 -translate-x-1/2",
  "bottom-right": "right-4",
};

const SWIPE: Record<ToastPosition, SwipeDirection> = {
  "top-left": "left",
  "top-center": "both",
  "top-right": "right",
  "bottom-left": "left",
  "bottom-center": "both",
  "bottom-right": "right",
};

const STACK_SPRING = { type: "spring", stiffness: 400, damping: 34, mass: 0.9 } as const;

function subscribeVisibility(listener: () => void) {
  document.addEventListener("visibilitychange", listener);
  return () => document.removeEventListener("visibilitychange", listener);
}

const documentHidden = () => document.hidden;
const serverHidden = () => false;

/**
 * Where toasts appear. Every toast is drawn dark in both themes, like the menus and
 * popovers that also float above the page, and the stack opens on hover or focus. Mount
 * one unnamed Toaster per app; a page that wants its toasts somewhere else mounts a named
 * one and addresses it with `toasterId`.
 */
export function Toaster({
  id,
  position = "top-right",
  offset = DEFAULT_OFFSET,
  visibleToasts = 3,
  hotkey = id === undefined,
  className,
}: ToasterProps) {
  const t = useT();
  const snapshot = useSyncExternalStore(
    toastStore.subscribe,
    toastStore.getSnapshot,
    toastStore.getSnapshot,
  );
  const hidden = useSyncExternalStore(subscribeVisibility, documentHidden, serverHidden);
  const [heights, setHeights] = useState<ReadonlyMap<ToastId, number>>(() => new Map());
  const [hovered, setHovered] = useState(false);
  const [focused, setFocused] = useState(false);
  const [touchExpanded, setTouchExpanded] = useState(false);
  const region = useRef<HTMLElement>(null);
  const elements = useRef(new Map<ToastId, HTMLLIElement>());
  const returnFocus = useRef<HTMLElement | null>(null);
  const polite = useRef<HTMLDivElement>(null);
  const assertive = useRef<HTMLDivElement>(null);

  useEffect(() => (id === undefined ? undefined : toastStore.registerToaster(id)), [id]);

  const toasts = snapshot.toasts.filter((toast) => belongsToToaster(toast, id, snapshot.toasters));
  const count = toasts.length;

  if (count === 0 && touchExpanded) {
    setTouchExpanded(false);
  }

  useEffect(() => {
    setHeights((current) => {
      const live = new Set(toasts.map((toast) => toast.id));
      let stale = false;
      for (const key of current.keys()) {
        if (!live.has(key)) {
          stale = true;
          break;
        }
      }
      if (!stale) {
        return current;
      }
      const next = new Map<ToastId, number>();
      for (const [key, value] of current) {
        if (live.has(key)) {
          next.set(key, value);
        }
      }
      return next;
    });
  }, [toasts]);

  useEffect(() => {
    if (!hotkey) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (!event.altKey || event.code !== HOTKEY_CODE || event.ctrlKey || event.metaKey) {
        return;
      }
      const front = toastStore.getSnapshot().toasts.find((toast) => elements.current.has(toast.id));
      const element = front ? elements.current.get(front.id) : undefined;
      if (!element) {
        return;
      }
      event.preventDefault();
      if (!region.current?.contains(document.activeElement)) {
        returnFocus.current =
          document.activeElement instanceof HTMLElement ? document.activeElement : null;
      }
      element.focus();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [hotkey]);

  const expanded = count > 1 && (hovered || (focused && count > 0) || touchExpanded);
  const paused = hovered || (focused && count > 0) || touchExpanded || hidden;
  const direction: 1 | -1 = position.startsWith("top") ? 1 : -1;
  const stack = layoutToastStack({
    heights: toasts.map((toast) => heights.get(toast.id) ?? 0),
    expanded,
    direction,
    gap: GAP,
    peek: PEEK,
    visible: visibleToasts,
    expandedVisible: EXPANDED_VISIBLE,
  });
  const showClear = expanded && count >= 2;

  const restoreFocus = () => {
    const target = returnFocus.current;
    returnFocus.current = null;
    if (target?.isConnected) {
      target.focus();
    } else if (region.current?.contains(document.activeElement)) {
      (document.activeElement as HTMLElement).blur();
    }
  };

  /** Moves focus off a toast that is leaving, to the next one or back where it came from. */
  const handOffFocus = (leaving: ToastId) => {
    const element = elements.current.get(leaving);
    if (!element?.contains(document.activeElement)) {
      return;
    }
    const index = toasts.findIndex((toast) => toast.id === leaving);
    const next = toasts[index + 1] ?? toasts[index - 1];
    const nextElement = next ? elements.current.get(next.id) : undefined;
    if (nextElement) {
      nextElement.focus();
    } else {
      restoreFocus();
    }
  };

  const handleDismiss = (toastId: ToastId) => {
    handOffFocus(toastId);
    toastStore.dismiss(toastId);
  };

  const handleExpire = (toastId: ToastId) => {
    handOffFocus(toastId);
    toastStore.expire(toastId);
  };

  const handleHeight = (toastId: ToastId, height: number) => {
    setHeights((current) => {
      if (current.get(toastId) === height) {
        return current;
      }
      const next = new Map(current);
      next.set(toastId, height);
      return next;
    });
  };

  const handleAnnounce = (text: string, urgent: boolean) => {
    const target = urgent ? assertive.current : polite.current;
    if (!target) {
      return;
    }
    target.textContent = "";
    window.requestAnimationFrame(() => {
      target.textContent = text;
    });
  };

  const handleElement = (toastId: ToastId, element: HTMLLIElement | null) => {
    if (element) {
      elements.current.set(toastId, element);
    } else {
      elements.current.delete(toastId);
    }
  };

  const clearAll = () => {
    restoreFocus();
    for (const toast of toasts) {
      toastStore.dismiss(toast.id);
    }
  };

  // i18n-ignore: CSS length for the stack's inset, not interface text
  const edge = `calc(${offset}px + env(safe-area-inset-${direction === 1 ? "top" : "bottom"}, 0px))`;
  const style: CSSProperties = direction === 1 ? { top: edge } : { bottom: edge };

  return (
    <section
      ref={region}
      aria-label={t("Notifications")}
      aria-keyshortcuts={hotkey ? "Alt+T" : undefined}
      tabIndex={-1}
      style={style}
      className={cn(
        "fixed z-100 w-[min(23rem,calc(100vw-2rem))] outline-none max-sm:left-1/2 max-sm:right-auto max-sm:-translate-x-1/2",
        PLACEMENT[position],
        count === 0 && "pointer-events-none",
        className,
      )}
      onPointerEnter={(event) => {
        if (event.pointerType === "mouse") {
          setHovered(true);
        }
      }}
      onPointerLeave={(event) => {
        if (event.pointerType === "mouse") {
          setHovered(false);
        }
      }}
      onFocus={() => setFocused(true)}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setFocused(false);
        }
      }}
      onKeyDown={(event) => {
        if (event.key === "Escape" && event.target === event.currentTarget) {
          event.preventDefault();
          restoreFocus();
        }
      }}
    >
      <m.ol
        className="relative m-0 list-none p-0"
        initial={false}
        animate={{ height: stack.height + (showClear ? CLEAR_SPACE : 0) }}
        transition={STACK_SPRING}
      >
        <AnimatePresence initial={false}>
          {toasts.map((toast, index) => (
            <ToastCard
              key={toast.id}
              toast={toast}
              slot={stack.slots[index]}
              index={index}
              direction={direction}
              paused={paused}
              swipe={SWIPE[position]}
              onHeight={handleHeight}
              onAnnounce={handleAnnounce}
              onDismiss={handleDismiss}
              onExpire={handleExpire}
              onTap={() => setTouchExpanded((value) => !value)}
              onElement={handleElement}
            />
          ))}
        </AnimatePresence>
      </m.ol>
      <m.button
        type="button"
        tabIndex={showClear ? 0 : -1}
        aria-hidden={!showClear || undefined}
        initial={false}
        animate={{
          opacity: showClear ? 1 : 0,
          y: (showClear ? stack.height + GAP : stack.height - 20) * direction,
        }}
        transition={STACK_SPRING}
        className={cn(
          "dark ui-focus-ring absolute right-0 h-6.5 cursor-pointer rounded-badge bg-popover px-2.5 text-xs font-medium text-popover-foreground ui-lift-float hover:bg-surface-hover",
          direction === 1 ? "top-0" : "bottom-0",
          position.endsWith("center") && "right-1/2 translate-x-1/2",
          !showClear && "pointer-events-none",
        )}
        onClick={clearAll}
      >
        {t("Clear all")}
      </m.button>
      <div ref={polite} aria-live="polite" aria-atomic="true" className="sr-only" />
      <div ref={assertive} aria-live="assertive" aria-atomic="true" className="sr-only" />
    </section>
  );
}
