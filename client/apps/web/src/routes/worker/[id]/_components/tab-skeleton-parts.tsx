import { cn } from "@trenova/shared/lib/utils";

export function ShimmerLine({ className }: { className?: string }) {
  return <span className={cn("block h-3 ui-shimmer rounded-sm", className)} />;
}

export function ShimmerBand({ figures = 4 }: { figures?: number }) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border p-4">
      <div className="flex items-center justify-between gap-3">
        <ShimmerLine className="h-5 w-48" />
        <ShimmerLine className="h-7 w-28 rounded-md" />
      </div>
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {Array.from({ length: figures }, (_, i) => (
          <div key={i} className="flex flex-col gap-2">
            <ShimmerLine className="w-20" />
            <ShimmerLine className="h-5 w-24" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function ShimmerPanel({ rows = 5, className }: { rows?: number; className?: string }) {
  return (
    <div className={cn("flex flex-col overflow-hidden rounded-lg border", className)}>
      <div className="flex h-9 items-center justify-between gap-2 border-b px-3">
        <ShimmerLine className="w-32" />
        <ShimmerLine className="h-5 w-16 rounded-md" />
      </div>
      <div className="flex flex-col divide-y">
        {Array.from({ length: rows }, (_, i) => (
          <div
            key={i}
            className="grid grid-cols-[minmax(0,1fr)_6rem_5rem] items-center gap-3 px-3 py-2.5"
          >
            <ShimmerLine className="w-3/5" />
            <ShimmerLine className="w-16" />
            <ShimmerLine className="h-4 w-14 rounded-sm" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function ShimmerMainAside({
  main = 2,
  aside = 2,
  narrow = false,
}: {
  main?: number;
  aside?: number;
  narrow?: boolean;
}) {
  return (
    <div
      className={cn(
        "grid items-start gap-4",
        narrow ? "xl:grid-cols-[minmax(0,1fr)_22rem]" : "xl:grid-cols-[minmax(0,1fr)_24rem]",
      )}
    >
      <div className="flex min-w-0 flex-col gap-4">
        {Array.from({ length: main }, (_, i) => (
          <ShimmerPanel key={i} rows={i === 0 ? 6 : 3} />
        ))}
      </div>
      <div className="flex min-w-0 flex-col gap-4">
        {Array.from({ length: aside }, (_, i) => (
          <ShimmerPanel key={i} rows={3} />
        ))}
      </div>
    </div>
  );
}

export function ShimmerTwoColumns() {
  return (
    <div className="grid items-start gap-4 lg:grid-cols-2">
      <ShimmerPanel rows={4} />
      <ShimmerPanel rows={4} />
    </div>
  );
}
