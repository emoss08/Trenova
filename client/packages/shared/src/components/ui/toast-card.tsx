import {
  CheckIcon,
  ChevronDownIcon,
  Copy01Icon,
  Loading02Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ToastErrorIcon } from "@trenova/shared/components/ui/toast-icons/toast-error-icon";
import { ToastInfoIcon } from "@trenova/shared/components/ui/toast-icons/toast-info-icon";
import { ToastSuccessIcon } from "@trenova/shared/components/ui/toast-icons/toast-success-icon";
import { ToastWarningIcon } from "@trenova/shared/components/ui/toast-icons/toast-warning-icon";
import type { ToastSlot } from "@trenova/shared/components/ui/toast-stack";
import type {
  ToastAction,
  ToastDetail,
  ToastId,
  ToastRecord,
  ToastType,
} from "@trenova/shared/components/ui/toast-store";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  animate,
  m,
  useMotionValue,
  useReducedMotionConfig,
  useTransform,
  type Transition,
} from "motion/react";
import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
  type ReactNode,
  type RefObject,
} from "react";

export type SwipeDirection = "left" | "right" | "both";

export type ToastCardProps = {
  toast: ToastRecord;
  slot: ToastSlot;
  index: number;
  direction: 1 | -1;
  paused: boolean;
  swipe: SwipeDirection;
  onHeight: (id: ToastId, height: number) => void;
  onAnnounce: (text: string, assertive: boolean) => void;
  /** The person closed it: a button, Escape or a swipe. */
  onDismiss: (id: ToastId) => void;
  /** Its time ran out. */
  onExpire: (id: ToastId) => void;
  /** A tap that was not a swipe, on a touch screen, opens or closes the stack. */
  onTap: () => void;
  onElement: (id: ToastId, element: HTMLLIElement | null) => void;
};

const STACK_SPRING: Transition = {
  type: "spring",
  stiffness: 400,
  damping: 34,
  mass: 0.9,
  opacity: { duration: 0.2, ease: "easeOut" },
};
const SWIPE_DISTANCE = 96;
const SWIPE_VELOCITY = 600;
const DRAG_SLOP = 6;
const RUBBER_BAND = 2.4;
const COPIED_MS = 1600;
const ENTER_OFFSET = 24;
const EXIT_OFFSET = 12;

type ExitState = { y: number; scale: number; direction: 1 | -1; reduced: boolean };

const cardVariants = {
  exit: ({ y, scale, direction, reduced }: ExitState) => ({
    opacity: 0,
    y: reduced ? y : y - direction * EXIT_OFFSET,
    scale: reduced ? scale : scale * 0.94,
    transition: { duration: 0.2, ease: [0.4, 0, 1, 1] as const },
  }),
};

function ToastIcon({ type }: { type: ToastType }) {
  switch (type) {
    case "success":
      return <ToastSuccessIcon />;
    case "error":
      return <ToastErrorIcon />;
    case "warning":
      return <ToastWarningIcon />;
    case "info":
      return <ToastInfoIcon />;
    case "loading":
      return <Loading02Icon className="size-4 animate-spin" />;
    default:
      return null;
  }
}

const ICON_TONE: Record<ToastType, string> = {
  default: "",
  success: "text-success",
  error: "text-danger",
  warning: "text-warning",
  info: "text-info",
  loading: "text-foreground-muted",
};

function squash(text: string | null | undefined): string {
  return text?.replace(/\s+/g, " ").trim() ?? "";
}

/** What a screen reader hears: the title, then the description, without the count badge. */
function announcementOf(content: HTMLElement | null): string {
  if (!content) {
    return "";
  }
  return Array.from(content.querySelectorAll<HTMLElement>("[data-toast-text]"))
    .map((part) => squash(part.textContent))
    .filter(Boolean)
    .join(". ");
}

/**
 * The toast's clock: it runs only while nothing holds it, and starts over when the toast
 * changes or repeats. The hairline along the bottom is drawn from the same pauses.
 */
