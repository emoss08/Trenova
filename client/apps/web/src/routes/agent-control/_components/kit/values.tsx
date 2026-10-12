import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type LineProps = {
  values: readonly number[];
  w?: number;
  h?: number;
  label?: string;
};

/** A bare trend line, scaled to its own range. A single value draws flat. */
export function Line({ values, w = 64, h = 18, label }: LineProps) {
  if (values.length === 0) {
    return null;
  }
  const high = Math.max(...values);
  const low = Math.min(...values);
  const range = high - low || 1;
  const step = values.length > 1 ? (w - 2) / (values.length - 1) : 0;
  const points = values
    .map((value, index) => `${index * step + 1},${h - 2 - ((value - low) / range) * (h - 4)}`)
    .join(" ");

  return (
    <svg
      className="ln"
      width={w}
      height={h}
      viewBox={`0 0 ${w} ${h}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      role={label ? "img" : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    >
      <polyline points={values.length > 1 ? points : `1,${h / 2} ${w - 1},${h / 2}`} />
    </svg>
  );
}

/** A change in points, signed and toned: up is good, down is bad. */
export function Pts({ value }: { value: number | null | undefined }) {
  const t = useT();
  if (value == null) {
    return <span className="dim">—</span>;
  }
  const rounded = Math.round(value);
  const sign = rounded > 0 ? "+" : rounded < 0 ? "−" : "±";

  return (
    <span className={cn("mono pts", rounded > 0 ? "t-k" : rounded < 0 ? "t-d" : "dim")}>
      {t("{0}{1} pts", sign, Math.abs(rounded))}
    </span>
  );
}

export type KVItem = readonly [label: string, value: ReactNode];

/** Label and value pairs in two columns, as a sheet's facts. */
export function KV({ items }: { items: readonly (KVItem | false | null | undefined)[] }) {
  return (
    <dl className="sfx">
      {items
        .filter((item): item is KVItem => Boolean(item))
        .map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
    </dl>
  );
}
