import type { InboundMessageRow } from "@/lib/graphql/inbox";
import { isAwaitingReply, isSnoozed, isWaiting, senderName } from "@/lib/triage/message-state";
import {
  useIsArrival,
  useIsLeaving,
  useIsSelected,
  useTriageStore,
  type TriageRowMode,
} from "@/stores/triage-store";
import { useContextMenuTrigger } from "@trenova/shared/components/context-menu/context-menu-host";
import type { ContextMenuSpec } from "@trenova/shared/components/context-menu/menu-spec";
import {
  CheckIcon,
  ClockIcon,
  PaperclipIcon,
  Send01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, formatUnixTime, getStartOfDay } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import {
  createContext,
  memo,
  useContext,
  useRef,
  useState,
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
} from "react";
import { USER_FORMATTERS, rowLine } from "../action-copy";
import { CLASSIFICATION_DOT, classificationLabel } from "../classification";
import { dueSpan, dueText } from "../due";
import { useMessagePresence, type TriagePresenceMember } from "../use-triage-presence";

/** How far a row must be dragged before letting go acts on it. */
export const SWIPE_COMMIT_PX = 90;
const SWIPE_MAX_PX = 140;
const SWIPE_START_PX = 8;

export type SwipeOutcome = "handle" | "snooze" | null;

/** What a drag that ended at `dx` does: left marks handled, right snoozes. */
export function swipeOutcome(dx: number): SwipeOutcome {
  if (dx <= -SWIPE_COMMIT_PX) {
    return "handle";
  }

  return dx >= SWIPE_COMMIT_PX ? "snooze" : null;
}

/**
 * What every row does, handed down once through context so a row's props are
 * only its message and how it is drawn. Every function is stable.
 */
export type RowHandlers = {
  /** A plain click or Enter opens; Shift extends the selection, ⌘/Ctrl toggles it. */
  open: (id: string) => void;
  toggleSelected: (id: string, extend: boolean) => void;
  swipe: (id: string, outcome: Exclude<SwipeOutcome, null>) => void;
  prefetch: (id: string) => void;
  menu: (id: string) => ContextMenuSpec | null;
};

export const RowHandlersContext = createContext<RowHandlers | null>(null);

function useRowHandlers(): RowHandlers {
  const handlers = useContext(RowHandlersContext);
  if (handlers === null) {
    throw new Error("TriageRow is used outside a list");
  }

  return handlers;
}

function PresenceAvatar({ member }: { member: TriagePresenceMember }) {
  const t = useT();
  const text =
    member.activity === "writing"
      ? t("{0} is writing a reply", member.name)
      : t("{0} is viewing", member.name);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            aria-label={text}
            className="bg-muted text-foreground-muted relative flex size-4 shrink-0 items-center justify-center rounded-full text-3xs font-semibold"
          />
        }
      >
        {getNameInitials(member.name, "?")}
        <span
          aria-hidden
          className="bg-success ring-canvas absolute -right-0.5 -bottom-0.5 size-1.5 rounded-full ring-1"
        />
      </TooltipTrigger>
      <TooltipContent>{text}</TooltipContent>
    </Tooltip>
  );
}

function TimeCell({ message, now }: { message: InboundMessageRow; now: number }) {
  const t = useT();
  if (isSnoozed(message)) {
    return <ClockIcon aria-label={t("Snoozed")} className="text-foreground-subtle size-3" />;
  }
  if (isAwaitingReply(message)) {
    return <Send01Icon aria-label={t("Waiting on them")} className="text-foreground-subtle size-3" />;
  }
  if (isWaiting(message) && message.dueAt !== null) {
    const span = dueSpan(message.dueAt, now);
    return (
      <span
        className={cn(
          "font-mono text-xs tabular-nums",
          span.state === "over" && "text-danger",
          span.state === "soon" && "text-warning",
          span.state === "later" && "text-foreground-subtle",
        )}
      >
        {dueText(t, span)}
      </span>
    );
  }
  const today = getStartOfDay(new Date(now * 1000));

  return (
    <span className="text-foreground-subtle font-mono text-xs tabular-nums">
      {message.receivedAt >= today
        ? formatUnixTime(message.receivedAt)
        : formatUnixMonthDay(message.receivedAt)}
    </span>
  );
}

