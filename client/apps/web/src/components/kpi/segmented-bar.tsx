export type Segment = {
  label: string;
  value: number;
  color: string;
};

type SegmentedBarProps = {
  segments: Segment[];
};

export function SegmentedBar({ segments }: SegmentedBarProps) {
  const total = segments.reduce((sum, segment) => sum + segment.value, 0) || 1;

  return (
    <div className="flex flex-col gap-1">
      <div className="bg-muted flex h-1.5 overflow-hidden rounded-sm">
        {segments.map((segment) => (
          <div
            key={segment.label}
            title={`${segment.label}: ${segment.value}`}
            style={{
              flex: segment.value / total,
              background: segment.color,
            }}
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-2.5 gap-y-0.5">
        {segments.map((segment) => (
          <span
            key={segment.label}
            className="text-muted-foreground inline-flex items-center gap-1 text-[9.5px] tracking-wide"
          >
            <span
              aria-hidden
              className="size-[5px] rounded-[1px]"
              style={{ background: segment.color }}
            />
            {segment.label}{" "}
            <span className="text-foreground/70 font-mono tabular-nums">{segment.value}</span>
          </span>
        ))}
      </div>
    </div>
  );
}
