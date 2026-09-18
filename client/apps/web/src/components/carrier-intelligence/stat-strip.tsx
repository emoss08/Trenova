import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { StatusDot, type StatusTone } from "./status-dot";

export type StatStripItem = {
  id: string;
  label: ReactNode;
  value: ReactNode;
  hint?: ReactNode;
  tone?: StatusTone;
  onClick?: () => void;
};

export type StatStripProps = {
  items: StatStripItem[];
  className?: string;
};

export function StatStrip({ items, className }: StatStripProps) {
  return (
    <dl
      className={cn(
        "grid grid-cols-2 divide-border overflow-hidden rounded-lg border sm:grid-cols-[repeat(auto-fit,minmax(0,1fr))] sm:divide-x",
        className,
      )}
    >
      {items.map((item) => {
        const body = (
          <>
            <dt className="flex items-center gap-2 text-xs text-muted-foreground">
              {item.tone && <StatusDot tone={item.tone} />}
              {item.label}
            </dt>
            <dd className="mt-1 text-xl font-semibold tracking-tight tabular-nums">{item.value}</dd>
            {item.hint && (
              <dd className="mt-0.5 truncate text-xs text-muted-foreground">{item.hint}</dd>
            )}
          </>
        );
        return item.onClick ? (
          <button
            key={item.id}
            type="button"
            onClick={item.onClick}
 className="ui-focus-ring flex min-w-0 flex-col px-4 py-3 text-left transition-colors hover:bg-muted/50 focus-visible:bg-muted/50"
          >
            {body}
          </button>
        ) : (
          <div key={item.id} className="flex min-w-0 flex-col px-4 py-3">
            {body}
          </div>
        );
      })}
    </dl>
  );
}