/**
 * One message in a folder, read at a glance: its kind, who sent it, how much
 * of a thread it heads, who else has it open, whose it is, how it stands
 * against its target, the subject, and the next thing to do with it.
 *
 * The kind's dot turns into a checkbox under the pointer or while anything is
 * selected. On touch a row swipes: left marks it handled, right snoozes it
 * until tomorrow. A right-click or a long press opens its menu.
 */
export const TriageRow = memo(function TriageRow({
  message,
  active,
  mode,
  now,
}: {
  message: InboundMessageRow;
  active: boolean;
  mode: TriageRowMode;
  now: number;
}) {
  const t = useT();
  const handlers = useRowHandlers();
  const picked = useIsSelected(message.id);
  const selecting = useTriageStore((state) => state.selected.size > 0);
  const leaving = useIsLeaving(message.id);
  const arriving = useIsArrival(message.id);
  const presence = useMessagePresence(message.id);
  const trigger = useContextMenuTrigger(() => handlers.menu(message.id));
  const [dx, setDx] = useState(0);
  const drag = useRef<{ x: number; y: number; moving: boolean } | null>(null);
  const swiped = useRef(0);

  const need = message.needsReview;
  const classification = message.classification ?? "Other";
  const line = mode === "next" ? rowLine(t, message, USER_FORMATTERS) : null;
  const outcome = swipeOutcome(dx);

  const onPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    trigger.onPointerDown(event);
    if (need && !selecting && event.button === 0) {
      drag.current = { x: event.clientX, y: event.clientY, moving: false };
    }
  };
  const onPointerMove = (event: PointerEvent<HTMLDivElement>) => {
    trigger.onPointerMove(event);
    const current = drag.current;
    if (current === null) {
      return;
    }
    const distance = event.clientX - current.x;
    if (
      !current.moving &&
      Math.abs(distance) > SWIPE_START_PX &&
      Math.abs(distance) > Math.abs(event.clientY - current.y)
    ) {
      current.moving = true;
      event.currentTarget.setPointerCapture(event.pointerId);
    }
    if (current.moving) {
      setDx(Math.max(-SWIPE_MAX_PX, Math.min(SWIPE_MAX_PX, distance)));
    }
  };
  const endDrag = (event: PointerEvent<HTMLDivElement>) => {
    trigger.onPointerUp(event);
    const current = drag.current;
    drag.current = null;
    if (current === null || !current.moving) {
      return;
    }
    swiped.current = Date.now();
    const result = swipeOutcome(dx);
    setDx(0);
    if (result !== null) {
      handlers.swipe(message.id, result);
    }
  };

  const onClick = (event: MouseEvent<HTMLDivElement>) => {
    if (Date.now() - swiped.current < 300) {
      return;
    }
    if (event.shiftKey || event.metaKey || event.ctrlKey || selecting) {
      handlers.toggleSelected(message.id, event.shiftKey);
      return;
    }
    handlers.open(message.id);
  };

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    trigger.onKeyDown(event);
    if (event.defaultPrevented) {
      return;
    }
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      handlers.open(message.id);
    }
  };

  return (
    <div
      data-leaving={leaving || undefined}
      className={cn(
        "relative mx-1.5 overflow-hidden rounded-lg transition-[opacity,transform] duration-240 ease-swift",
        "data-leaving:-translate-x-3 data-leaving:opacity-0",
        arriving && "animate-arrive",
      )}
    >
      {dx !== 0 && (
        <>
          <span
            aria-hidden
            className={cn(
              "absolute inset-y-0 left-0 flex w-1/2 items-center gap-1.5 px-4.5 text-sm font-medium",
              outcome === "snooze" ? "bg-warning text-warning-on-solid" : "bg-sunken text-foreground-subtle",
            )}
          >
            <ClockIcon className="size-3.5" />
            {t("Snooze")}
          </span>
          <span
            aria-hidden
            className={cn(
              "absolute inset-y-0 right-0 flex w-1/2 items-center justify-end gap-1.5 px-4.5 text-sm font-medium",
              outcome === "handle" ? "bg-success text-foreground-on-solid" : "bg-sunken text-foreground-subtle",
            )}
          >
            {t("Handled")}
            <CheckIcon className="size-3.5" />
          </span>
        </>
      )}
      <div
        role="option"
        tabIndex={active ? 0 : -1}
        aria-selected={picked}
        aria-current={active || undefined}
        data-message-id={message.id}
        onClick={onClick}
        onKeyDown={onKeyDown}
        onPointerEnter={() => handlers.prefetch(message.id)}
        onFocus={() => handlers.prefetch(message.id)}
        onContextMenu={trigger.onContextMenu}
        onClickCapture={trigger.onClickCapture}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={endDrag}
        onPointerCancel={(event) => {
          trigger.onPointerCancel(event);
          drag.current = null;
          setDx(0);
        }}
        style={dx === 0 ? undefined : { transform: `translateX(${dx}px)`, transition: "none" }}
        className={cn(
          "group/row ui-inset-focus-ring bg-canvas relative z-1 grid cursor-pointer touch-pan-y grid-cols-[minmax(0,1fr)_auto] gap-x-2.5 gap-y-px rounded-lg px-3 py-2.5 transition-[transform,background-color] select-none",
          "duration-260 ease-spring",
          active ? "bg-nav-active" : picked ? "bg-surface-selected" : "hover:bg-surface-hover",
        )}
      >
        {arriving && (
          <span
            aria-hidden
            className="bg-brand-subtle animate-arrive-glow pointer-events-none absolute inset-0 rounded-lg"
          />
        )}
        <span className="relative flex min-w-0 items-center gap-2 text-sm">
          <span
            aria-hidden
            onClick={(event) => {
              event.stopPropagation();
              handlers.toggleSelected(message.id, event.shiftKey);
            }}
            className="flex size-3.5 shrink-0 items-center justify-center"
          >
            <span
              title={classificationLabel(t, classification)}
              className={cn(
                "size-1.75 rounded-full",
                CLASSIFICATION_DOT[classification],
                !need && "opacity-35",
                (picked || selecting) && "hidden",
                "group-hover/row:hidden",
              )}
            />
            <span
              className={cn(
                "border-border-strong bg-field size-3.5 items-center justify-center rounded-sm border",
                picked ? "bg-brand border-brand text-foreground-on-solid flex" : "hidden",
                (picked || selecting) && "flex",
                "group-hover/row:flex",
              )}
            >
              {picked && <CheckIcon className="size-2.5" strokeWidth={3} />}
            </span>
          </span>
          <span
            className={cn(
              "truncate",
              need ? "text-foreground font-semibold" : "text-foreground-muted font-medium",
            )}
          >
            {senderName(message)}
          </span>
          {arriving && (
            <Badge variant="brand" className="h-4 px-1.5 text-3xs">
              {t("New")}
            </Badge>
          )}
          {message.threadCount > 1 && (
            <span className="bg-surface-hover text-foreground-subtle shrink-0 rounded-full px-1.5 font-mono text-2xs tabular-nums">
              {message.threadCount}
            </span>
          )}
          {message.attachmentCount > 0 && (
            <PaperclipIcon
              aria-label={t("{0, plural, one {# attachment} other {# attachments}}", message.attachmentCount)}
              className="text-foreground-subtle size-3 shrink-0"
            />
          )}
          {presence.map((member) => (
            <PresenceAvatar key={member.userId} member={member} />
          ))}
        </span>
        <span className="relative flex items-center justify-end gap-2">
          {need && message.owner !== null && (
            <span
              title={t("Assigned to {0}", message.owner.name)}
              className="bg-muted text-foreground-muted flex size-4 items-center justify-center rounded-full text-3xs font-semibold"
            >
              {getNameInitials(message.owner.name, "?")}
            </span>
          )}
          <TimeCell message={message} now={now} />
        </span>
        <span
          className={cn(
            "relative col-span-2 truncate pl-5.5 text-base",
            need ? "text-foreground" : "text-foreground-muted",
          )}
        >
          {message.subject === "" ? t("(no subject)") : message.subject}
        </span>
        <span className="relative col-span-2 flex min-w-0 pl-5.5 text-sm">
          {line !== null ? (
            <span className="text-foreground-subtle truncate">{line.text}</span>
          ) : (
            <span className="text-foreground-subtle truncate">{message.preview}</span>
          )}
        </span>
      </div>
    </div>
  );
});
