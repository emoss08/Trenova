import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties, ReactNode } from "react";
import { Ic } from "../kit/ic";

type AnalyticsCardProps = {
  title: string;
  description?: ReactNode;
  right?: ReactNode;
  className?: string;
  children: ReactNode;
};

/** A chart's card: a title, what it measures, its controls at the right, then the chart. */
export function AnalyticsCard({
  title,
  description,
  right,
  className,
  children,
}: AnalyticsCardProps) {
  return (
    <section className={cn("an-card", className)} aria-label={title}>
      <header className="an-ch">
        <div className="an-ct">
          <h3>{title}</h3>
          {description && <p>{description}</p>}
        </div>
        {right && <div className="an-cr">{right}</div>}
      </header>
      {children}
    </section>
  );
}

/** Which way a metric improves. */
export function Better({ higher }: { higher: boolean }) {
  const t = useT();

  return (
    <span className="an-bt">
      <Ic n={higher ? "up" : "down"} s={10} w={2.4} />
      {higher ? t("Higher is better") : t("Lower is better")}
    </span>
  );
}

/** A series' colour as a small square. */
export function Swatch({ color }: { color: string | undefined }) {
  return <i className="an-sw" style={{ "--col": color } as CSSProperties} />;
}

type TipProps = {
  x: number;
  y: number;
  width?: number;
  /** The chart's width, to keep the tip inside it. */
  bound?: number;
  /** Open to the left of the point, for a point near the right edge. */
  flip?: boolean;
  /** Anchored beside the point rather than above it. */
  beside?: boolean;
  children: ReactNode;
};

/** A chart's hover card: inverted and opaque, like everything that floats from a trigger. */
export function Tip({
  x,
  y,
  width = 220,
  bound,
  flip = false,
  beside = false,
  children,
}: TipProps) {
  const left = bound ? Math.max(width / 2 + 4, Math.min(bound - width / 2 - 4, x)) : x;

  return (
    <div
      className={cn("an-tip dark", beside && "beside", flip && "flip")}
      style={{ left: beside ? x : left, top: y, "--tw": `${width}px` } as CSSProperties}
      role="presentation"
    >
      {children}
    </div>
  );
}
