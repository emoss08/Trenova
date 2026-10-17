import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { clockCaption, clockRemaining, formatCountdown, laneOf } from "./lanes";

const SIZE = 44;
const RADIUS = 19;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

/**
 * How long is left before the item's deadline: a ring that closes over the
 * last four hours, the countdown beside it, and what happens when it runs
 * out. The colour is the lane's: red inside four hours, amber today, blue
 * further out. It moves a step a second rather than animating, so nothing on
 * the card is in motion that the person did not cause.
 */
export function DeadlineClock({
  item,
  minutes,
}: {
  item: Pick<WatchtowerItem, "dueLabel">;
  minutes: number;
}) {
  const t = useT();
  const countdown = formatCountdown(t, minutes);
  const caption = clockCaption(t, item, minutes);

  return (
    <div className={cn("dk-w3-clock", `dk-l-${laneOf(minutes)}`)} role="timer" aria-live="off">
      <span className="dk-w3-cr">
        <svg width={SIZE} height={SIZE} viewBox={`0 0 ${SIZE} ${SIZE}`} aria-hidden fill="none">
          <circle
            cx={SIZE / 2}
            cy={SIZE / 2}
            r={RADIUS}
            stroke="currentColor"
            strokeOpacity={0.18}
            strokeWidth={3}
          />
          <circle
            cx={SIZE / 2}
            cy={SIZE / 2}
            r={RADIUS}
            stroke="currentColor"
            strokeWidth={3}
            strokeLinecap="round"
            strokeDasharray={CIRCUMFERENCE}
            strokeDashoffset={CIRCUMFERENCE * clockRemaining(minutes)}
            transform={`rotate(-90 ${SIZE / 2} ${SIZE / 2})`}
          />
        </svg>
      </span>
      <span>
        <b>{countdown}</b>
        <em>{caption}</em>
      </span>
    </div>
  );
}
