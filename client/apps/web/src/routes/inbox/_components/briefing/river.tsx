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
    top: `${8 + (index % 4) * 5}px`,
    animationDelay: `${index * 22}ms`,
  };
}

/**
 * The day so far, from six in the morning to now: what Nova handled on the
 * top lane, and what was left for you below, coloured by kind, ringed amber
 * once over its target and checked once done. Hovering a dot says what it
 * is, under the dot and inside the river; clicking it opens the message.
 * Dots are placed once and only their transform and opacity move.
 */
export const River = memo(function River({ briefing, now, onOpenMessage }: RiverProps) {
  const t = useT();
  const [hovered, setHovered] = useState<string | null>(null);
  const window = useMemo(() => riverWindow(briefing.dayStart, now), [briefing.dayStart, now]);
  const ticks = useMemo(() => riverTicks(window, briefing.dayStart), [window, briefing.dayStart]);
  const tip = hovered === null ? null : briefing.yours.find((point) => point.messageId === hovered);

  return (
    <div className="relative pt-5.5 pb-1 pl-14" aria-label={t("Today so far")} role="group">
      <div aria-hidden className="relative mb-1.5 h-3.5">
        {ticks.map((tick) => (
          <span
            key={tick.at}
            className="text-2xs text-foreground-subtle absolute -translate-x-1/2 font-mono"
            style={{ left: `${tick.x}%` }}
          >
            {formatUnixInUserTimezone(tick.at, { hour: "numeric" }, "")}
          </span>
        ))}
        <span
          className="text-2xs text-foreground-subtle absolute -translate-x-1/2 font-mono"
          style={{ left: `${riverX(now, window)}%` }}
        >
          {t("now")}
        </span>
      </div>

      <div className="tb-lane-nova relative mb-1.5 h-8 rounded-full">
        <span className="text-foreground-subtle absolute top-1/2 -left-14 -translate-y-1/2 text-xs">
          {t("Nova")}
        </span>
        {briefing.novaPoints.map((point, index) => (
          <i
            key={point.messageId}
            aria-hidden
            className="tb-nova-dot absolute -ml-0.75 size-1.5 rounded-full"
            style={novaDotStyle(riverX(point.receivedAt, window), index)}
          />
        ))}
      </div>

      <div className="bg-surface-hover relative h-9 rounded-full">
        <span className="text-foreground-subtle absolute top-1/2 -left-14 -translate-y-1/2 text-xs">
          {t("You")}
        </span>
        {briefing.yours.map((point, index) => (
          <YourDot
            key={point.messageId}
            point={point}
            x={riverX(point.receivedAt, window)}
            index={index}
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
        className="bg-foreground pointer-events-none absolute top-9 bottom-0 w-px -translate-x-1/2 opacity-50"
        style={{ left: `calc(3.5rem + (100% - 3.5rem) * ${riverX(now, window) / 100})` }}
      />
    </div>
  );
});

const YourDot = memo(function YourDot({
  point,
  x,
  index,
  now,
  onHover,
  onOpen,
}: {
  point: RiverPoint;
  x: number;
  index: number;
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
      className="tb-your-dot ui-focus-ring group absolute top-1/2 -mt-1.75 -ml-1.75 size-3.5 rounded-full"
      style={{ left: `${x}%`, animationDelay: `${300 + index * 50}ms` }}
    >
      <span
        className={cn(
          "ring-canvas flex size-full items-center justify-center rounded-full ring-3 transition-transform duration-200 ease-spring group-hover:scale-150 group-focus-visible:scale-150",
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
          className="tb-over-ring border-warning pointer-events-none absolute -inset-1.25 rounded-full border-[1.5px]"
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
      className="dark pointer-events-none absolute top-[calc(100%+6px)] z-40"
      style={{ left: `${x}%` }}
    >
      <div
        role="tooltip"
        className={cn(
          "animate-pop-in bg-raised text-foreground border-border-strong ui-lift-float relative flex max-w-70 flex-col gap-px rounded-lg border px-2.75 py-2 text-sm",
          align === "center" && "-translate-x-1/2",
          align === "start" && "-translate-x-3.5",
          align === "end" && "-translate-x-[calc(100%-0.875rem)]",
        )}
      >
        <span
          aria-hidden
          className={cn(
            "bg-raised border-border-strong absolute -top-1.25 size-2 rotate-45 border-t border-l",
            align === "center" && "left-1/2 -ml-1",
            align === "start" && "left-2.5",
            align === "end" && "right-2.5",
          )}
        />
        <b className="truncate font-semibold">{senderName(point)}</b>
        <span className="text-foreground-subtle truncate">
          {point.subject.trim() === "" ? t("(no subject)") : point.subject}
        </span>
        <em
          className={cn(
            "text-2xs font-mono not-italic",
            state === "over" ? "text-warning" : "text-foreground-subtle",
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
