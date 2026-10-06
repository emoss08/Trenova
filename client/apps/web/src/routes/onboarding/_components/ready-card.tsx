import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import type { CSSProperties } from "react";

const SPARK_COLORS = [
  "var(--dsk-gem-1)",
  "var(--brand)",
  "var(--dsk-gem-3)",
  "var(--warning)",
  "var(--success)",
];
const SPARK_COUNT = 22;

const SPARKS = Array.from({ length: SPARK_COUNT }, (_, index) => ({
  "--nv-a": `${(index * 360) / SPARK_COUNT + Math.random() * 10}deg`,
  "--nv-d": `${34 + Math.random() * 40}px`,
  background: SPARK_COLORS[index % SPARK_COLORS.length],
  animationDelay: `${120 + Math.random() * 80}ms`,
})) as CSSProperties[];

function Burst() {
  return (
    <span className="nv-burst" aria-hidden="true">
      {SPARKS.map((style, index) => (
        <i key={index} className="nv-spark" style={style} />
      ))}
    </span>
  );
}

/** The workspace exists: a check, a small burst, and the way into the app. */
export function ReadyCard({
  title,
  detail,
  opening,
  onOpen,
}: {
  title: string;
  detail: string;
  opening: boolean;
  onOpen: () => void;
}) {
  const t = useT();
  return (
    <div className="nv-rdy" role="status">
      <span className="nv-okr" aria-hidden="true">
        <svg
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path className="nv-draw" d="m3.5 8.5 3 3 6-7" />
        </svg>
      </span>
      <Burst />
      <span className="nv-rdy-t">
        <b>{title}</b>
        <span>{detail}</span>
      </span>
      <button type="button" className="nv-bt" data-ink="true" onClick={onOpen} disabled={opening}>
        {t("Open Trenova")}
        <ArrowRightIcon size={14} strokeWidth={1.7} aria-hidden="true" />
      </button>
    </div>
  );
}
