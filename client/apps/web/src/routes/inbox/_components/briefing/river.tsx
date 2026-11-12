import type { TriageBriefing } from "@/lib/graphql/inbox";
import { senderName } from "@/lib/triage/message-state";
import { CheckIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useMemo, useState, type CSSProperties } from "react";
import { CLASSIFICATION_DOT } from "../classification";
import { dueSpan, dueText } from "../due";
import {
  riverTicks,
  riverWindow,
  riverX,
  tipAlign,
  yourDotState,
  type RiverPoint,
} from "./briefing-model";

type RiverProps = {
  briefing: Pick<TriageBriefing, "dayStart" | "novaPoints" | "yours">;
  now: number;
  onOpenMessage: (id: string) => void;
};

/** Nova's dots sit at slightly different heights so a busy hour reads as a cluster, not a line. */
function novaDotStyle(x: number, index: number): CSSProperties {
  return {
    left: `${x + (index % 3) * 0.4}%`,
    top: `${8 + (index % 4) * 4}px`,
  };
}

/**
 * The day so far, from six in the morning to now: what Nova handled on the
 * top lane, and what was left for you below, coloured by kind, ringed amber
 * once over its target and checked once done. Hovering a dot says what it
 * is, under the dot and inside the river; clicking it opens the message.
 */
export const River = memo(function River({ briefing, now, onOpenMessage }: RiverProps) {
  const t = useT();
  const [hovered, setHovered] = useState<string | null>(null);
  const window = useMemo(() => riverWindow(briefing.dayStart, now), [briefing.dayStart, now]);
  const ticks = useMemo(() => riverTicks(window, briefing.dayStart), [window, briefing.dayStart]);
  const tip = hovered === null ? null : briefing.yours.find((point) => point.messageId === hovered);

  return (
    <div
      className="relative pl-(--river-pad) [--river-pad:3rem] @max-2xl:[--river-pad:0px]"
      aria-label={t("Today so far")}
      role="group"
    >
      <div aria-hidden className="relative mb-1.5 h-3.5">
        {ticks.map((tick) => (
          <span
            key={tick.at}
            className="absolute -translate-x-1/2 text-2xs text-dsk-faint tabular-nums"
            style={{ left: `${tick.x}%` }}
          >
            {formatUnixInUserTimezone(tick.at, { hour: "numeric" }, "")}
          </span>
        ))}
        <span
          className="absolute -translate-x-1/2 text-2xs text-dsk-subtle"
          style={{ left: `${riverX(now, window)}%` }}
        >
          {t("now")}
        </span>
      </div>

      <div className="relative mb-1.5 h-8 rounded-[6px] border border-dsk-b-sub bg-dsk-card">
        <span className="absolute top-1/2 -left-12 -translate-y-1/2 text-xs text-dsk-subtle @max-2xl:sr-only">
          {t("Nova")}
        </span>
        {briefing.novaPoints.map((point, index) => (
          <i
            key={point.messageId}
            aria-hidden
            className="absolute -ml-0.75 size-1.5 rounded-full bg-dsk-agent"
            style={novaDotStyle(riverX(point.receivedAt, window), index)}
          />
        ))}
      </div>

      <div className="relative h-8 rounded-[6px] border border-dsk-b-sub bg-dsk-card">
        <span className="absolute top-1/2 -left-12 -translate-y-1/2 text-xs text-dsk-subtle @max-2xl:sr-only">
          {t("You")}
        </span>
        {briefing.yours.map((point) => (
          <YourDot
            key={point.messageId}
            point={point}
            x={riverX(point.receivedAt, window)}
            now={now}
            onHover={setHovered}
            onOpen={onOpenMessage}
          />
        ))}
        {tip !== null && tip !== undefined && (
          <RiverTip point={tip} x={riverX(tip.receivedAt, window)} now={now} />
        )}
      </div>

      <span
        aria-hidden
        className="pointer-events-none absolute top-4.5 bottom-0 w-px -translate-x-1/2 bg-dsk-b-strong"
        style={{
          left: `calc(var(--river-pad) + (100% - var(--river-pad)) * ${riverX(now, window) / 100})`,
        }}
      />
    </div>
  );
});

const YourDot = memo(function YourDot({
  point,
  x,
  now,
  onHover,
  onOpen,
}: {
  point: RiverPoint;
  x: number;
  now: number;
  onHover: (id: string | null) => void;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  const state = yourDotState(point, now);
  const due =
    state === "done"
      ? t("done")
      : point.dueAt === null
        ? ""
        : dueText(t, dueSpan(point.dueAt, now));

  return (
    <button
      type="button"
      aria-label={t("{0}: {1}, {2}", senderName(point), point.subject || t("(no subject)"), due)}
      onPointerEnter={() => onHover(point.messageId)}
      onPointerLeave={() => onHover(null)}
      onFocus={() => onHover(point.messageId)}
      onBlur={() => onHover(null)}
      onClick={() => onOpen(point.messageId)}
      className="group absolute top-1/2 -mt-1.5 -ml-1.5 size-3 rounded-full ui-focus-ring"
      style={{ left: `${x}%` }}
    >
      <span
        className={cn(
          "flex size-full items-center justify-center rounded-full ring-2 ring-dsk-card transition-transform duration-200 ease-spring group-hover:scale-150 group-focus-visible:scale-150",
          state === "done"
            ? "bg-success text-foreground-on-solid opacity-70"
            : CLASSIFICATION_DOT[point.classification ?? "Other"],
        )}
      >
        {state === "done" && <CheckIcon aria-hidden className="size-2" strokeWidth={3.4} />}
      </span>
      {state === "over" && (
        <span
          aria-hidden
          className="pointer-events-none absolute -inset-1 rounded-full border border-dsk-warn"
        />
      )}
    </button>
  );
});

function RiverTip({ point, x, now }: { point: RiverPoint; x: number; now: number }) {
  const t = useT();
  const align = tipAlign(x);
  const state = yourDotState(point, now);

  return (
    <div
      className="pointer-events-none absolute top-[calc(100%+6px)] z-40"
      style={{ left: `${x}%` }}
    >
      <div
        role="tooltip"
        className={cn(
          "relative flex max-w-70 animate-pop-in flex-col gap-px rounded-[8px] border border-dsk-b bg-dsk-raised px-2.75 py-2 text-dsk-base text-dsk-fg ui-lift-float",
          align === "center" && "-translate-x-1/2",
          align === "start" && "-translate-x-3.5",
          align === "end" && "-translate-x-[calc(100%-0.875rem)]",
        )}
      >
        <span
          aria-hidden
          className={cn(
            "absolute -top-1.25 size-2 rotate-45 border-t border-l border-dsk-b bg-dsk-raised",
            align === "center" && "left-1/2 -ml-1",
            align === "start" && "left-2.5",
            align === "end" && "right-2.5",
          )}
        />
        <b className="truncate font-medium">{senderName(point)}</b>
        <span className="truncate text-dsk-subtle">
          {point.subject.trim() === "" ? t("(no subject)") : point.subject}
        </span>
        <em
          className={cn(
            "text-xs not-italic tabular-nums",
            state === "over" ? "text-dsk-warn-fg" : "text-dsk-subtle",
          )}
        >
          {state === "done"
            ? t("done")
            : point.dueAt === null
              ? t("no target")
              : dueText(t, dueSpan(point.dueAt, now))}
        </em>
      </div>
    </div>
  );
}
