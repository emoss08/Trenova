import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState } from "react";

/**
 * Seconds left before something happens again, as a ring that empties with
 * the number inside it: a model being asked again, a message that can be sent
 * in a moment. It stops at one rather than reading zero while the next try
 * is under way.
 */
export function DeskCountdown({
  from,
  size = 14,
  tone = "warn",
}: {
  from: number;
  size?: number;
  tone?: "warn" | "ink";
}) {
  const [left, setLeft] = useState(from);
  useEffect(() => {
    setLeft(from);
    const timer = window.setInterval(() => setLeft((value) => Math.max(1, value - 1)), 1000);
    return () => window.clearInterval(timer);
  }, [from]);

  const r = size / 2 - 1.5;
  const c = 2 * Math.PI * r;
  return (
    <span
      className={cn("dk-ec-cd", `dk-t-${tone}`)}
      style={{ width: size, height: size }}
      aria-hidden
    >
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="currentColor"
          strokeOpacity=".22"
          strokeWidth="1.6"
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="currentColor"
          strokeWidth="1.6"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c * (1 - left / from)}
          transform={`rotate(-90 ${size / 2} ${size / 2})`}
          style={{ transition: "stroke-dashoffset 1s linear" }}
        />
      </svg>
      <b>{left}</b>
    </span>
  );
}