function useToastTimer({
  duration,
  paused,
  restartKey,
  bar,
  showBar,
  onElapse,
}: {
  duration: number;
  paused: boolean;
  restartKey: string;
  bar: RefObject<HTMLSpanElement | null>;
  showBar: boolean;
  onElapse: () => void;
}) {
  const remaining = useRef(duration);
  const runningSince = useRef<number | null>(null);
  const animation = useRef<Animation | null>(null);
  const elapse = useLatestCallback(onElapse);
  const finite = Number.isFinite(duration);

  useEffect(() => {
    remaining.current = duration;
  }, [duration, restartKey]);

  useEffect(() => {
    const element = bar.current;
    if (!finite || !showBar || !element || typeof element.animate !== "function") {
      return;
    }
    const run = element.animate([{ transform: "scaleX(1)" }, { transform: "scaleX(0)" }], {
      duration,
      easing: "linear",
      fill: "forwards",
    });
    // Cancelling on unmount rejects `finished`; nothing waits on it, so the rejection is expected.
    void run.finished.catch(() => undefined);
    const since = runningSince.current;
    const left = remaining.current - (since === null ? 0 : Date.now() - since);
    run.currentTime = Math.min(duration, Math.max(0, duration - left));
    if (since === null) {
      run.pause();
    }
    animation.current = run;
    return () => {
      run.cancel();
      animation.current = null;
    };
  }, [bar, duration, finite, restartKey, showBar]);

  useEffect(() => {
    if (paused || !finite) {
      return;
    }
    const startedAt = Date.now();
    runningSince.current = startedAt;
    animation.current?.play();
    const handle = window.setTimeout(elapse, Math.max(0, remaining.current));
    return () => {
      window.clearTimeout(handle);
      animation.current?.pause();
      runningSince.current = null;
      remaining.current -= Date.now() - startedAt;
    };
  }, [duration, elapse, finite, paused, restartKey]);
}

function ToastDetails({ details }: { details: readonly ToastDetail[] }) {
  const t = useT();
  const [copied, setCopied] = useState(false);
  const list = useRef<HTMLDListElement>(null);

  useEffect(() => {
    if (!copied) {
      return;
    }
    const handle = window.setTimeout(() => setCopied(false), COPIED_MS);
    return () => window.clearTimeout(handle);
  }, [copied]);

  const copy = async (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    const text = details.map((detail) => `${detail.label}: ${detail.value}`).join("\n");
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      const selection = window.getSelection();
      if (list.current && selection) {
        const range = document.createRange();
        range.selectNodeContents(list.current);
        selection.removeAllRanges();
        selection.addRange(range);
      }
    }
  };

  return (
    <m.div
      data-toast-details=""
      initial={{ opacity: 0, y: -4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: "easeOut" }}
      className="mt-2.5 grid grid-cols-[minmax(0,1fr)_auto] items-start gap-2 rounded-md bg-muted/60 p-2.5 ring-1 ring-foreground/10 select-text"
    >
      <dl ref={list} className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-2.5 gap-y-1 text-xs">
        {details.map((detail) => (
          <div key={detail.label} className="contents">
            <dt className="text-foreground-muted">{detail.label}</dt>
            <dd className={cn("break-all", detail.monospace && "font-mono")}>{detail.value}</dd>
          </div>
        ))}
      </dl>
      <Button type="button" variant="secondary" size="xs" onClick={(event) => void copy(event)}>
        {copied ? <CheckIcon className="size-3" /> : <Copy01Icon className="size-3" />}
        {copied ? t("Copied") : t("Copy details")}
      </Button>
    </m.div>
  );
}

function ActionButton({
  action,
  variant,
  onDone,
}: {
  action: ToastAction;
  variant: "default" | "secondary";
  onDone: () => void;
}) {
  return (
    <Button
      type="button"
      variant={variant}
      size="xs"
      onClick={(event) => {
        event.stopPropagation();
        action.onClick(event);
        if (!event.defaultPrevented) {
          onDone();
        }
      }}
    >
      {action.label}
    </Button>
  );
}

function CountBadge({ count }: { count: number }) {
  return (
    <m.span
      key={count}
      aria-hidden="true"
      initial={{ scale: 1.35 }}
      animate={{ scale: 1 }}
      transition={{ type: "spring", stiffness: 600, damping: 18 }}
      className="inline-flex h-4 shrink-0 items-center rounded-badge bg-muted px-1.5 font-mono text-2xs tabular-nums ring-1 ring-foreground/10"
    >
      ×{count}
    </m.span>
  );
}

