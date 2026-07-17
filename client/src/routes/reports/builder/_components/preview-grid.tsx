import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import type { ReportPreview } from "@/lib/graphql/reports";
import { cn } from "@/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { CircleAlertIcon, Table2Icon } from "lucide-react";
import { m } from "motion/react";
import { useRef } from "react";

type PreviewGridProps = {
  preview: ReportPreview | undefined;
  loading: boolean;
  error: string | null;
  ready: boolean;
};

const ROW_HEIGHT = 30;

function formatCell(value: unknown, type: string): string {
  if (value === null || value === undefined) return "";
  if (type === "epoch" && typeof value === "number") {
    return new Date(value * 1000).toLocaleString();
  }
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (typeof value === "string") return value;
  if (typeof value === "number") return String(value);
  return JSON.stringify(value);
}

function isNumericType(type: string): boolean {
  return type === "int" || type === "decimal";
}

function CenteredState({ children }: { children: React.ReactNode }) {
  return (
    <m.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, ease: "easeOut" }}
      className="flex h-full flex-col items-center justify-center gap-3 p-6"
    >
      {children}
    </m.div>
  );
}

export function PreviewGrid({ preview, loading, error, ready }: PreviewGridProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const rows = Array.isArray(preview?.rows) ? (preview.rows as unknown[][]) : [];

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 20,
  });

  if (!ready) {
    return (
      <CenteredState>
        <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
          <Table2Icon className="size-5 text-muted-foreground" strokeWidth={1.75} />
        </div>
        <div className="text-center">
          <p className="text-sm font-medium">Live preview</p>
          <p className="max-w-xs text-xs text-muted-foreground">
            Add fields from the catalog on the left — the preview updates as you build.
          </p>
        </div>
      </CenteredState>
    );
  }

  if (error) {
    return (
      <CenteredState>
        <div className="flex size-10 items-center justify-center rounded-lg bg-destructive/10">
          <CircleAlertIcon className="size-5 text-destructive" strokeWidth={1.75} />
        </div>
        <div className="text-center">
          <p className="text-sm font-medium">The preview couldn&apos;t be compiled</p>
          <p className="max-w-lg text-xs whitespace-pre-wrap text-muted-foreground">{error}</p>
        </div>
      </CenteredState>
    );
  }

  if (loading && !preview) {
    return (
      <div className="flex flex-col gap-1.5 p-4">
        {Array.from({ length: 10 }, (_, i) => (
          <Skeleton key={i} className="h-6" style={{ opacity: 1 - i * 0.09 }} />
        ))}
      </div>
    );
  }

  if (!preview) return null;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-8 items-center gap-2 border-b border-border px-3">
        <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          Preview
        </span>
        <span className="rounded-sm bg-muted px-1.5 py-px text-2xs text-muted-foreground tabular-nums">
          {rows.length} row{rows.length === 1 ? "" : "s"}
        </span>
        {preview.truncated && (
          <span className="rounded-sm bg-amber-500/10 px-1.5 py-px text-2xs text-amber-600 dark:text-amber-400">
            first 100 shown
          </span>
        )}
        <div className="flex-1" />
        {loading && <Spinner className="size-3.5 text-muted-foreground" />}
      </div>
      <div ref={scrollRef} className="min-h-0 flex-1 overflow-auto">
        <div className="min-w-fit">
          <div className="sticky top-0 z-10 flex border-b border-border bg-background/95 backdrop-blur-sm">
            {preview.columns.map((column) => (
              <div
                key={column.id}
                className={cn(
                  "w-44 shrink-0 truncate px-3 py-1.5 text-2xs font-medium tracking-wide text-muted-foreground uppercase",
                  isNumericType(column.type) && "text-right",
                )}
                title={column.label}
              >
                {column.label}
              </div>
            ))}
          </div>
          <div
            className={cn("relative transition-opacity", loading && "opacity-50")}
            style={{ height: virtualizer.getTotalSize() }}
          >
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const row = rows[virtualRow.index];
              return (
                <div
                  key={virtualRow.key}
                  className="absolute top-0 left-0 flex w-full border-b border-border/40 transition-colors hover:bg-accent/40"
                  style={{
                    height: virtualRow.size,
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  {preview.columns.map((column, columnIndex) => (
                    <div
                      key={column.id}
                      className={cn(
                        "w-44 shrink-0 truncate px-3 py-1.5 text-xs",
                        isNumericType(column.type)
                          ? "text-right font-mono tabular-nums"
                          : "text-foreground/90",
                      )}
                      title={formatCell(row?.[columnIndex], column.type)}
                    >
                      {formatCell(row?.[columnIndex], column.type)}
                    </div>
                  ))}
                </div>
              );
            })}
          </div>
          {rows.length === 0 && (
            <p className="p-8 text-center text-xs text-muted-foreground">
              The report compiled but returned no rows for the preview window.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
