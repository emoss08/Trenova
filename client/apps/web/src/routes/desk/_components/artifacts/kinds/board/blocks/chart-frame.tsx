import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import type { ReactNode } from "react";
import { CHART_WIDTH } from "./chart-values";

/**
 * A chart's box: as wide as its exhibit and as tall as its form draws, with the SVG laid out at
 * that measured width (never scaled, so every text keeps its class's size). Until the box is
 * measured the marks are laid out at the conversation's width. `role="img"` and the label say
 * what the chart shows and its headline number.
 */
export function ChartFrame({
  height,
  label,
  children,
}: {
  height: number;
  label: string;
  children: (width: number) => ReactNode;
}) {
  const [ref, measured] = useElementWidth<HTMLDivElement>();
  const width = Math.round(measured > 0 ? measured : CHART_WIDTH);
  return (
    <div ref={ref} className="dk-ch-box" style={{ height }}>
      <svg
        className="dk-ch"
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        role="img"
        aria-label={label}
      >
        {children(width)}
      </svg>
    </div>
  );
}