function ToastContent({
  toast,
  onDismiss,
  contentRef,
}: {
  toast: ToastRecord;
  onDismiss: () => void;
  contentRef: RefObject<HTMLDivElement | null>;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const { type, title, description, action, cancel, details, revision, count } = toast;
  const actionCount = (action ? 1 : 0) + (cancel ? 1 : 0);
  const inline = !description && !details && actionCount <= 1;

  const actions: ReactNode =
    actionCount > 0 || details ? (
      <div className={cn("flex flex-wrap items-center gap-1.5", !inline && "mt-2.5")}>
        {action ? <ActionButton action={action} variant="default" onDone={onDismiss} /> : null}
        {cancel ? <ActionButton action={cancel} variant="secondary" onDone={onDismiss} /> : null}
        {details ? (
          <Button
            type="button"
            variant="quiet"
            size="xs"
            aria-expanded={open}
            onClick={(event) => {
              event.stopPropagation();
              setOpen((value) => !value);
            }}
          >
            {open ? t("Hide details") : t("Details")}
            <ChevronDownIcon
              className={cn("size-3 transition-transform duration-200", open && "rotate-180")}
            />
          </Button>
        ) : null}
      </div>
    ) : null;

  return (
    <div className="flex items-start gap-2.5 py-3 pr-2.5 pl-3.5">
      {type !== "default" ? (
        <m.span
          key={type}
          aria-hidden="true"
          initial={revision === 0 ? false : { scale: 0.6, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ type: "spring", stiffness: 500, damping: 24 }}
          className={cn("flex h-4.5 w-4 shrink-0 items-center justify-center", ICON_TONE[type])}
        >
          <ToastIcon type={type} />
        </m.span>
      ) : null}
      <div className="min-w-0 flex-1">
        <m.div
          key={revision}
          ref={contentRef}
          initial={revision === 0 ? false : { opacity: 0, y: 3, filter: "blur(3px)" }}
          animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
          transition={{ duration: 0.28, ease: [0.2, 0.8, 0.2, 1] }}
          className="flex flex-col gap-0.5 text-sm"
        >
          <div className="flex min-w-0 items-center gap-1.5 leading-4.5 font-medium break-words">
            <span data-toast-text="" className="min-w-0">
              {title}
            </span>
            {count > 1 ? <CountBadge count={count} /> : null}
          </div>
          {description ? (
            <div data-toast-text="" className="leading-4.5 break-words text-foreground-muted">
              {description}
            </div>
          ) : null}
        </m.div>
        {inline ? null : actions}
        {details && open ? <ToastDetails details={details} /> : null}
      </div>
      {inline ? <div className="-mt-0.5 shrink-0">{actions}</div> : null}
      <Button
        type="button"
        variant="quiet"
        size="icon-xs"
        aria-label={t("Dismiss")}
        className="-mt-0.5 -mr-0.5 shrink-0 opacity-0 transition-opacity group-hover/toast:opacity-100 group-focus-within/toast:opacity-100 focus-visible:opacity-100 [@media(hover:none)]:opacity-70"
        onClick={(event) => {
          event.stopPropagation();
          onDismiss();
        }}
      >
        <XCloseIcon className="size-3.5" />
      </Button>
    </div>
  );
}

export function ToastCard({
  toast,
  slot,
  index,
  direction,
  paused,
  swipe,
  onHeight,
  onAnnounce,
  onDismiss,
  onExpire,
  onTap,
  onElement,
}: ToastCardProps) {
  const reduced = useReducedMotionConfig() ?? false;
  const measured = useRef<HTMLDivElement>(null);
  const textRef = useRef<HTMLDivElement>(null);
  const bar = useRef<HTMLSpanElement>(null);
  const [holding, setHolding] = useState(false);
  const x = useMotionValue(0);
  const dragOpacity = useTransform(x, (value) => 1 - Math.min(Math.abs(value) / 320, 0.55));
  const drag = useRef<{
    pointerId: number;
    pointerType: string;
    startX: number;
    lastX: number;
    lastAt: number;
    velocity: number;
    active: boolean;
  } | null>(null);
  const { id, revision, count, type, urgent, render } = toast;
  const report = useLatestCallback(onHeight);
  const announce = useLatestCallback(onAnnounce);

  useLayoutEffect(() => {
    const element = measured.current;
    if (!element) {
      return;
    }
    report(id, element.offsetHeight);
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    const observer = new ResizeObserver(() => report(id, element.offsetHeight));
    observer.observe(element);
    return () => observer.disconnect();
  }, [id, report]);

  useEffect(() => {
    const text = render ? squash(measured.current?.textContent) : announcementOf(textRef.current);
    if (text) {
      announce(text, urgent || type === "error");
    }
  }, [announce, count, render, revision, type, urgent]);

  useToastTimer({
    duration: toast.duration,
    paused: paused || holding,
    restartKey: `${revision}:${count}`,
    bar,
    showBar: !reduced && !render,
    onElapse: () => onExpire(id),
  });

  const allowed = (offset: number) =>
    swipe === "both" || (swipe === "right" ? offset > 0 : offset < 0);

  const handlePointerDown = (event: PointerEvent<HTMLLIElement>) => {
    if (event.button !== 0 || slot.hidden) {
      return;
    }
    const target = event.target as HTMLElement;
    if (target.closest("button, a, input, textarea, select, [data-toast-details]")) {
      return;
    }
    event.currentTarget.setPointerCapture(event.pointerId);
    drag.current = {
      pointerId: event.pointerId,
      pointerType: event.pointerType,
      startX: event.clientX,
      lastX: event.clientX,
      lastAt: event.timeStamp,
      velocity: 0,
      active: false,
    };
    setHolding(true);
  };

  const handlePointerMove = (event: PointerEvent<HTMLLIElement>) => {
    const state = drag.current;
    if (!state || event.pointerId !== state.pointerId) {
      return;
    }
    const raw = event.clientX - state.startX;
    if (!state.active) {
      if (Math.abs(raw) < DRAG_SLOP) {
        return;
      }
      state.active = true;
    }
    const elapsed = Math.max(1, event.timeStamp - state.lastAt);
    const instant = ((event.clientX - state.lastX) / elapsed) * 1000;
    state.velocity = state.velocity * 0.6 + instant * 0.4;
    state.lastX = event.clientX;
    state.lastAt = event.timeStamp;
    x.set(allowed(raw) ? raw : Math.sign(raw) * Math.sqrt(Math.abs(raw)) * RUBBER_BAND);
  };

  const settleDrag = (event: PointerEvent<HTMLLIElement>, cancelled: boolean) => {
    const state = drag.current;
    if (!state || event.pointerId !== state.pointerId) {
      return;
    }
    drag.current = null;
    setHolding(false);
    if (!state.active) {
      if (!cancelled && state.pointerType !== "mouse") {
        onTap();
      }
      return;
    }
    const offset = x.get();
    const flung =
      !cancelled &&
      allowed(offset) &&
      (Math.abs(offset) > SWIPE_DISTANCE ||
        (Math.abs(state.velocity) > SWIPE_VELOCITY &&
          Math.sign(state.velocity) === Math.sign(offset)));
    if (flung) {
      const width = event.currentTarget.offsetWidth;
      void animate(x, Math.sign(offset) * (width + 64), {
        type: "spring",
        stiffness: 260,
        damping: 32,
        velocity: state.velocity,
        onComplete: () => onDismiss(id),
      });
      return;
    }
    void animate(x, 0, { type: "spring", stiffness: 520, damping: 34, velocity: state.velocity });
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLLIElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      onDismiss(id);
    }
  };

  const exit: ExitState = { y: slot.y, scale: slot.scale, direction, reduced };

  return (
    <m.li
      ref={(element: HTMLLIElement | null) => onElement(id, element)}
      data-slot="toast"
      data-type={type}
      tabIndex={slot.hidden ? -1 : 0}
      aria-hidden={slot.hidden || undefined}
      inert={slot.hidden}
      custom={exit}
      variants={cardVariants}
      initial={{ opacity: 0, y: reduced ? slot.y : -direction * ENTER_OFFSET, scale: 0.92 }}
      animate={{
        opacity: slot.hidden ? 0 : 1,
        y: slot.y,
        scale: slot.scale,
        height: slot.height > 0 ? slot.height : "auto",
      }}
      exit="exit"
      transition={STACK_SPRING}
      style={{
        zIndex: 1000 - index,
        transformOrigin: direction === 1 ? "50% 100%" : "50% 0%",
      }}
      className={cn(
        "group/toast ui-focus-ring absolute inset-x-0 touch-pan-y rounded-lg outline-none select-none",
        direction === 1 ? "top-0" : "bottom-0",
        !render && "dark",
      )}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={(event) => settleDrag(event, false)}
      onPointerCancel={(event) => settleDrag(event, true)}
      onLostPointerCapture={(event) => settleDrag(event, true)}
      onKeyDown={handleKeyDown}
    >
      <m.div
        style={{ x, opacity: dragOpacity }}
        className={cn(
          "relative h-full rounded-lg",
          !render && "overflow-hidden bg-popover text-popover-foreground ui-lift-float",
        )}
      >
        <div
          ref={measured}
          className={cn(
            "transition-opacity duration-200",
            slot.behind && "pointer-events-none opacity-0",
          )}
        >
          {render ? (
            render(id)
          ) : (
            <ToastContent toast={toast} onDismiss={() => onDismiss(id)} contentRef={textRef} />
          )}
        </div>
        {Number.isFinite(toast.duration) && !render ? (
          <span
            ref={bar}
            aria-hidden="true"
            className={cn(
              "pointer-events-none absolute inset-x-0 bottom-0 h-0.5 origin-left bg-foreground/20",
              (reduced || slot.behind) && "hidden",
            )}
          />
        ) : null}
      </m.div>
    </m.li>
  );
}
