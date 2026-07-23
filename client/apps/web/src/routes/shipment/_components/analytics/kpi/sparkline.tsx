import { useId } from "react";

type SparklineProps = {
  data: number[];
  color?: string;
  width?: number;
  height?: number;
  fill?: boolean;
};

function buildPath(data: number[], w: number, h: number, pad: number) {
  if (data.length === 0) return { line: "", area: "" };
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const step = data.length > 1 ? (w - pad * 2) / (data.length - 1) : 0;
  const pts = data.map<[number, number]>((value, index) => {
    const x = pad + index * step;
    const y = h - pad - ((value - min) / range) * (h - pad * 2);
    return [x, y];
  });
  const line = pts
    .map((point, index) => `${index === 0 ? "M" : "L"}${point[0].toFixed(1)},${point[1].toFixed(1)}`)
    .join(" ");
  const last = pts[pts.length - 1];
  const first = pts[0];
  const area = `${line} L ${last[0].toFixed(1)},${h} L ${first[0].toFixed(1)},${h} Z`;
  return { line, area };
}

export function Sparkline({
  data,
  color = "var(--brand)",
  width = 88,
  height = 24,
  fill = true,
}: SparklineProps) {
  const id = useId();
  const { line, area } = buildPath(data, width, height, 2);

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      className="shrink-0"
      aria-hidden
    >
      {fill && (
        <>
          <defs>
            <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity="0.3" />
              <stop offset="100%" stopColor={color} stopOpacity="0" />
            </linearGradient>
          </defs>
          <path d={area} fill={`url(#${id})`} />
        </>
      )}
      <path d={line} stroke={color} strokeWidth={1.25} fill="none" />
    </svg>
  );
}
